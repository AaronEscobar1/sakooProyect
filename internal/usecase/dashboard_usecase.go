package usecase

import (
	"context"
	"errors"
	"log/slog"
	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// DashboardResponseDTO agrupa los datos consolidados requeridos por el Dashboard.
type DashboardResponseDTO struct {
	LatestRate       domain.ExchangeRate   `json:"latest_rate"`
	VariationPercent decimal.Decimal       `json:"variation_percent"`
	History          []domain.ExchangeRate `json:"history"`
}

// DashboardUseCase define el caso de uso para el Dashboard del MVP.
type DashboardUseCase interface {
	GetDashboardSummary(ctx context.Context, currencyCode string) (*DashboardResponseDTO, error)
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

// GetDashboardSummary obtiene la tasa actual, calcula la variación porcentual con el día hábil anterior y recupera los últimos 7 días de historial.
func (uc *dashboardUseCase) GetDashboardSummary(ctx context.Context, currencyCode string) (*DashboardResponseDTO, error) {
	if currencyCode == "" {
		return nil, errors.New("el código de moneda es requerido")
	}

	slog.Info("Obteniendo resumen de dashboard", "currency_code", currencyCode)

	// 1. Obtener la tasa actual (GetLatestRate)
	latestRate, err := uc.repo.GetLatestRate(ctx, currencyCode)
	if err != nil {
		return nil, err
	}

	// 2. Obtener la tasa del día hábil anterior a la actual
	previousRate, err := uc.repo.GetPreviousRate(ctx, currencyCode, latestRate.ValueDate)
	var variationPercent decimal.Decimal
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) || (err.Error() != "" && errors.Is(errors.Unwrap(err), pgx.ErrNoRows)) {
			slog.Warn("No se encontró tasa para el día hábil anterior (registro único). Se asume variación 0.00", "currency_code", currencyCode, "error", err)
			variationPercent = decimal.Zero
		} else {
			// Intentar verificar si el error contiene 'no rows in result set' de forma robusta
			if errors.Is(err, pgx.ErrNoRows) || (err != nil && (errors.Is(err, pgx.ErrNoRows) || errors.Is(errors.Unwrap(err), pgx.ErrNoRows) || err.Error() == "no rows in result set" || (len(err.Error()) > 20 && err.Error()[len(err.Error())-22:] == "no rows in result set"))) {
				slog.Warn("No se encontró tasa para el día hábil anterior. Se asume variación 0.00", "currency_code", currencyCode)
				variationPercent = decimal.Zero
			} else {
				// De lo contrario, registrar como advertencia pero no abortar, usar cero o propagar si es grave.
				// Para robustez y continuidad de negocio, si falla de alguna forma, registramos el warning y ponemos 0.00.
				slog.Warn("Error al obtener tasa anterior, se asume variación 0.00", "currency_code", currencyCode, "error", err)
				variationPercent = decimal.Zero
			}
		}
	} else if previousRate == nil || previousRate.RateAverage.IsZero() {
		slog.Warn("La tasa anterior es nil o su promedio es cero, se asume variación 0.00", "currency_code", currencyCode)
		variationPercent = decimal.Zero
	} else {
		// 3. Calcular la variación porcentual matemática: ((TasaActual - TasaAnterior) / TasaAnterior) * 100
		diff := latestRate.RateAverage.Sub(previousRate.RateAverage)
		variationPercent = diff.Div(previousRate.RateAverage).Mul(decimal.NewFromInt(100))
	}

	// 4. Obtener el histórico de los últimos 7 días
	history, err := uc.repo.GetRatesHistory(ctx, currencyCode, 7)
	if err != nil {
		return nil, err
	}

	// Si no hay histórico, garantizamos que sea un slice vacío y no null
	if history == nil {
		history = []domain.ExchangeRate{}
	}

	return &DashboardResponseDTO{
		LatestRate:       *latestRate,
		VariationPercent: variationPercent,
		History:          history,
	}, nil
}
