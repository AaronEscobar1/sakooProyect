package repository

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/notification"
	"github.com/aaron/sakoo-backend/ent/userdevicetoken"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type notificationRepository struct {
	client *ent.Client
}

// NewNotificationRepository crea una nueva instancia de NotificationRepository usando Ent.
func NewNotificationRepository(client *ent.Client) domain.NotificationRepository {
	return &notificationRepository{
		client: client,
	}
}

func (r *notificationRepository) SaveDeviceToken(ctx context.Context, userID int64, token, platform string) error {
	existing, err := r.client.UserDeviceToken.Query().
		Where(userdevicetoken.TokenEQ(token)).
		Only(ctx)

	if err == nil {
		_, err = r.client.UserDeviceToken.UpdateOne(existing).
			SetUserID(userID).
			SetPlatform(platform).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		return err
	}

	_, err = r.client.UserDeviceToken.Create().
		SetUserID(userID).
		SetToken(token).
		SetPlatform(platform).
		Save(ctx)

	if err != nil {
		slog.Error("Fallo al guardar token de dispositivo en Ent", "error", err, "user_id", userID)
		return err
	}
	return nil
}

func (r *notificationRepository) DeleteDeviceToken(ctx context.Context, userID int64, token string) error {
	_, err := r.client.UserDeviceToken.Delete().
		Where(
			userdevicetoken.UserID(userID),
			userdevicetoken.TokenEQ(token),
		).
		Exec(ctx)

	if err != nil {
		slog.Error("Fallo al eliminar token de dispositivo", "error", err, "user_id", userID)
		return err
	}
	return nil
}

func (r *notificationRepository) DeleteAllUserDeviceTokens(ctx context.Context, userID int64) error {
	_, err := r.client.UserDeviceToken.Delete().
		Where(userdevicetoken.UserID(userID)).
		Exec(ctx)

	if err != nil {
		slog.Error("Fallo al eliminar todos los tokens de dispositivo del usuario", "error", err, "user_id", userID)
		return err
	}
	return nil
}

func (r *notificationRepository) GetDeviceTokensByUserID(ctx context.Context, userID int64) ([]string, error) {
	tokens, err := r.client.UserDeviceToken.Query().
		Where(userdevicetoken.UserID(userID)).
		Order(ent.Desc(userdevicetoken.FieldUpdatedAt)).
		All(ctx)

	if err != nil {
		slog.Error("Fallo al obtener tokens de dispositivo", "error", err, "user_id", userID)
		return nil, err
	}

	var res []string
	for _, t := range tokens {
		res = append(res, t.Token)
	}
	return res, nil
}

func (r *notificationRepository) GetAllDeviceTokens(ctx context.Context) ([]string, error) {
	tokens, err := r.client.UserDeviceToken.Query().All(ctx)
	if err != nil {
		slog.Error("Fallo al obtener todos los tokens de dispositivo", "error", err)
		return nil, err
	}

	var res []string
	for _, t := range tokens {
		res = append(res, t.Token)
	}
	return res, nil
}

func (r *notificationRepository) SaveNotification(ctx context.Context, n *domain.Notification) error {
	builder := r.client.Notification.Create().
		SetUserID(n.UserID).
		SetTitle(n.Title).
		SetBody(n.Body).
		SetIsRead(n.IsRead)

	if n.Data != nil {
		builder.SetData(n.Data)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		slog.Error("Fallo al persistir notificación en Ent", "error", err, "user_id", n.UserID)
		return err
	}

	n.ID = int64(created.ID)
	n.CreatedAt = created.CreatedAt
	return nil
}

func (r *notificationRepository) GetNotificationsByUserID(ctx context.Context, userID int64) ([]domain.Notification, error) {
	entNotifs, err := r.client.Notification.Query().
		Where(notification.UserID(userID)).
		Order(ent.Desc(notification.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		slog.Error("Fallo al consultar bandeja de notificaciones en Ent", "error", err, "user_id", userID)
		return nil, err
	}

	var list []domain.Notification
	for _, n := range entNotifs {
		list = append(list, domain.Notification{
			ID:        int64(n.ID),
			UserID:    n.UserID,
			Title:     n.Title,
			Body:      n.Body,
			IsRead:    n.IsRead,
			Data:      n.Data,
			CreatedAt: n.CreatedAt,
		})
	}
	return list, nil
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, userID int64, notificationID int64) error {
	count, err := r.client.Notification.Update().
		Where(
			notification.IDEQ(int(notificationID)),
			notification.UserID(userID),
		).
		SetIsRead(true).
		Save(ctx)

	if err != nil {
		slog.Error("Fallo al marcar notificación como leída en Ent", "error", err, "notification_id", notificationID, "user_id", userID)
		return err
	}
	if count == 0 {
		return errors.New("Notificación no encontrada o no autorizada")
	}
	return nil
}
