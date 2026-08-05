package cron

import (
	"context"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/apilog"
	"github.com/aaron/sakoo-backend/ent/user"
	"github.com/aaron/sakoo-backend/ent/usersession"
	"github.com/aaron/sakoo-backend/internal/infrastructure/scraper"
	"github.com/aaron/sakoo-backend/internal/usecase"
	"github.com/robfig/cron/v3"
)

type CronManager struct {
	bcvScraperUseCase   *usecase.ScraperUseCase
	exchangeRateUseCase *usecase.ExchangeRateUseCase
	client              *ent.Client
	cronInstance        *cron.Cron
}

type cronLogger struct{}

func (cronLogger) Info(msg string, keysAndValues ...interface{}) {
	slog.Info(msg, keysAndValues...)
}

func (cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	slog.Error(msg, append(keysAndValues, "error", err)...)
}

func NewCronManager(
	bcvScraperUseCase *usecase.ScraperUseCase,
	exchangeRateUseCase *usecase.ExchangeRateUseCase,
	client *ent.Client,
) *CronManager {
	logger := cronLogger{}

	c := cron.New(
		cron.WithLocation(time.UTC),
		cron.WithChain(
			cron.Recover(logger),
		),
	)

	return &CronManager{
		bcvScraperUseCase:   bcvScraperUseCase,
		exchangeRateUseCase: exchangeRateUseCase,
		client:              client,
		cronInstance:        c,
	}
}

func (cm *CronManager) Start(ctx context.Context) {
	slog.Info("Inicializando el planificador CronManager con Ent...")

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

	cronExprApprove := "*/30 * * * *"
	_, errApprove := cm.cronInstance.AddFunc(cronExprApprove, func() {
		slog.Info("Cron Triggered: Auto-aprobando tasas cuyo value_date ya llegó...")
		approveCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		affected, err := cm.exchangeRateUseCase.ApproveDueRates(approveCtx)
		if err != nil {
			slog.Error("Fallo en la ejecución automática del Cron de Auto-Aprobación de tasas", "error", err)
		} else {
			slog.Info("Ciclo automático de auto-aprobación de tasas completado", "tasas_aprobadas", affected)
		}
	})

	if errApprove != nil {
		slog.Error("Fallo crítico al registrar la tarea de Auto-Aprobación en CronManager", "expr", cronExprApprove, "error", errApprove)
		return
	}

	// Tarea de Limpieza Automática usando Ent
	cronExprCleanup := "0 */3 * * *"
	_, errCleanup := cm.cronInstance.AddFunc(cronExprCleanup, func() {
		slog.Info("Cron Triggered: Ejecutando limpieza automática periódica con Ent...")
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		cutoff24h := time.Now().Add(-24 * time.Hour)
		nLogs, err := cm.client.ApiLog.Delete().Where(apilog.CreatedAtLT(cutoff24h)).Exec(cleanupCtx)
		if err != nil {
			slog.Error("Fallo en la limpieza de logs antiguos en Ent", "error", err)
		} else {
			slog.Info("Limpieza periódica de logs completada", "filas_eliminadas", nLogs)
		}

		nSess, errSess := cm.client.UserSession.Delete().Where(usersession.ExpiresAtLT(time.Now())).Exec(cleanupCtx)
		if errSess != nil {
			slog.Error("Fallo en la limpieza de sesiones expiradas en Ent", "error", errSess)
		} else {
			slog.Info("Limpieza periódica de sesiones completada", "filas_eliminadas", nSess)
		}

		cutoff15d := time.Now().AddDate(0, 0, -15)
		nPurg, errPurge := cm.client.User.Delete().Where(user.DeletedAtLT(cutoff15d)).Exec(cleanupCtx)
		if errPurge != nil {
			slog.Error("Fallo en la purga de cuentas eliminadas en Ent", "error", errPurge)
		} else {
			slog.Info("Purga definitiva de cuentas eliminadas completada", "cuentas_purgadas", nPurg)
		}
	})

	if errCleanup != nil {
		slog.Error("Fallo crítico al registrar la tarea de Limpieza de Logs en CronManager", "expr", cronExprCleanup, "error", errCleanup)
		return
	}

	cronExprBinanceUSDT := "0 * * * *"
	_, errUSDT := cm.cronInstance.AddFunc(cronExprBinanceUSDT, func() {
		slog.Info("Cron Triggered: Iniciando Binance P2P Worker para USDT...")
		workerCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := scraper.RunBinanceWorker(workerCtx, cm.client, "USDT"); err != nil {
			slog.Error("Fallo en la ejecución del Binance P2P Worker para USDT", "error", err)
		} else {
			slog.Info("Ciclo automático de Binance P2P Worker para USDT completado con éxito")
		}
	})

	if errUSDT != nil {
		slog.Error("Fallo crítico al registrar la tarea de Binance P2P USDT", "expr", cronExprBinanceUSDT, "error", errUSDT)
		return
	}

	cronExprBinanceUSDC := "5 * * * *"
	_, errUSDC := cm.cronInstance.AddFunc(cronExprBinanceUSDC, func() {
		slog.Info("Cron Triggered: Iniciando Binance P2P Worker para USDC...")
		workerCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := scraper.RunBinanceWorker(workerCtx, cm.client, "USDC"); err != nil {
			slog.Error("Fallo en la ejecución del Binance P2P Worker para USDC", "error", err)
		} else {
			slog.Info("Ciclo automático de Binance P2P Worker para USDC completado con éxito")
		}
	})

	if errUSDC != nil {
		slog.Error("Fallo crítico al registrar la tarea de Binance P2P USDC", "expr", cronExprBinanceUSDC, "error", errUSDC)
		return
	}

	cm.cronInstance.Start()
	slog.Info("CronManager iniciado con éxito en segundo plano", "zona_horaria", "UTC")
}

func (cm *CronManager) Stop() {
	slog.Info("Deteniendo el planificador CronManager de forma ordenada...")
	ctx := cm.cronInstance.Stop()

	select {
	case <-ctx.Done():
		slog.Info("CronManager detenido correctamente sin tareas pendientes")
	case <-time.After(10 * time.Second):
		slog.Warn("CronManager forzado a detenerse.")
	}
}
