package domain

import (
	"context"
	"time"
)

// Message representa un mensaje interno entre dos usuarios.
type Message struct {
	ID         int64      `json:"id"`
	SenderID   int64      `json:"sender_id"`
	ReceiverID int64      `json:"receiver_id"`
	Content    string     `json:"content"`
	ReadAt     *time.Time `json:"read_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// MessageRepository define las operaciones de persistencia para mensajería.
type MessageRepository interface {
	Create(ctx context.Context, msg *Message) error
	ListByUserID(ctx context.Context, userID int64) ([]Message, error)
	GetUnreadCount(ctx context.Context, userID int64) (int, error)
}

// MessageUseCase define la lógica de negocio para mensajería.
type MessageUseCase interface {
	SendMessage(ctx context.Context, senderID, receiverID int64, content string) (*Message, error)
	GetMessages(ctx context.Context, userID int64) ([]Message, error)
	GetUnreadCount(ctx context.Context, userID int64) (int, error)
}
