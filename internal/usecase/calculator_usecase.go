package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// CalculatorUseCase define la lógica del caso de uso de la calculadora.
type CalculatorUseCase interface {
	CalculateConversion(ctx context.Context, currencyCode string, amount decimal.Decimal, dateStr string) (decimal.Decimal, error)
}

type calculatorUseCase struct {
	repo domain.ExchangeRateRepository
}

// NewCalculatorUseCase crea una nueva instancia de CalculatorUseCase.
func NewCalculatorUseCase(repo domain.ExchangeRateRepository) CalculatorUseCase {
	return &calculatorUseCase{
		repo: repo,
	}
}

// CalculateConversion obtiene la tasa de cambio para la moneda (la más reciente o la de una fecha específica si se provee) y multiplica el monto por la tasa promedio.
func (uc *calculatorUseCase) CalculateConversion(ctx context.Context, currencyCode string, amount decimal.Decimal, dateStr string) (decimal.Decimal, error) {
	if currencyCode == "" {
		return decimal.Zero, errors.New("el código de moneda es requerido")
	}
	if amount.IsNegative() {
		return decimal.Zero, errors.New("el monto a convertir no puede ser negativo")
	}

	slog.Info("Realizando conversión de moneda", "currency_code", currencyCode, "amount", amount.String(), "date", dateStr)

	var rate *domain.ExchangeRate
	var err error

	if dateStr != "" {
		// Intentar parsear la fecha provista
		parsedDate, errParse := time.Parse("2006-01-02", dateStr)
		if errParse != nil {
			return decimal.Zero, fmt.Errorf("formato de fecha inválido (se requiere YYYY-MM-DD): %w", errParse)
		}
		// Buscar la tasa para esa fecha específica
		rate, err = uc.repo.GetRateByDate(ctx, currencyCode, parsedDate)
		if err != nil {
			return decimal.Zero, err
		}
	} else {
		// Obtener la tasa de cambio más reciente
		rate, err = uc.repo.GetLatestRate(ctx, currencyCode)
		if err != nil {
			return decimal.Zero, err
		}
	}

	// Multiplicar amount * rate_average
	result := amount.Mul(rate.RateAverage)

	slog.Debug("Conversión realizada con éxito", 
		"currency_code", currencyCode, 
		"amount", amount.String(), 
		"rate_average", rate.RateAverage.String(), 
		"result", result.String(),
	)

	return result, nil
}
