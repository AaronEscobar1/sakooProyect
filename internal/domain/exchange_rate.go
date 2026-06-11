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
	Status      string          // Estado (REGISTERED / APPROVED)
	Source      string          // Origen (SCRAPING / MANUAL)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ApproveExchangeRateRequest define el DTO de entrada para aprobar/modificar una tasa desde el BackOffice.
type ApproveExchangeRateRequest struct {
	RateID      int64           `json:"rate_id"`
	RateFrom    decimal.Decimal `json:"rate_from"`
	RateTo      decimal.Decimal `json:"rate_to"`
	RateAverage decimal.Decimal `json:"rate_average"`
	Source      string          `json:"source"`
}

// ExchangeRateRepository define las operaciones de persistencia de tasas de cambio.
type ExchangeRateRepository interface {
	Upsert(ctx context.Context, rate *ExchangeRate) error
	GetCurrencyIDs(ctx context.Context) (map[string]int64, error) // Obtiene mapa de código de moneda a ID de la BD
	GetLatestRates(ctx context.Context) ([]ExchangeRate, error)   // Obtiene la última tasa de cada moneda
	GetRatesHistoryPaginated(ctx context.Context, page, limit int, currencyCode string, startDate, endDate *time.Time) ([]ExchangeRate, int, error)
	GetLatestRate(ctx context.Context, currencyCode string) (*ExchangeRate, error)
	GetLatestRateBeforeOrAt(ctx context.Context, currencyCode string, date time.Time) (*ExchangeRate, error)
	GetPreviousRate(ctx context.Context, currencyCode string, beforeDate time.Time) (*ExchangeRate, error)
	GetRateByDate(ctx context.Context, currencyCode string, date time.Time) (*ExchangeRate, error)
	GetRatesHistory(ctx context.Context, currencyCode string, limit int) ([]ExchangeRate, error)
	GetRatesHistoryBeforeOrAt(ctx context.Context, currencyCode string, date time.Time, limit int) ([]ExchangeRate, error)
	GetCalendarDates(ctx context.Context) ([]string, error)
	UpdateRateApproval(ctx context.Context, rateID int64, rateFrom, rateTo, rateAverage decimal.Decimal, source string) error
}

// ExchangeRateUseCase define la lógica de negocio para interactuar con las tasas de cambio.
type ExchangeRateUseCase interface {
	GetLatestRates(ctx context.Context) ([]ExchangeRate, error)
	GetRatesHistory(ctx context.Context, page, limit int, currencyCode, startDateStr, endDateStr string) ([]ExchangeRate, int, error)
	GetCalendarDates(ctx context.Context) ([]string, error)
	ApproveRate(ctx context.Context, req ApproveExchangeRateRequest, adminUserID int64) error
}

