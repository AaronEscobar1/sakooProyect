package repository

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type notificationRepository struct {
	db *pgxpool.Pool
}

// NewNotificationRepository crea una nueva instancia de NotificationRepository.
func NewNotificationRepository(db *pgxpool.Pool) domain.NotificationRepository {
	return &notificationRepository{
		db: db,
	}
}

func (r *notificationRepository) SaveDeviceToken(ctx context.Context, userID int64, token, platform string) error {
	query := `
		INSERT INTO public.user_device_tokens (user_id, token, platform, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (token) DO UPDATE 
		SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform, updated_at = NOW();
	`
	_, err := r.db.Exec(ctx, query, userID, token, platform)
	if err != nil {
		slog.Error("Fallo al guardar token de dispositivo", "error", err, "user_id", userID)
		return err
	}
	return nil
}

func (r *notificationRepository) DeleteDeviceToken(ctx context.Context, userID int64, token string) error {
	query := `DELETE FROM public.user_device_tokens WHERE user_id = $1 AND token = $2;`
	_, err := r.db.Exec(ctx, query, userID, token)
	if err != nil {
		slog.Error("Fallo al eliminar token de dispositivo", "error", err, "user_id", userID)
		return err
	}
	return nil
}

func (r *notificationRepository) GetDeviceTokensByUserID(ctx context.Context, userID int64) ([]string, error) {
	query := `SELECT token FROM public.user_device_tokens WHERE user_id = $1 ORDER BY updated_at DESC;`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		slog.Error("Fallo al obtener tokens de dispositivo", "error", err, "user_id", userID)
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil {
			tokens = append(tokens, t)
		}
	}
	return tokens, nil
}

func (r *notificationRepository) GetAllDeviceTokens(ctx context.Context) ([]string, error) {
	query := `SELECT token FROM public.user_device_tokens;`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		slog.Error("Fallo al obtener todos los tokens de dispositivo", "error", err)
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil {
			tokens = append(tokens, t)
		}
	}
	return tokens, nil
}

func (r *notificationRepository) SaveNotification(ctx context.Context, n *domain.Notification) error {
	var dataJSON []byte
	var err error
	if n.Data != nil {
		dataJSON, err = json.Marshal(n.Data)
		if err != nil {
			slog.Error("Fallo al serializar payload de datos de notificación", "error", err, "user_id", n.UserID)
			return err
		}
	}

	query := `
		INSERT INTO public.notifications (user_id, title, body, is_read, data, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, created_at;
	`
	err = r.db.QueryRow(ctx, query, n.UserID, n.Title, n.Body, n.IsRead, dataJSON).Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		slog.Error("Fallo al persistir notificación en base de datos", "error", err, "user_id", n.UserID)
		return err
	}
	return nil
}

func (r *notificationRepository) GetNotificationsByUserID(ctx context.Context, userID int64) ([]domain.Notification, error) {
	query := `
		SELECT id, user_id, title, body, is_read, data, created_at
		FROM public.notifications
		WHERE user_id = $1
		ORDER BY created_at DESC;
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		slog.Error("Fallo al consultar bandeja de notificaciones", "error", err, "user_id", userID)
		return nil, err
	}
	defer rows.Close()

	var list []domain.Notification
	for rows.Next() {
		var n domain.Notification
		var dataJSON []byte
		err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.IsRead, &dataJSON, &n.CreatedAt)
		if err == nil {
			if len(dataJSON) > 0 {
				_ = json.Unmarshal(dataJSON, &n.Data)
			}
			list = append(list, n)
		}
	}
	return list, nil
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, userID int64, notificationID int64) error {
	query := `UPDATE public.notifications SET is_read = TRUE WHERE id = $1 AND user_id = $2;`
	res, err := r.db.Exec(ctx, query, notificationID, userID)
	if err != nil {
		slog.Error("Fallo al marcar notificación como leída", "error", err, "notification_id", notificationID, "user_id", userID)
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("notificación no encontrada o no autorizada")
	}
	return nil
}
