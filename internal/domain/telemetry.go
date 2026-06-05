package domain

import (
	"context"
	"time"
)

// APILog representa la entidad para un log de auditoría registrado por el middleware de trazabilidad.
type APILog struct {
	ID           int64     `json:"id"`
	TrackCode    string    `json:"track_code"`
	UserID       *int64    `json:"user_id,omitempty"`
	Username     *string   `json:"username,omitempty"` // Campo auxiliar del JOIN de usuarios
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	HTTPStatus   int       `json:"http_status"`
	ResponseCode *string   `json:"response_code,omitempty"`
	LatencyMS    int64     `json:"latency_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

// TelemetryRepository define los métodos de persistencia para telemetría y logs.
type TelemetryRepository interface {
	GetAPILogs(ctx context.Context, page, limit int) ([]APILog, int, error)
}

// TelemetryUseCase define la lógica de negocio para auditoría de telemetría.
type TelemetryUseCase interface {
	GetAPILogs(ctx context.Context, page, limit int) ([]APILog, int, error)
}
