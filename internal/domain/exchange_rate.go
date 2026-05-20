package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// ExchangeRate representa la tasa de cambio de una moneda para una fecha específica.
type ExchangeRate struct {
	ID           int64
	CurrencyID   int64
	CurrencyCode string          `json:"currency_code,omitempty"` // Código auxiliar de moneda (ej: USD) no persistido directamente
	RateFrom     decimal.Decimal // Tasa de origen
	RateTo      decimal.Decimal // Tasa de destino
	RateAverage decimal.Decimal // Tasa promedio
	ValueDate   time.Time       // Fecha de la tasa (solo año, mes y día)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ExchangeRateRepository define las operaciones de persistencia de tasas de cambio.
type ExchangeRateRepository interface {
	Upsert(ctx context.Context, rate *ExchangeRate) error
	GetCurrencyIDs(ctx context.Context) (map[string]int64, error) // Obtiene mapa de código de moneda a ID de la BD
	GetLatestRates(ctx context.Context) ([]ExchangeRate, error)   // Obtiene la última tasa de cada moneda
	GetRatesHistoryPaginated(ctx context.Context, page, limit int, currencyCode string, startDate, endDate *time.Time) ([]ExchangeRate, int, error)
	GetLatestRate(ctx context.Context, currencyCode string) (*ExchangeRate, error)
	GetPreviousRate(ctx context.Context, currencyCode string, beforeDate time.Time) (*ExchangeRate, error)
	GetRateByDate(ctx context.Context, currencyCode string, date time.Time) (*ExchangeRate, error)
	GetRatesHistory(ctx context.Context, currencyCode string, limit int) ([]ExchangeRate, error)
	GetCalendarDates(ctx context.Context) ([]string, error)
}
