package domain

import "context"

// ScraperService define el contrato para el servicio de raspado de tasas de cambio.
type ScraperService interface {
	ScrapeRates(ctx context.Context) ([]ExchangeRate, error)
}
