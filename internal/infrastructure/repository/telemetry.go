package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type telemetryRepository struct {
	db *pgxpool.Pool
}

// NewTelemetryRepository crea una nueva instancia del repositorio de telemetría.
func NewTelemetryRepository(db *pgxpool.Pool) domain.TelemetryRepository {
	return &telemetryRepository{
		db: db,
	}
}

// GetAPILogs obtiene los logs de auditoría de forma paginada y ordenada por fecha descendente.
func (r *telemetryRepository) GetAPILogs(ctx context.Context, page, limit int) ([]domain.APILog, int, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando historial de logs de auditoría en base de datos", "page", page, "limit", limit)

	// 1. Obtener total de elementos
	countQuery := `SELECT COUNT(*) FROM api_logs;`
	var totalItems int
	err := r.db.QueryRow(dbCtx, countQuery).Scan(&totalItems)
	if err != nil {
		slog.Error("Fallo al contar logs de auditoría en PostgreSQL", "error", err)
		return nil, 0, fmt.Errorf("error al contar logs de auditoría: %w", err)
	}

	if totalItems == 0 {
		return []domain.APILog{}, 0, nil
	}

	// 2. Obtener los elementos paginados
	offset := (page - 1) * limit
	dataQuery := `
		SELECT 
			l.id, 
			l.track_code, 
			l.user_id, 
			COALESCE(u.username, u.first_name || ' ' || u.last_name) AS username,
			l.method, 
			l.path, 
			l.http_status, 
			l.response_code, 
			l.latency_ms, 
			l.created_at
		FROM api_logs l
		LEFT JOIN users u ON l.user_id = u.id
		ORDER BY l.created_at DESC
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.db.Query(dbCtx, dataQuery, limit, offset)
	if err != nil {
		slog.Error("Fallo al consultar logs de auditoría paginados", "error", err)
		return nil, 0, fmt.Errorf("error al consultar logs de auditoría: %w", err)
	}
	defer rows.Close()

	var logs []domain.APILog
	for rows.Next() {
		var l domain.APILog
		err := rows.Scan(
			&l.ID,
			&l.TrackCode,
			&l.UserID,
			&l.Username,
			&l.Method,
			&l.Path,
			&l.HTTPStatus,
			&l.ResponseCode,
			&l.LatencyMS,
			&l.CreatedAt,
		)
		if err != nil {
			slog.Error("Error al escanear fila de log de auditoría", "error", err)
			return nil, 0, fmt.Errorf("error al escanear log de auditoría: %w", err)
		}
		logs = append(logs, l)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error en iteración de logs de auditoría: %w", err)
	}

	if logs == nil {
		logs = []domain.APILog{}
	}

	return logs, totalItems, nil
}
