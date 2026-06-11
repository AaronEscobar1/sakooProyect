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

// ApproveRate valida y ejecuta la aprobación manual de una tasa de cambio desde el BackOffice.
// Actualiza rate_from, rate_to, rate_average, cambia el status a 'APPROVED' y asienta el source.
func (uc *ExchangeRateUseCase) ApproveRate(ctx context.Context, req domain.ApproveExchangeRateRequest, adminUserID int64) error {
	slog.Info("Ejecutando caso de uso de aprobación de tasa de cambio (BackOffice)",
		"rate_id", req.RateID, "admin_user_id", adminUserID,
	)

	// 1. Validaciones defensivas de entrada
	if req.RateID <= 0 {
		return errors.New("el ID de la tasa de cambio es requerido y debe ser un número positivo")
	}

	zero := decimal.NewFromInt(0)
	if req.RateFrom.LessThanOrEqual(zero) {
		return errors.New("el campo rate_from debe ser un valor positivo mayor a cero")
	}
	if req.RateTo.LessThanOrEqual(zero) {
		return errors.New("el campo rate_to debe ser un valor positivo mayor a cero")
	}
	if req.RateAverage.LessThanOrEqual(zero) {
		return errors.New("el campo rate_average debe ser un valor positivo mayor a cero")
	}

	if req.Source == "" {
		return errors.New("el campo source es requerido (ej: 'MANUAL', 'SCRAPING')")
	}

	// 2. Ejecutar la aprobación en el repositorio
	if err := uc.repo.UpdateRateApproval(ctx, req.RateID, req.RateFrom, req.RateTo, req.RateAverage, req.Source); err != nil {
		slog.Error("Fallo al aprobar tasa de cambio en el caso de uso",
			"error", err, "rate_id", req.RateID, "admin_user_id", adminUserID,
		)
		return err
	}

	slog.Info("Tasa de cambio aprobada exitosamente desde el BackOffice",
		"rate_id", req.RateID, "admin_user_id", adminUserID,
		"rate_from", req.RateFrom.String(),
		"rate_to", req.RateTo.String(),
		"rate_average", req.RateAverage.String(),
		"source", req.Source,
	)

	return nil
}

// GetLast7DaysRates obtiene las tasas de los últimos 7 días delegando en el repositorio.
func (uc *ExchangeRateUseCase) GetLast7DaysRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	return uc.repo.GetLast7DaysRates(ctx)
}

