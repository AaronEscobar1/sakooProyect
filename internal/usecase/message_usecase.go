package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/aaron/sakoo-backend/internal/domain"
)

type messageUseCase struct {
	repo domain.MessageRepository
}

// NewMessageUseCase crea un caso de uso para la mensajería interna.
func NewMessageUseCase(repo domain.MessageRepository) domain.MessageUseCase {
	return &messageUseCase{
		repo: repo,
	}
}

func (uc *messageUseCase) SendMessage(ctx context.Context, senderID, receiverID int64, content string) (*domain.Message, error) {
	if content == "" {
		return nil, errors.New("El contenido del mensaje no puede estar vacío")
	}
	if senderID == receiverID {
		return nil, errors.New("No puedes enviarte un mensaje a ti mismo")
	}

	slog.Info("Procesando envío de mensaje", "sender_id", senderID, "receiver_id", receiverID)

	msg := &domain.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
	}

	err := uc.repo.Create(ctx, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (uc *messageUseCase) GetMessages(ctx context.Context, userID int64) ([]domain.Message, error) {
	slog.Info("Procesando obtención de mensajes para usuario", "user_id", userID)
	return uc.repo.ListByUserID(ctx, userID)
}

func (uc *messageUseCase) GetUnreadCount(ctx context.Context, userID int64) (int, error) {
	slog.Info("Procesando conteo de mensajes no leídos para usuario", "user_id", userID)
	return uc.repo.GetUnreadCount(ctx, userID)
}
