package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/message"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type messageRepository struct {
	client *ent.Client
}

// NewMessageRepository crea un repositorio para mensajería usando Ent.
func NewMessageRepository(client *ent.Client) domain.MessageRepository {
	return &messageRepository{
		client: client,
	}
}

func (r *messageRepository) Create(ctx context.Context, msg *domain.Message) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando nuevo mensaje en Ent", "sender_id", msg.SenderID, "receiver_id", msg.ReceiverID)

	builder := r.client.Message.Create().
		SetContent(msg.Content)

	if msg.SenderID != 0 {
		builder.SetSenderID(msg.SenderID)
	}
	if msg.ReceiverID != 0 {
		builder.SetReceiverID(msg.ReceiverID)
	}

	created, err := builder.Save(dbCtx)
	if err != nil {
		slog.Error("Fallo al guardar mensaje en Ent", "error", err)
		return fmt.Errorf("error al guardar mensaje: %w", err)
	}

	msg.ID = int64(created.ID)
	msg.CreatedAt = created.CreatedAt
	return nil
}

func (r *messageRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Message, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Recuperando mensajes de usuario en Ent", "user_id", userID)

	entMsgs, err := r.client.Message.Query().
		Where(
			message.Or(
				message.SenderID(userID),
				message.ReceiverID(userID),
			),
		).
		Order(ent.Asc(message.FieldCreatedAt)).
		All(dbCtx)

	if err != nil {
		slog.Error("Error al listar mensajes desde Ent", "error", err, "user_id", userID)
		return nil, fmt.Errorf("error al listar mensajes de la base de datos: %w", err)
	}

	var messages []domain.Message
	for _, m := range entMsgs {
		var readAt *time.Time
		if !m.ReadAt.IsZero() {
			readAt = &m.ReadAt
		}
		messages = append(messages, domain.Message{
			ID:         int64(m.ID),
			SenderID:   m.SenderID,
			ReceiverID: m.ReceiverID,
			Content:    m.Content,
			ReadAt:     readAt,
			CreatedAt:  m.CreatedAt,
		})
	}

	if messages == nil {
		messages = []domain.Message{}
	}

	// Marcar recibidos como leídos
	now := time.Now().UTC()
	_, _ = r.client.Message.Update().
		Where(
			message.ReceiverID(userID),
			message.ReadAtIsNil(),
		).
		SetReadAt(now).
		Save(dbCtx)

	for i := range messages {
		if messages[i].ReceiverID == userID && messages[i].ReadAt == nil {
			messages[i].ReadAt = &now
		}
	}

	return messages, nil
}

func (r *messageRepository) GetUnreadCount(ctx context.Context, userID int64) (int, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Contando mensajes no leídos para usuario en Ent", "user_id", userID)

	count, err := r.client.Message.Query().
		Where(
			message.ReceiverID(userID),
			message.ReadAtIsNil(),
		).
		Count(dbCtx)

	if err != nil {
		slog.Error("Fallo al contar mensajes no leídos en Ent", "error", err, "user_id", userID)
		return 0, fmt.Errorf("error al obtener mensajes no leídos: %w", err)
	}

	return count, nil
}
