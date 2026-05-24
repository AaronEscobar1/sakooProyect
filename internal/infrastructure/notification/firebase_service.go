package notification

import (
	"context"
	"log/slog"
	"os"

	"github.com/aaron/sakoo-backend/internal/domain"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type firebasePushService struct {
	fcmClient *messaging.Client
	isMock    bool
}

// NewPushNotificationService crea un nuevo servicio de notificaciones Firebase.
// Lee las credenciales en formato JSON desde la variable de entorno FIREBASE_CREDENTIALS.
// Si no se encuentra la variable, opera en modo MOCK para desarrollo ágil y local sin dependencias.
func NewPushNotificationService() domain.PushNotificationService {
	creds := os.Getenv("FIREBASE_CREDENTIALS")
	if creds == "" {
		slog.Warn("⚠️ La variable FIREBASE_CREDENTIALS no está configurada. El servicio de Notificaciones Push funcionará en modo MOCK (Consola).")
		return &firebasePushService{
			isMock: true,
		}
	}

	ctx := context.Background()
	opt := option.WithCredentialsJSON([]byte(creds))
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		slog.Error("Fallo crítico al inicializar Firebase App", "error", err)
		return &firebasePushService{isMock: true}
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		slog.Error("Fallo crítico al obtener el cliente FCM de Firebase", "error", err)
		return &firebasePushService{isMock: true}
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
	}
}

func (s *firebasePushService) SendPush(ctx context.Context, token string, title, body string, data map[string]string) error {
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
