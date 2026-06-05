package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aaron/sakoo-backend/internal/api"
	"github.com/AaronEscobar1/common/middleware"
	"github.com/aaron/sakoo-backend/internal/infrastructure/cron"
	"github.com/AaronEscobar1/common/database"
	"github.com/aaron/sakoo-backend/internal/infrastructure/email"
	"github.com/aaron/sakoo-backend/internal/infrastructure/notification"
	"github.com/aaron/sakoo-backend/internal/infrastructure/repository"
	"github.com/aaron/sakoo-backend/internal/infrastructure/scraper"
	"github.com/AaronEscobar1/common/security"
	"github.com/aaron/sakoo-backend/internal/usecase"
	"github.com/joho/godotenv"

	docs "github.com/aaron/sakoo-backend/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Sakoo Backend API
// @version         1.0
// @description     Servidor API para el Proyecto Sakoo (Arquitectura Limpia).
// @termsOfService  http://swagger.io/terms/

// @contact.name   Soporte Sakoo
// @contact.url    http://www.sakoo.com
// @contact.email  soporte@sakoo.com

// @license.name  Propietaria
// @license.url   http://www.sakoo.com

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey ApiKeyAuth
// @in                         header
// @name                       Authorization
// @description                Coloque el token JWT de la siguiente manera: "Bearer <token>"
func main() {
	// 1. Inicializar logger estructurado slog en formato JSON para trazabilidad avanzada
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Permite visualizar logs detallados en desarrollo
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Iniciando el backend del Proyecto Sakoo (Clean Architecture Server)...")

	// Cargar variables de entorno desde el archivo .env si existe
	if err := godotenv.Load(); err != nil {
		slog.Warn("No se pudo cargar el archivo .env o no existe. Usando variables de entorno del sistema.")
	} else {
		slog.Info("Archivo .env cargado exitosamente")
	}

	// El host de Swagger se resuelve dinámicamente en cada request desde las cabeceras
	// HTTP del propio servidor (Host, X-Forwarded-Host, X-Forwarded-Proto).
	// Cloudflared inyecta automáticamente estas cabeceras → nunca hay que configurar nada.
	docs.SwaggerInfo.Schemes = []string{"https", "http"}
	slog.Info("Swagger configurado en modo host-dinámico (lee Host del request en tiempo real)")

	// 2. Inicializar llaves RSA de tránsito en memoria de manera segura
	if err := security.InitRSAKeys(); err != nil {
		slog.Error("Fallo crítico al inicializar criptografía RSA", "error", err)
		os.Exit(1)
	}

	// 3. Leer la variable de entorno de base de datos
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Fallback amigable para pruebas rápidas de desarrollo local
		dbURL = "postgres://postgres:postgres@localhost:5432/sakoo?sslmode=disable"
		slog.Warn("La variable de entorno DATABASE_URL no está definida. Utilizando dirección de desarrollo por defecto", 
			"fallback_url", "postgres://postgres:xxxxx@localhost:5432/sakoo?sslmode=disable",
		)
	}

	// 4. Conectar a PostgreSQL y ejecutar las migraciones automáticamente
	searchPaths := []string{"security", "market", "finance", "notifications", "catalogs", "telemetry", "public"}
	pool, err := database.ConnectAndMigrate(dbURL, searchPaths, "file://migrations")
	if (err == nil) {
		// Paso Autocurativo: Aplicar visibilidad de monedas al front
		ctxAuto, cancelAuto := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelAuto()
		setupQueries := []string{
			`ALTER TABLE catalogs.currency ADD COLUMN IF NOT EXISTS "show" BOOLEAN DEFAULT TRUE;`,
			`UPDATE catalogs.currency SET "show" = FALSE WHERE code NOT IN ('USD', 'EUR', 'USDT', 'USDC', 'UDI');`,
			`UPDATE catalogs.currency SET "show" = TRUE WHERE code IN ('USD', 'EUR', 'USDT', 'USDC', 'UDI');`,
			`INSERT INTO telemetry.configurations (key, payload) VALUES ('visible_currencies', '["USD", "EUR", "USDT", "USDC", "UDI"]'::jsonb) ON CONFLICT (key) DO UPDATE SET payload = EXCLUDED.payload;`,
		}
		for _, q := range setupQueries {
			if _, err := pool.Exec(ctxAuto, q); err != nil {
				slog.Warn("No se pudo ejecutar query autocurativo de visibilidad en base de datos", "query", q, "error", err)
			}
		}
	}
	if err != nil {
		// Imprimir en texto plano con alta visibilidad para los logs de Railway
		os.Stderr.WriteString("\n==================================================\n")
		os.Stderr.WriteString("❌ ERROR CRÍTICO EN MIGRACIONES/BASE DE DATOS:\n")
		os.Stderr.WriteString(err.Error() + "\n")
		os.Stderr.WriteString("==================================================\n\n")

		slog.Error("Fallo crítico en la inicialización de la base de datos/migraciones", "error", err.Error())
		os.Exit(1)
	}
	// Garantizar el cierre seguro del pool al terminar la aplicación
	defer func() {
		slog.Info("Cerrando el pool de conexiones de base de datos...")
		pool.Close()
		slog.Info("Pool de conexiones cerrado con éxito")
	}()

	// 5. Leer clave secreta para firmar tokens JWT y la API Key administrativa
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super-secret-key-change-in-production-12345!"
		slog.Warn("La variable de entorno JWT_SECRET no está definida. Utilizando clave de desarrollo por defecto.")
	}

	adminApiKey := os.Getenv("ADMIN_API_KEY")
	if adminApiKey == "" {
		adminApiKey = "admin-secret-api-key-12345!"
		slog.Warn("La variable de entorno ADMIN_API_KEY no está definida. Utilizando API Key de desarrollo por defecto.")
	}

	// 6. Instanciar los repositorios core de la capa de infraestructura
	userRepo := repository.NewUserRepository(pool)
	middleware.SetSessionValidator(userRepo)
	exchangeRateRepo := repository.NewExchangeRateRepository(pool)
	otpRepo := repository.NewOTPRepository(pool)
	emailSrv := email.NewEmailService()
	bankAccountRepo := repository.NewBankAccountRepository(pool)
	paymentCommitmentRepo := repository.NewPaymentCommitmentRepository(pool)
	messageRepo := repository.NewMessageRepository(pool)
	commentRepo := repository.NewCommentRepository(pool)
	bannerRepo := repository.NewBannerRepository(pool)
	catalogRepo := repository.NewCatalogRepository(pool)
	notificationRepo := repository.NewNotificationRepository(pool)
	telemetryRepo := repository.NewTelemetryRepository(pool)

	// Instanciar servicios de notificaciones push globales
	pushService := notification.NewPushNotificationService()
	notificationUseCase := usecase.NewNotificationUseCase(notificationRepo, pushService)

	// 7. Instanciar casos de uso de la capa de dominio
	authUseCase := usecase.NewAuthUseCase(userRepo, otpRepo, emailSrv, jwtSecret)
	
	// Instanciar servicios de Scraping y Cron
	bcvScraperService := scraper.NewBCVScraper()
	bcvScraperUseCase := usecase.NewScraperUseCase(bcvScraperService, exchangeRateRepo, notificationUseCase)
	
	// Instanciar servicios de Scraping Mercantil
	mercantilScraperService := scraper.NewMercantilScraper()
	mercantilScraperUseCase := usecase.NewMercantilScraperUseCase(mercantilScraperService, exchangeRateRepo, notificationUseCase)
	
	cronManager := cron.NewCronManager(bcvScraperUseCase, mercantilScraperUseCase, pool)

	exchangeRateUseCase := usecase.NewExchangeRateUseCase(exchangeRateRepo)
	dashboardUseCase := usecase.NewDashboardUseCase(exchangeRateRepo)
	calculatorUseCase := usecase.NewCalculatorUseCase(exchangeRateRepo)
	bankAccountUseCase := usecase.NewBankAccountUseCase(bankAccountRepo)
	paymentCommitmentUseCase := usecase.NewPaymentCommitmentUseCase(paymentCommitmentRepo)
	messageUseCase := usecase.NewMessageUseCase(messageRepo)
	commentUseCase := usecase.NewCommentUseCase(commentRepo)
	bannerUseCase := usecase.NewBannerUseCase(bannerRepo)
	catalogUseCase := usecase.NewCatalogUseCase(catalogRepo)
	telemetryUseCase := usecase.NewTelemetryUseCase(telemetryRepo)

	// 8. Instanciar controladores HTTP de la capa API
	authHandler := api.NewAuthHandler(authUseCase)
	scraperHandler := api.NewScraperHandler(bcvScraperUseCase, mercantilScraperUseCase, pool)
	exchangeRateHandler := api.NewExchangeRateHandler(exchangeRateUseCase)
	ratesHandler := api.NewRatesHandler(dashboardUseCase, calculatorUseCase, exchangeRateUseCase)
	bankAccountHandler := api.NewBankAccountHandler(bankAccountUseCase)
	paymentCommitmentHandler := api.NewPaymentCommitmentHandler(paymentCommitmentUseCase)
	messageHandler := api.NewMessageHandler(messageUseCase)
	commentHandler := api.NewCommentHandler(commentUseCase)
	bannerHandler := api.NewBannerHandler(bannerUseCase)
	catalogHandler := api.NewCatalogHandler(catalogUseCase)
	notificationHandler := api.NewNotificationHandler(notificationUseCase)
	adminHandler := api.NewAdminHandler(telemetryUseCase)

	slog.Info("Capa de persistencia, casos de uso y controladores HTTP instanciados de manera limpia.")

	// 9. Configurar enrutamiento HTTP usando las ventajas nativas de Go 1.22+
	mux := http.NewServeMux()

	// NOTA: Los preflights OPTIONS son interceptados por el middleware CORS *antes* de llegar
	// al router, por lo que NO se necesita un handler OPTIONS aquí. Agregarlo dentro del mux
	// causaría que el router respondiera sin los headers CORS (race condition de middlewares).

	// Ruta de Swagger UI — host dinámico resuelto desde el request
	// Cloudflared inyecta X-Forwarded-Proto y X-Forwarded-Host automáticamente.
	// En local, el Host header vale "localhost:8080". En tunnel, vale la URL de cloudflare.
	// Nunca hay que cambiar .env ni reiniciar el servidor cuando cambia la URL del túnel.
	mux.Handle("GET /swagger/{any...}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Determinar el scheme real: https si viene por tunnel, http en local
		scheme := r.Header.Get("X-Forwarded-Proto")
		if scheme == "" {
			if r.TLS != nil {
				scheme = "https"
			} else {
				scheme = "http"
			}
		}

		// Determinar el host real: cloudflared pone X-Forwarded-Host; local usa Host
		host := r.Header.Get("X-Forwarded-Host")
		if host == "" {
			host = r.Host
		}

		// Actualizar el spec de Swagger en tiempo real con el host y scheme del request actual
		docs.SwaggerInfo.Host = host
		docs.SwaggerInfo.Schemes = []string{scheme}

		// Servir la UI de Swagger con la config ya actualizada
		httpSwagger.WrapHandler(w, r)
	}))

	// Endpoint de Salud (Healthcheck) para Monitoreo de Railway
	// @Summary      Verificar el estado de salud del backend
	// @Description  Retorna el estado de operatividad del servidor de Sakoo y su conexión con la base de datos PostgreSQL.
	// @Tags         Monitoreo
	// @Produce      json
	// @Success      200   {object}  map[string]string  "Servidor operativo y conectado"
	// @Failure      500   {object}  map[string]string  "Servidor con fallos o base de datos desconectada"
	// @Router       /health [get]
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")
		if err := pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"status":"DOWN","database":"DISCONNECTED","error":"` + err.Error() + `"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"UP","database":"CONNECTED"}`))
	})

	// Rutas Públicas de Autenticación
	mux.HandleFunc("GET /api/auth/public-key", authHandler.HandlePublicKey)
	mux.HandleFunc("POST /api/auth/encrypt", authHandler.HandleEncryptString)
	mux.HandleFunc("POST /api/auth/register", authHandler.HandleRegister)
	mux.HandleFunc("POST /api/auth/login", authHandler.HandleLogin)

	// Rutas de Autenticación v1 (OTP centralizado)
	mux.HandleFunc("POST /api/v1/auth/otp/request", authHandler.HandleRequestOTP)
	mux.HandleFunc("POST /api/v1/auth/otp/validate", authHandler.HandleValidateOTP)
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.HandleRegister)
	mux.HandleFunc("POST /api/v1/auth/password/reset", authHandler.HandleResetPassword)
	mux.Handle("DELETE /api/v1/account", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(authHandler.HandleDeleteAccountV1)))
	mux.Handle("POST /api/v1/auth/logout", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(authHandler.HandleLogout)))

	// Ruta Pública de Tasas de Cambio
	mux.HandleFunc("POST /api/rates", exchangeRateHandler.HandleGetLatestRates)
	mux.HandleFunc("POST /api/rates/history", exchangeRateHandler.HandleGetRatesHistory)

	// Rutas del Core Business v1 (Dashboard y Calculadora)
	mux.HandleFunc("GET /api/v1/rates/dashboard", ratesHandler.HandleGetDashboardSummary)
	mux.HandleFunc("POST /api/v1/rates/calculate", ratesHandler.HandleCalculateConversion)
	mux.HandleFunc("GET /api/v1/rates/calendar-dates", ratesHandler.HandleGetCalendarDates)

	// Rutas de Mensajería Interna (Protegidas)
	mux.Handle("POST /api/v1/messages/send", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(messageHandler.HandleSendMessage)))
	mux.Handle("GET /api/v1/messages", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(messageHandler.HandleGetMessages)))
	mux.Handle("GET /api/v1/messages/unread-count", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(messageHandler.HandleGetUnreadCount)))

	// Rutas de Comentarios de Tasas
	mux.Handle("POST /api/v1/rates/comments", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(commentHandler.HandleAddComment)))
	mux.HandleFunc("GET /api/v1/rates/{rate_id}/comments", commentHandler.HandleGetRateComments)
	mux.HandleFunc("GET /api/v1/rates/comments", commentHandler.HandleGetRateComments)

	// Ruta de Banners (Pública)
	mux.HandleFunc("GET /api/v1/banners", bannerHandler.HandleGetBanners)

	// Rutas de Catálogos (Públicas)
	mux.HandleFunc("GET /api/v1/catalogs/document-types", catalogHandler.HandleGetDocumentTypes)
	mux.HandleFunc("GET /api/v1/catalogs/currencies", catalogHandler.HandleGetCurrencies)
	mux.HandleFunc("GET /api/v1/catalogs/banks", catalogHandler.HandleGetBanks)

	// Ruta de Scraping Manual (Pruebas en caliente) - Protegidas con API Key
	mux.Handle("POST /api/admin/scrape-now", middleware.AdminApiKeyMiddleware(adminApiKey)(http.HandlerFunc(scraperHandler.HandleScrapeNow)))
	mux.Handle("POST /api/admin/scrape-mercantil", middleware.AdminApiKeyMiddleware(adminApiKey)(http.HandlerFunc(scraperHandler.HandleScrapeMercantilNow)))
	mux.Handle("POST /api/admin/scrape-binance", middleware.AdminApiKeyMiddleware(adminApiKey)(http.HandlerFunc(scraperHandler.HandleScrapeBinance)))
	// Ruta de Consulta de Logs de Auditoría (Admin) - Protegida con API Key
	mux.Handle("GET /api/admin/logs", middleware.AdminApiKeyMiddleware(adminApiKey)(http.HandlerFunc(adminHandler.HandleGetAuditLogs)))

	// Ruta Protegida: Endpoint para obtener el perfil completo del usuario autenticado
	mux.Handle("GET /api/v1/me", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(authHandler.HandleGetProfile)))
	// Ruta Protegida: Buscar otros usuarios en la plataforma de manera liviana
	mux.Handle("GET /api/v1/users/search", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(authHandler.HandleSearchUsers)))

	// Rutas Protegidas de Cuentas Bancarias (Propias)
	mux.Handle("POST /api/v1/accounts/own", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(bankAccountHandler.HandleOwnAccounts)))
	mux.Handle("GET /api/v1/accounts/own", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(bankAccountHandler.HandleOwnAccounts)))
	mux.Handle("PUT /api/v1/accounts/own/{id}", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(bankAccountHandler.HandleOwnAccountDetail)))
	mux.Handle("DELETE /api/v1/accounts/own/{id}", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(bankAccountHandler.HandleOwnAccountDetail)))

	// Rutas Protegidas de Cuentas Bancarias (Terceros)
	mux.Handle("POST /api/v1/accounts/third-party", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(bankAccountHandler.HandleThirdPartyAccounts)))
	mux.Handle("GET /api/v1/accounts/third-party", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(bankAccountHandler.HandleThirdPartyAccounts)))
	mux.Handle("PUT /api/v1/accounts/third-party/{id}", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(bankAccountHandler.HandleThirdPartyAccountDetail)))
	mux.Handle("DELETE /api/v1/accounts/third-party/{id}", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(bankAccountHandler.HandleThirdPartyAccountDetail)))

	// Rutas Protegidas de Compromisos de Pago
	mux.Handle("POST /api/v1/payments/commitments", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(paymentCommitmentHandler.HandleCommitments)))
	mux.Handle("GET /api/v1/payments/commitments", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(paymentCommitmentHandler.HandleCommitments)))
	mux.Handle("PUT /api/v1/payments/commitments/{id}", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(paymentCommitmentHandler.HandleCommitmentDetail)))
	mux.Handle("DELETE /api/v1/payments/commitments/{id}", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(paymentCommitmentHandler.HandleCommitmentDetail)))

	// Rutas Protegidas de Notificaciones Push (Dispositivos e Inbox)
	mux.Handle("POST /api/v1/devices/register", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(notificationHandler.HandleRegisterDevice)))
	mux.Handle("POST /api/v1/devices/unregister", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(notificationHandler.HandleUnregisterDevice)))
	mux.Handle("GET /api/v1/notifications", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(notificationHandler.HandleGetNotifications)))
	mux.Handle("PUT /api/v1/notifications/{id}/read", middleware.AuthMiddleware(jwtSecret)(http.HandlerFunc(notificationHandler.HandleMarkAsRead)))

	// Ruta de Administración de Notificaciones (BackOffice) - Protegidas con API Key
	mux.Handle("POST /api/admin/notifications/send", middleware.AdminApiKeyMiddleware(adminApiKey)(http.HandlerFunc(notificationHandler.HandleSendAdminNotification)))
	mux.Handle("POST /api/admin/notifications/test", middleware.AdminApiKeyMiddleware(adminApiKey)(http.HandlerFunc(notificationHandler.HandleTestPushNotification)))

	// 10. Aplicar el Middleware de Trazabilidad y Logs asíncronos de forma global
	globalHandler := middleware.TraceAndLogMiddleware(pool)(mux)

	// Habilitar CORS para depuración local (Flutter Web, Swagger, etc.)
	corsHandler := middleware.CORS()(globalHandler)

	// 11. Configuración detallada del servidor HTTP
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      corsHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 11. Iniciar el planificador de Cron Jobs en segundo plano
	cronManager.Start(context.Background())

	// Iniciar el servidor HTTP de forma asíncrona
	go func() {
		slog.Info("Servidor HTTP iniciado y escuchando peticiones", "puerto", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Fallo crítico en el servidor HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// 12. Soporte para Apagado Ordenado (Graceful Shutdown)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Bloquear aquí hasta recibir una señal del sistema operativo
	sig := <-stopChan
	slog.Info("Señal de apagado detectada", "señal", sig.String())

	// Timeout de cortesía para el apagado ordenado
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Detener el gestor de Cron de forma ordenada esperando tareas activas
	cronManager.Stop()

	// Detener el servidor HTTP aceptando conexiones pendientes
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Error durante el apagado ordenado del servidor HTTP", "error", err)
	} else {
		slog.Info("Servidor HTTP apagado con éxito de forma limpia y ordenada.")
	}

	slog.Info("Proceso del backend finalizado de forma limpia.")
}
// Trigger Air rebuild

