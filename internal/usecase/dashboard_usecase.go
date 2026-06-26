package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// DashboardResponseDTO agrupa los datos consolidados requeridos por el Dashboard.
type DashboardResponseDTO struct {
	VariationPercent decimal.Decimal       `json:"variation_percent"`
	History          []domain.ExchangeRate `json:"history"`
}

// DashboardUseCase define el caso de uso para el Dashboard del MVP.
type DashboardUseCase interface {
	GetDashboardSummary(ctx context.Context, currencyCode string, date *time.Time) (*DashboardResponseDTO, error)
}

type dashboardUseCase struct {
	repo domain.ExchangeRateRepository
}

// NewDashboardUseCase crea una nueva instancia de DashboardUseCase.
func NewDashboardUseCase(repo domain.ExchangeRateRepository) DashboardUseCase {
	return &dashboardUseCase{
		repo: repo,
	}
}

// GetDashboardSummary obtiene la tasa actual (o en o antes de la fecha dada), calcula la variación porcentual con el día hábil anterior y recupera los últimos 7 días de historial.
func (uc *dashboardUseCase) GetDashboardSummary(ctx context.Context, currencyCode string, date *time.Time) (*DashboardResponseDTO, error) {
	if currencyCode == "" {
		return nil, errors.New("El código de moneda es requerido")
	}

	slog.Info("Obteniendo resumen de dashboard", "currency_code", currencyCode, "date", date)

	// 1. Obtener la tasa de referencia (GetLatestRate o GetLatestRateBeforeOrAt)
	var latestRate *domain.ExchangeRate
	var err error
	if date != nil {
		latestRate, err = uc.repo.GetLatestRateBeforeOrAt(ctx, currencyCode, *date)
	} else {
		latestRate, err = uc.repo.GetLatestRate(ctx, currencyCode)
	}
	if err != nil {
		return nil, err
	}

	// 2. Obtener la tasa del día hábil anterior a la de referencia
	previousRate, err := uc.repo.GetPreviousRate(ctx, currencyCode, latestRate.ValueDate)
	var variationPercent decimal.Decimal
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			slog.Warn("No se encontró tasa para el día hábil anterior (registro único). Se asume variación 0.00", "currency_code", currencyCode)
			variationPercent = decimal.Zero
		} else {
			slog.Warn("Error al obtener tasa anterior, se asume variación 0.00", "currency_code", currencyCode, "error", err)
			variationPercent = decimal.Zero
		}
	} else if previousRate == nil || previousRate.RateAverage.IsZero() {
		slog.Warn("La tasa anterior es nil o su promedio es cero, se asume variación 0.00", "currency_code", currencyCode)
		variationPercent = decimal.Zero
	} else {
		// 3. Calcular la variación porcentual matemática: ((TasaActual - TasaAnterior) / TasaAnterior) * 100
		diff := latestRate.RateAverage.Sub(previousRate.RateAverage)
		variationPercent = diff.Div(previousRate.RateAverage).Mul(decimal.NewFromInt(100))
	}

	// 4. Obtener el histórico: los últimos 7 días distintos registrados, INCLUYENDO días futuros
	//    ya publicados (p.ej. la tasa del próximo día hábil que el BCV publica por anticipado).
	//    La app determina qué día mostrar (el del dispositivo) y permite navegar entre todos.
	history, err := uc.repo.GetRatesHistory(ctx, currencyCode, 7)
	if err != nil {
		return nil, err
	}

	// Si no hay histórico, garantizamos que sea un slice vacío y no null
	if history == nil {
		history = []domain.ExchangeRate{}
	}

	return &DashboardResponseDTO{
		VariationPercent: variationPercent,
		History:          history,
	}, nil
}
