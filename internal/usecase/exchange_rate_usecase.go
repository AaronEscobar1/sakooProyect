package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
)

// ExchangeRateUseCase define la lógica de negocio para interactuar con las tasas de cambio.
type ExchangeRateUseCase struct {
	repo domain.ExchangeRateRepository
}

// NewExchangeRateUseCase crea una nueva instancia de ExchangeRateUseCase.
func NewExchangeRateUseCase(repo domain.ExchangeRateRepository) *ExchangeRateUseCase {
	return &ExchangeRateUseCase{
		repo: repo,
	}
}

// GetLatestRates devuelve la última tasa disponible de cada moneda en el sistema.
func (uc *ExchangeRateUseCase) GetLatestRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	return uc.repo.GetLatestRates(ctx)
}

// GetRatesHistory obtiene el historial paginado y filtrado de tasas de cambio con validación.
func (uc *ExchangeRateUseCase) GetRatesHistory(
	ctx context.Context,
	page, limit int,
	currencyCode string,
	startDateStr, endDateStr string,
) ([]domain.ExchangeRate, int, error) {
	// Validaciones básicas de paginación
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10 // Valor por defecto amigable
	} else if limit > 100 {
		limit = 100 // Evitar consultas maliciosas o sobrecargas de memoria
	}

	var startDate, endDate *time.Time

	// Parsear fecha de inicio YYYY-MM-DD
	if startDateStr != "" {
		t, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return nil, 0, fmt.Errorf("formato de start_date inválido (se requiere YYYY-MM-DD): %w", err)
		}
		startDate = &t
	}

	// Parsear fecha de fin YYYY-MM-DD
	if endDateStr != "" {
		t, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return nil, 0, fmt.Errorf("formato de end_date inválido (se requiere YYYY-MM-DD): %w", err)
		}
		endDate = &t
	}

	return uc.repo.GetRatesHistoryPaginated(ctx, page, limit, currencyCode, startDate, endDate)
}

// GetCalendarDates obtiene la lista de fechas únicas formateadas.
func (uc *ExchangeRateUseCase) GetCalendarDates(ctx context.Context) ([]string, error) {
	return uc.repo.GetCalendarDates(ctx)
}

