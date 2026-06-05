package usecase

import (
	"context"
	"log/slog"

	"github.com/aaron/sakoo-backend/internal/domain"
)

type telemetryUseCase struct {
	repo domain.TelemetryRepository
}

// NewTelemetryUseCase crea una nueva instancia del caso de uso de telemetría.
func NewTelemetryUseCase(repo domain.TelemetryRepository) domain.TelemetryUseCase {
	return &telemetryUseCase{
		repo: repo,
	}
}

// GetAPILogs obtiene el listado paginado de logs de auditoría aplicando validaciones básicas de entrada.
func (uc *telemetryUseCase) GetAPILogs(ctx context.Context, page, limit int) ([]domain.APILog, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // Límite de seguridad para evitar sobrecarga del servidor y base de datos
	}

	slog.Info("Procesando caso de uso de listado de logs de auditoría", "page", page, "limit", limit)
	return uc.repo.GetAPILogs(ctx, page, limit)
}
