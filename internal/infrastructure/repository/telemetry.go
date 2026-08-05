package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/apilog"
	"github.com/aaron/sakoo-backend/ent/user"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type telemetryRepository struct {
	client *ent.Client
}

// NewTelemetryRepository crea una nueva instancia del repositorio de telemetría usando Ent.
func NewTelemetryRepository(client *ent.Client) domain.TelemetryRepository {
	return &telemetryRepository{
		client: client,
	}
}

func (r *telemetryRepository) GetAPILogs(ctx context.Context, page, limit int) ([]domain.APILog, int, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando historial de logs de auditoría en Ent", "page", page, "limit", limit)

	totalItems, err := r.client.ApiLog.Query().Count(dbCtx)
	if err != nil {
		slog.Error("Fallo al contar logs de auditoría en Ent", "error", err)
		return nil, 0, fmt.Errorf("error al contar logs de auditoría: %w", err)
	}

	if totalItems == 0 {
		return []domain.APILog{}, 0, nil
	}

	offset := (page - 1) * limit
	logs, err := r.client.ApiLog.Query().
		Order(ent.Desc(apilog.FieldCreatedAt)).
		Offset(offset).
		Limit(limit).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al consultar logs de auditoría paginados en Ent", "error", err)
		return nil, 0, fmt.Errorf("error al consultar logs de auditoría: %w", err)
	}

	var result []domain.APILog
	for _, l := range logs {
		var usernamePtr *string
		if l.UserID != 0 {
			u, err := r.client.User.Query().Where(user.IDEQ(int(l.UserID))).Only(dbCtx)
			if err == nil {
				var uname string
				if u.Username != "" {
					uname = u.Username
				} else {
					uname = fmt.Sprintf("%s %s", u.FirstName, u.LastName)
				}
				usernamePtr = &uname
			}
		}

		var uid *int64
		if l.UserID != 0 {
			uid = &l.UserID
		}

		var respCodePtr *string
		if l.ResponseCode != "" {
			respCodePtr = &l.ResponseCode
		}

		result = append(result, domain.APILog{
			ID:           int64(l.ID),
			TrackCode:    l.TrackCode,
			UserID:       uid,
			Username:     usernamePtr,
			Method:       l.Method,
			Path:         l.Path,
			HTTPStatus:   l.HTTPStatus,
			ResponseCode: respCodePtr,
			LatencyMS:    l.LatencyMs,
			CreatedAt:    l.CreatedAt,
		})
	}

	if result == nil {
		result = []domain.APILog{}
	}

	return result, totalItems, nil
}
