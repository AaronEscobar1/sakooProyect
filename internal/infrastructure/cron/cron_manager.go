package cron

import (
	"context"
	"log/slog"
	"time"

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

	// 1. Cron del BCV: cada 30 minutos, entre 7:00 PM y 10:59 PM UTC, de lunes a viernes
	cronExprBCV := "*/30 19-22 * * 1-5"
	_, err := cm.cronInstance.AddFunc(cronExprBCV, func() {
		slog.Info("Cron Triggered: Iniciando ciclo automático de scraping de tasas del BCV...")
		
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
	})

	if errCleanup != nil {
		slog.Error("Fallo crítico al registrar la tarea de Limpieza de Logs en CronManager", "expr", cronExprCleanup, "error", errCleanup)
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
