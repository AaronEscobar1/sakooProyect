package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/aaron/sakoo-backend/internal/infrastructure/scraper"
)

// MercantilScraperUseCase orquesta el proceso de raspado de la tasa "Dólar Intervención" (UDI) del Mercantil.
type MercantilScraperUseCase struct {
	scraperService scraper.ScraperService
	repo           domain.ExchangeRateRepository
}

// NewMercantilScraperUseCase crea una nueva instancia del caso de uso de Scraping para el Mercantil Banco.
func NewMercantilScraperUseCase(scraperService scraper.ScraperService, repo domain.ExchangeRateRepository) *MercantilScraperUseCase {
	return &MercantilScraperUseCase{
		scraperService: scraperService,
		repo:           repo,
	}
}

// ExecuteScraping ejecuta el raspado de Mercantil Banco y lo persiste en PostgreSQL.
func (uc *MercantilScraperUseCase) ExecuteScraping(ctx context.Context) error {
	slog.Info("Iniciando ejecución del caso de uso de Scraping de Mercantil Banco (UDI)...")

	// 1. Obtener la tasa desde el servicio
	scrapedRates, err := uc.scraperService.ScrapeRates(ctx)
	if err != nil {
		slog.Error("Fallo crítico al realizar el scraping de la tasa de Mercantil", "error", err)
		return fmt.Errorf("error al obtener tasa de Mercantil: %w", err)
	}

	if len(scrapedRates) == 0 {
		slog.Warn("No se extrajo ninguna tasa de cambio del Mercantil Banco en este ciclo")
		return nil
	}

	// 2. Cargar catálogo de monedas dinámicamente
	currencyMap, err := uc.repo.GetCurrencyIDs(ctx)
	if err != nil {
		slog.Error("Fallo crítico al cargar el mapa de monedas desde la base de datos (Mercantil)", "error", err)
		return fmt.Errorf("error al obtener mapa de monedas: %w", err)
	}

	rate := scrapedRates[0]
	id, exists := currencyMap[rate.CurrencyCode]
	if !exists {
		slog.Error("Resiliencia - La moneda no está registrada en el catálogo de base de datos", "moneda", rate.CurrencyCode)
		return fmt.Errorf("moneda %s no encontrada en base de datos. Asegúrese de correr la migración SQL.", rate.CurrencyCode)
	}

	rate.CurrencyID = id

	slog.Info("Guardando tasa de cambio mapeada (Mercantil)...",
		"moneda", rate.CurrencyCode,
		"currency_id", rate.CurrencyID,
		"tasa", rate.RateAverage.String(),
	)

	// 3. Persistir de forma segura e idempotente
	if err := uc.repo.Upsert(ctx, &rate); err != nil {
		slog.Error("Resiliencia - Error al guardar la tasa de cambio de Mercantil en la base de datos",
			"moneda", rate.CurrencyCode,
			"currency_id", rate.CurrencyID,
			"error", err,
		)
		return err
	}

	slog.Info("Resumen del ciclo de Scraping Mercantil completado exitosamente")
	return nil
}
