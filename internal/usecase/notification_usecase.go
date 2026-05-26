package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aaron/sakoo-backend/internal/domain"
)

type notificationUseCase struct {
	repo        domain.NotificationRepository
	pushService domain.PushNotificationService
}

// NewNotificationUseCase crea una nueva instancia del caso de uso de notificaciones.
func NewNotificationUseCase(repo domain.NotificationRepository, pushService domain.PushNotificationService) domain.NotificationUseCase {
	return &notificationUseCase{
		repo:        repo,
		pushService: pushService,
	}
}

func (uc *notificationUseCase) RegisterDevice(ctx context.Context, userID int64, token, platform string) error {
	if token == "" || platform == "" {
		return fmt.Errorf("Token y plataforma son requeridos")
	}
	return uc.repo.SaveDeviceToken(ctx, userID, token, platform)
}

func (uc *notificationUseCase) UnregisterDevice(ctx context.Context, userID int64, token string) error {
	if token == "" {
		return fmt.Errorf("El token es requerido")
	}
	return uc.repo.DeleteDeviceToken(ctx, userID, token)
}

func (uc *notificationUseCase) GetNotifications(ctx context.Context, userID int64) ([]domain.Notification, error) {
	return uc.repo.GetNotificationsByUserID(ctx, userID)
}

func (uc *notificationUseCase) MarkNotificationAsRead(ctx context.Context, userID int64, notificationID int64) error {
	return uc.repo.MarkAsRead(ctx, userID, notificationID)
}

func (uc *notificationUseCase) SendSystemNotification(ctx context.Context, userID int64, title, body string, payload map[string]interface{}) error {
	// 1. Persistir la notificación en la base de datos (Inbox de la App)
	n := &domain.Notification{
		UserID: userID,
		Title:  title,
		Body:   body,
		IsRead: false,
		Data:   payload,
	}

	if err := uc.repo.SaveNotification(ctx, n); err != nil {
		slog.Error("Fallo al persistir la notificación del sistema", "error", err, "user_id", userID)
		return err
	}

	// 2. Obtener los tokens de los dispositivos activos del usuario
	tokens, err := uc.repo.GetDeviceTokensByUserID(ctx, userID)
	if err != nil || len(tokens) == 0 {
		// No hay tokens activos para este usuario, guardado en inbox finalizado con éxito
		slog.Debug("No hay tokens de dispositivos activos registrados para el usuario", "user_id", userID)
		return nil
	}

	// 3. Convertir el payload a map[string]string para la API de Firebase
	fcmData := make(map[string]string)
	for k, v := range payload {
		fcmData[k] = fmt.Sprintf("%v", v)
	}

	// 4. Disparar el envío de la notificación push de forma asíncrona para no bloquear latencia de la API
	go func(activeTokens []string, t, b string, data map[string]string) {
		bgCtx := context.Background()
		_ = uc.pushService.SendMulticastPush(bgCtx, activeTokens, t, b, data)
	}(tokens, title, body, fcmData)

	return nil
}

func (uc *notificationUseCase) SendBroadcastNotification(ctx context.Context, title, body string, payload map[string]interface{}) error {
	// 1. Obtener todos los tokens del sistema para el envío masivo
	tokens, err := uc.repo.GetAllDeviceTokens(ctx)
	if err != nil || len(tokens) == 0 {
		slog.Info("No hay ningún dispositivo registrado en el sistema para envío broadcast.")
		return nil
	}

	// 2. Convertir el payload a map[string]string
	fcmData := make(map[string]string)
	for k, v := range payload {
		fcmData[k] = fmt.Sprintf("%v", v)
	}

	// 3. Disparar el envío masivo asíncrono
	go func(activeTokens []string, t, b string, data map[string]string) {
		bgCtx := context.Background()
		_ = uc.pushService.SendMulticastPush(bgCtx, activeTokens, t, b, data)
	}(tokens, title, body, fcmData)

	return nil
}

func (uc *notificationUseCase) SendTopicNotification(ctx context.Context, topic string, title, body string, payload map[string]interface{}) error {
	// 1. Convertir el payload a map[string]string
	fcmData := make(map[string]string)
	for k, v := range payload {
		fcmData[k] = fmt.Sprintf("%v", v)
	}

	// 2. Disparar el envío asíncrono al Topic
	go func(tp, t, b string, data map[string]string) {
		bgCtx := context.Background()
		_ = uc.pushService.SendTopicPush(bgCtx, tp, t, b, data)
	}(topic, title, body, fcmData)

	return nil
}
