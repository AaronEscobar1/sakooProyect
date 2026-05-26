package domain

import (
	"context"
	"time"
)

// UserDeviceToken representa un token FCM registrado para un dispositivo de usuario.
type UserDeviceToken struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"` // 'android', 'ios', 'web'
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Notification representa una notificación enviada al historial del usuario (inbox).
type Notification struct {
	ID        int64                  `json:"id"`
	UserID    int64                  `json:"user_id"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	IsRead    bool                   `json:"is_read"`
	Data      map[string]interface{} `json:"data"` // Datos JSON adicionales
	CreatedAt time.Time              `json:"created_at"`
}

// DTOs para solicitudes de la API
type RegisterDeviceRequest struct {
	Token    string `json:"token" example:"fcm_token_12345"`
	Platform string `json:"platform" example:"android"`
}

type UnregisterDeviceRequest struct {
	Token string `json:"token" example:"fcm_token_12345"`
}

type SendAdminNotificationRequest struct {
	UserID *int64                 `json:"user_id,omitempty" example:"4"` // Nulo significa broadcast a todos
	Title  string                 `json:"title" example:"Actualización de Tasas"`
	Body   string                 `json:"body" example:"El BCV acaba de actualizar las tasas para el día hábil de mañana."`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// NotificationRepository define la interfaz para persistir notificaciones y tokens.
type NotificationRepository interface {
	SaveDeviceToken(ctx context.Context, userID int64, token, platform string) error
	DeleteDeviceToken(ctx context.Context, userID int64, token string) error
	GetDeviceTokensByUserID(ctx context.Context, userID int64) ([]string, error)
	GetAllDeviceTokens(ctx context.Context) ([]string, error)
	SaveNotification(ctx context.Context, notification *Notification) error
	GetNotificationsByUserID(ctx context.Context, userID int64) ([]Notification, error)
	MarkAsRead(ctx context.Context, userID int64, notificationID int64) error
}

// PushNotificationService define el contrato para enviar notificaciones mediante Firebase FCM.
type PushNotificationService interface {
	SendPush(ctx context.Context, token string, title, body string, data map[string]string) error
	SendMulticastPush(ctx context.Context, tokens []string, title, body string, data map[string]string) error
	SendTopicPush(ctx context.Context, topic string, title, body string, data map[string]string) error
}

// NotificationUseCase define la lógica de negocio para las notificaciones.
type NotificationUseCase interface {
	RegisterDevice(ctx context.Context, userID int64, token, platform string) error
	UnregisterDevice(ctx context.Context, userID int64, token string) error
	GetNotifications(ctx context.Context, userID int64) ([]Notification, error)
	MarkNotificationAsRead(ctx context.Context, userID int64, notificationID int64) error
	SendSystemNotification(ctx context.Context, userID int64, title, body string, payload map[string]interface{}) error
	SendBroadcastNotification(ctx context.Context, title, body string, payload map[string]interface{}) error
	SendTopicNotification(ctx context.Context, topic string, title, body string, payload map[string]interface{}) error
}
