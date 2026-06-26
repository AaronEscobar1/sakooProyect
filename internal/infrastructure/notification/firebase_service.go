package notification

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/aaron/sakoo-backend/internal/domain"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type firebasePushService struct {
	fcmClient *messaging.Client
	isMock    bool
	disabled  bool
}

// pushNotificationsDisabled determina si el envío de notificaciones push está deshabilitado
// por configuración. Se controla con la variable de entorno PUSH_NOTIFICATIONS_ENABLED:
// cualquier valor distinto de "true" (false/0/no/off) deshabilita el envío. Por defecto
// (variable ausente) está HABILITADO, para no alterar el comportamiento en desarrollo.
// Se usa para apagar el envío de push en entornos quality y production.
func pushNotificationsDisabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PUSH_NOTIFICATIONS_ENABLED")))
	switch v {
	case "", "true", "1", "yes", "on":
		return false
	default:
		return true
	}
}

// NewPushNotificationService crea un nuevo servicio de notificaciones Firebase.
// Lee las credenciales en formato JSON desde la variable de entorno FIREBASE_CREDENTIALS.
// Si no se encuentra la variable, opera en modo MOCK para desarrollo ágil y local sin dependencias.
func NewPushNotificationService() domain.PushNotificationService {
	disabled := pushNotificationsDisabled()
	if disabled {
		slog.Warn("🔕 Envío de notificaciones push DESHABILITADO por configuración (PUSH_NOTIFICATIONS_ENABLED=false).")
	}

	creds := os.Getenv("FIREBASE_CREDENTIALS")
	if creds == "" {
		slog.Warn("⚠️ La variable FIREBASE_CREDENTIALS no está configurada. El servicio de Notificaciones Push funcionará en modo MOCK (Consola).")
		return &firebasePushService{
			isMock:   true,
			disabled: disabled,
		}
	}

	ctx := context.Background()
	opt := option.WithCredentialsJSON([]byte(creds))
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		slog.Error("Fallo crítico al inicializar Firebase App", "error", err)
		return &firebasePushService{isMock: true, disabled: disabled}
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		slog.Error("Fallo crítico al obtener el cliente FCM de Firebase", "error", err)
		return &firebasePushService{isMock: true, disabled: disabled}
	}

	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "production"
	}

	var envSpanish string
	switch env {
	case "production":
		envSpanish = "producción"
	case "qa":
		envSpanish = "qa"
	case "local":
		envSpanish = "local"
	default:
		envSpanish = env
	}

	slog.Info("🚀 Firebase Cloud Messaging (FCM) inicializado correctamente en " + envSpanish + ".", "ambiente", env)
	return &firebasePushService{
		fcmClient: client,
		isMock:    false,
		disabled:  disabled,
	}
}

func (s *firebasePushService) SendPush(ctx context.Context, token string, title, body string, data map[string]string) error {
	if s.disabled {
		slog.Debug("Envío de push individual omitido: notificaciones deshabilitadas por configuración", "title", title)
		return nil
	}
	if s.isMock {
		slog.Info("📢 [MOCK PUSH] Enviando notificación push individual", 
			"token", token, 
			"title", title, 
			"body", body, 
			"data", data,
		)
		return nil
	}

	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	res, err := s.fcmClient.Send(ctx, msg)
	if err != nil {
		slog.Error("Fallo al enviar notificación push individual vía Firebase", "error", err, "token", token)
		return err
	}

	slog.Info("Notificación push enviada con éxito vía Firebase", "message_id", res, "token", token)
	return nil
}

func (s *firebasePushService) SendMulticastPush(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if s.disabled {
		slog.Debug("Envío de push multicast omitido: notificaciones deshabilitadas por configuración", "title", title)
		return nil
	}
	if len(tokens) == 0 {
		return nil
	}

	if s.isMock {
		slog.Info("📢 [MOCK PUSH] Enviando notificación push masiva (multicast)", 
			"cantidad_tokens", len(tokens), 
			"tokens", tokens, 
			"title", title, 
			"body", body, 
			"data", data,
		)
		return nil
	}

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	br, err := s.fcmClient.SendEachForMulticast(ctx, msg)
	if err != nil {
		slog.Error("Fallo crítico al enviar multicast push vía Firebase", "error", err)
		return err
	}

	slog.Info("Notificación multicast push enviada vía Firebase", 
		"éxitos", br.SuccessCount, 
		"fallos", br.FailureCount,
	)
	return nil
}

func (s *firebasePushService) SendTopicPush(ctx context.Context, topic string, title, body string, data map[string]string) error {
	if s.disabled {
		slog.Debug("Envío de push a topic omitido: notificaciones deshabilitadas por configuración", "topic", topic, "title", title)
		return nil
	}
	if s.isMock {
		slog.Info("📢 [MOCK PUSH] Enviando notificación push a Topic", 
			"topic", topic, 
			"title", title, 
			"body", body, 
			"data", data,
		)
		return nil
	}

	msg := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	res, err := s.fcmClient.Send(ctx, msg)
	if err != nil {
		slog.Error("Fallo al enviar notificación push a Topic vía Firebase", "error", err, "topic", topic)
		return err
	}

	slog.Info("Notificación push a Topic enviada con éxito vía Firebase", "message_id", res, "topic", topic)
	return nil
}
