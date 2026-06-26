package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aaron/sakoo-backend/internal/domain"
)

// ScraperUseCase orquesta el proceso de raspado de tasas de cambio del BCV e inserción resiliente en BD.
type ScraperUseCase struct {
	scraperService      domain.ScraperService
	repo                domain.ExchangeRateRepository
	notificationUseCase domain.NotificationUseCase
}

// NewScraperUseCase crea una nueva instancia del caso de uso de Scraping.
func NewScraperUseCase(
	scraperService domain.ScraperService,
	repo domain.ExchangeRateRepository,
	notificationUseCase domain.NotificationUseCase,
) *ScraperUseCase {
	return &ScraperUseCase{
		scraperService:      scraperService,
		repo:                repo,
		notificationUseCase: notificationUseCase,
	}
}

// ExecuteScraping ejecuta el raspado de tasas de cambio y las persiste individualmente de manera resiliente.
func (uc *ScraperUseCase) ExecuteScraping(ctx context.Context) error {
	slog.Info("Iniciando ejecución del caso de uso de Scraping de tasas de cambio...")

	// 1. Obtener las tasas de cambio desde el servicio de scraping
	scrapedRates, err := uc.scraperService.ScrapeRates(ctx)
	if err != nil {
		slog.Error("Fallo crítico al realizar el scraping de las tasas del BCV", "error", err)
		return fmt.Errorf("error al obtener tasas desde el scraper: %w", err)
	}

	if len(scrapedRates) == 0 {
		slog.Warn("No se extrajo ninguna tasa de cambio del BCV en este ciclo")
		return nil
	}

	// 2. Cargar dinámicamente el catálogo de monedas desde la base de datos para mapear códigos a IDs
	currencyMap, err := uc.repo.GetCurrencyIDs(ctx)
	if err != nil {
		slog.Error("Fallo crítico al cargar el mapa de monedas desde la base de datos", "error", err)
		return fmt.Errorf("error al obtener mapa de monedas: %w", err)
	}

	// 3. Iterar sobre cada tasa obtenida de forma resiliente
	var totalSuccess int
	var totalFailures int
	var changedRates []domain.ExchangeRate

	for _, rate := range scrapedRates {
		// Validar que la moneda exista en el catálogo de base de datos
		id, exists := currencyMap[rate.CurrencyCode]
		if !exists {
			slog.Error("Resiliencia - La moneda no está registrada en el catálogo de base de datos", 
				"moneda", rate.CurrencyCode,
			)
			totalFailures++
			continue // Continuar con la siguiente moneda sin detener el proceso
		}

		// Asignar el ID correcto de base de datos a la entidad
		rate.CurrencyID = id

		slog.Info("Guardando tasa de cambio mapeada...", 
			"moneda", rate.CurrencyCode, 
			"currency_id", rate.CurrencyID,
			"tasa", rate.RateAverage.String(),
		)

		// Persistir de forma segura e idempotente (el Upsert asigna rate.ID).
		if err := uc.repo.Upsert(ctx, &rate); err != nil {
			slog.Error("Resiliencia - Error al guardar la tasa de cambio en la base de datos",
				"moneda", rate.CurrencyCode,
				"currency_id", rate.CurrencyID,
				"error", err,
			)
			totalFailures++
			continue
		}

		slog.Info("Resiliencia - Tasa de cambio guardada/actualizada con éxito",
			"moneda", rate.CurrencyCode,
			"currency_id", rate.CurrencyID,
		)
		totalSuccess++

		// --- Decisión de notificación push ---
		// Solo para monedas visibles relevantes (USD/EUR), solo cuando el VALOR (rate_to) cambia
		// respecto del día hábil anterior, y como máximo una vez por tasa.
		//
		// IMPORTANTE: NO se compara por value_date. El BCV publica por la tarde/noche la fecha
		// del próximo día hábil; esa fila futura quedaba siempre "distinta" de la última visible
		// (filtrada a <= hoy), disparando un push en cada ciclo del cron (cada 30 min, sobre todo
		// los fines de semana). Comparar rate_to vs el día previo + deduplicar elimina ese spam.
		if rate.CurrencyCode != "USD" && rate.CurrencyCode != "EUR" {
			continue
		}
		if uc.notificationUseCase == nil {
			continue
		}

		prev, errPrev := uc.repo.GetPreviousRate(ctx, rate.CurrencyCode, rate.ValueDate)
		valueChanged := false
		switch {
		case errPrev != nil && errors.Is(errPrev, domain.ErrNotFound):
			valueChanged = true // Primera tasa registrada para esta moneda.
		case errPrev != nil:
			slog.Warn("No se pudo obtener la tasa previa para evaluar el cambio; se omite la notificación",
				"moneda", rate.CurrencyCode, "error", errPrev)
			continue
		case prev != nil && !prev.RateTo.Equal(rate.RateTo):
			valueChanged = true
		}

		if !valueChanged {
			continue
		}

		// Reclamar la notificación de forma atómica: deduplica entre ciclos del cron.
		claimed, errMark := uc.repo.MarkRateNotified(ctx, rate.ID)
		if errMark != nil {
			slog.Warn("Fallo al reclamar la notificación de la tasa; se omite",
				"moneda", rate.CurrencyCode, "rate_id", rate.ID, "error", errMark)
			continue
		}
		if claimed {
			changedRates = append(changedRates, rate)
		}
	}

	// 4. Si hubo cambios, disparar la notificación push al Topic "exchange_rates"
	if len(changedRates) > 0 {
		var title, body string
		payload := map[string]interface{}{
			"type":   "rate_update",
			"source": "BCV",
		}

		if len(changedRates) == 1 {
			// Solo cambió una tasa (ej: USD)
			changed := changedRates[0]
			rateStr := changed.RateTo.Truncate(2).StringFixed(2)
			title = fmt.Sprintf("¡La tasa de %s ha cambiado! 🚀", changed.CurrencyCode)
			body = fmt.Sprintf("La nueva tasa oficial es de %s Bs.", rateStr)
			payload["currency_code"] = changed.CurrencyCode
			payload["rate"] = rateStr
		} else {
			// Cambiaron múltiples tasas del BCV
			title = "¡Las tasas del BCV han cambiado! 🚀"
			body = "El Banco Central de Venezuela actualizó múltiples tasas oficiales en la plataforma."
		}

		slog.Info("Disparando notificación push masiva del BCV por Topic...", "cantidad_cambios", len(changedRates))
		_ = uc.notificationUseCase.SendTopicNotification(ctx, "exchange_rates", title, body, payload)
	}

	slog.Info("Resumen del ciclo de Scraping completado", 
		"exitosos", totalSuccess, 
		"fallidos", totalFailures, 
		"total_procesados", len(scrapedRates),
	)

	return nil
}
