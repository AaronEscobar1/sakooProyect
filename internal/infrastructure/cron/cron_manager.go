package cron

import (
	"context"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/infrastructure/scraper"
	"github.com/aaron/sakoo-backend/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

// CronManager administra las tareas recurrentes en segundo plano utilizando robfig/cron/v3.
type CronManager struct {
	bcvScraperUseCase       *usecase.ScraperUseCase
	mercantilScraperUseCase *usecase.MercantilScraperUseCase
	db                      *pgxpool.Pool
	cronInstance            *cron.Cron
}

// cronLogger adapta los logs estructurados de robfig/cron hacia el logger moderno slog.
type cronLogger struct{}

func (cronLogger) Info(msg string, keysAndValues ...interface{}) {
	slog.Info(msg, keysAndValues...)
}

func (cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	slog.Error(msg, append(keysAndValues, "error", err)...)
}

// NewCronManager crea una nueva instancia de CronManager.
func NewCronManager(
	bcvScraperUseCase *usecase.ScraperUseCase, 
	mercantilScraperUseCase *usecase.MercantilScraperUseCase,
	db *pgxpool.Pool,
) *CronManager {
	logger := cronLogger{}

	// Crear el planificador configurando de forma explícita la zona horaria UTC
	// Incorporamos un middleware recuperador de pánicos para resiliencia absoluta
	c := cron.New(
		cron.WithLocation(time.UTC),
		cron.WithChain(
			cron.Recover(logger),
		),
	)

	return &CronManager{
		bcvScraperUseCase:       bcvScraperUseCase,
		mercantilScraperUseCase: mercantilScraperUseCase,
		db:                      db,
		cronInstance:            c,
	}
}

// Start registra las tareas programadas e inicia el planificador de manera no bloqueante.
func (cm *CronManager) Start(ctx context.Context) {
	slog.Info("Inicializando el planificador CronManager...")

	// 1. Cron del BCV (Vespertino/Nocturno): cada 30 minutos, entre 3:00 PM y 10:59 PM VET (19:00 a 02:59 UTC del día siguiente)
	cronExprBCV := "*/30 19-23,0-2 * * *"
	_, err := cm.cronInstance.AddFunc(cronExprBCV, func() {
		slog.Info("Cron Triggered: Iniciando ciclo automático de scraping de tasas del BCV (Vespertino/Nocturno)...")
		
		scrapeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := cm.bcvScraperUseCase.ExecuteScraping(scrapeCtx); err != nil {
			slog.Error("Fallo en la ejecución automática del Cron de Scraping BCV", "error", err)
		} else {
			slog.Info("Ciclo automático de scraping de tasas BCV ejecutado con éxito")
		}
	})

	if err != nil {
		slog.Error("Fallo crítico al registrar la tarea de Scraping BCV en CronManager", "expr", cronExprBCV, "error", err)
		return
	}

	// 1.1 Cron del BCV (Respaldo Matutino): cada hora, entre 8:00 AM y 11:59 AM VET (12:00 a 15:59 UTC), de lunes a viernes
	// Garantiza capturar las tasas si el BCV las publica sumamente tarde en la noche o si hubo fallas de red previas.
	cronExprBCVMorning := "0 12-15 * * 1-5"
	_, errMorning := cm.cronInstance.AddFunc(cronExprBCVMorning, func() {
		slog.Info("Cron Triggered: Iniciando ciclo de respaldo matutino de scraping de tasas del BCV...")
		
		scrapeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := cm.bcvScraperUseCase.ExecuteScraping(scrapeCtx); err != nil {
			slog.Error("Fallo en la ejecución automática del Cron de Respaldo Matutino BCV", "error", err)
		} else {
			slog.Info("Ciclo automático de respaldo matutino de scraping de tasas BCV ejecutado con éxito")
		}
	})

	if errMorning != nil {
		slog.Error("Fallo crítico al registrar la tarea de Respaldo Matutino BCV en CronManager", "expr", cronExprBCVMorning, "error", errMorning)
		return
	}

	// 2. Cron de Mercantil: cada 2 horas todos los días
	cronExprMercantil := "0 */2 * * *"
	_, errMercantil := cm.cronInstance.AddFunc(cronExprMercantil, func() {
		slog.Info("Cron Triggered: Iniciando ciclo automático de scraping de tasas del Mercantil Banco...")
		
		scrapeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := cm.mercantilScraperUseCase.ExecuteScraping(scrapeCtx); err != nil {
			slog.Error("Fallo en la ejecución automática del Cron de Scraping Mercantil", "error", err)
		} else {
			slog.Info("Ciclo automático de scraping de tasas Mercantil ejecutado con éxito")
		}
	})

	if errMercantil != nil {
		slog.Error("Fallo crítico al registrar la tarea de Scraping Mercantil en CronManager", "expr", cronExprMercantil, "error", errMercantil)
		return
	}

	// 3. Tarea de Limpieza Automática de Logs: cada 3 horas (prune de logs mayores a 24 horas para evitar bloat)
	cronExprCleanup := "0 */3 * * *"
	_, errCleanup := cm.cronInstance.AddFunc(cronExprCleanup, func() {
		slog.Info("Cron Triggered: Ejecutando limpieza automática periódica de logs de auditoría antiguos...")
		
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		query := `DELETE FROM api_logs WHERE created_at < NOW() - INTERVAL '24 hours';`
		result, err := cm.db.Exec(cleanupCtx, query)
		if err != nil {
			slog.Error("Fallo en la ejecución de la tarea automática de limpieza de logs antiguos", "error", err)
		} else {
			slog.Info("Limpieza periódica de logs de auditoría completada con éxito", "filas_eliminadas", result.RowsAffected())
		}

		// Limpiar las sesiones expiradas para evitar el crecimiento innecesario de la tabla user_sessions
		sessionQuery := `DELETE FROM user_sessions WHERE expires_at < NOW();`
		sessResult, errSess := cm.db.Exec(cleanupCtx, sessionQuery)
		if errSess != nil {
			slog.Error("Fallo en la ejecución de la tarea automática de limpieza de sesiones expiradas", "error", errSess)
		} else {
			slog.Info("Limpieza periódica de sesiones expiradas completada con éxito", "filas_eliminadas", sessResult.RowsAffected())
		}

		// Purga definitiva de cuentas eliminadas: tras el periodo de gracia de 15 días desde la
		// solicitud de borrado, se elimina físicamente al usuario. El DELETE dispara los
		// ON DELETE CASCADE (cuentas bancarias, terceros, tokens FCM, sesiones, historial de
		// contraseñas, notificaciones) y anonimiza vía ON DELETE SET NULL (comentarios, mensajes,
		// compromisos). El borrado es irreversible.
		purgeQuery := `DELETE FROM users WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - INTERVAL '15 days';`
		purgeResult, errPurge := cm.db.Exec(cleanupCtx, purgeQuery)
		if errPurge != nil {
			slog.Error("Fallo en la purga definitiva de cuentas eliminadas tras el periodo de gracia", "error", errPurge)
		} else {
			slog.Info("Purga definitiva de cuentas eliminadas completada con éxito", "cuentas_purgadas", purgeResult.RowsAffected())
		}
	})

	if errCleanup != nil {
		slog.Error("Fallo crítico al registrar la tarea de Limpieza de Logs en CronManager", "expr", cronExprCleanup, "error", errCleanup)
		return
	}

	// 4. Cron de Binance P2P USDT: cada 1 hora todos los días (en el minuto 0)
	cronExprBinanceUSDT := "0 * * * *"
	_, errUSDT := cm.cronInstance.AddFunc(cronExprBinanceUSDT, func() {
		slog.Info("Cron Triggered: Iniciando ciclo automático de Binance P2P Worker para USDT...")
		
		workerCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := scraper.RunBinanceWorker(workerCtx, cm.db, "USDT"); err != nil {
			slog.Error("Fallo en la ejecución del Binance P2P Worker para USDT", "error", err)
		} else {
			slog.Info("Ciclo automático de Binance P2P Worker para USDT completado con éxito")
		}
	})

	if errUSDT != nil {
		slog.Error("Fallo crítico al registrar la tarea de Binance P2P USDT en CronManager", "expr", cronExprBinanceUSDT, "error", errUSDT)
		return
	}

	// 5. Cron de Binance P2P USDC: cada 1 hora todos los días (en el minuto 5)
	cronExprBinanceUSDC := "5 * * * *"
	_, errUSDC := cm.cronInstance.AddFunc(cronExprBinanceUSDC, func() {
		slog.Info("Cron Triggered: Iniciando ciclo automático de Binance P2P Worker para USDC...")
		
		workerCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := scraper.RunBinanceWorker(workerCtx, cm.db, "USDC"); err != nil {
			slog.Error("Fallo en la ejecución del Binance P2P Worker para USDC", "error", err)
		} else {
			slog.Info("Ciclo automático de Binance P2P Worker para USDC completado con éxito")
		}
	})

	if errUSDC != nil {
		slog.Error("Fallo crítico al registrar la tarea de Binance P2P USDC en CronManager", "expr", cronExprBinanceUSDC, "error", errUSDC)
		return
	}

	// Iniciar el cron en segundo plano
	cm.cronInstance.Start()
	slog.Info("CronManager iniciado con éxito en segundo plano", "zona_horaria", "UTC")
}

// Stop detiene el planificador de manera ordenada (Graceful Shutdown) esperando que terminen las tareas activas.
func (cm *CronManager) Stop() {
	slog.Info("Deteniendo el planificador CronManager de forma ordenada...")
	ctx := cm.cronInstance.Stop()
	
	// Bloquear hasta que las tareas terminen o se agote el tiempo de cortesía de 10 segundos
	select {
	case <-ctx.Done():
		slog.Info("CronManager detenido correctamente sin tareas pendientes")
	case <-time.After(10 * time.Second):
		slog.Warn("CronManager forzado a detenerse. Algunas tareas en ejecución podrían haber sido interrumpidas.")
	}
}
