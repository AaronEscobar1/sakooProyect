package domain

import (
	"context"
	"time"
)

// Comment representa un comentario u opinión sobre una tasa de cambio.
type Comment struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username,omitempty"` // Campo auxiliar para respuestas HTTP
	RateID    int64     `json:"rate_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// CommentRepository define las operaciones de persistencia para comentarios.
type CommentRepository interface {
	Create(ctx context.Context, comment *Comment) error
	HasCommentedOnRate(ctx context.Context, userID, rateID int64) (bool, error)
	ListByRateIDAndDate(ctx context.Context, rateID int64, date time.Time) ([]Comment, error)
}

// CommentUseCase define la lógica de negocio para comentarios.
type CommentUseCase interface {
	AddComment(ctx context.Context, userID, rateID int64, content string) (*Comment, error)
	GetCommentsByRate(ctx context.Context, rateID int64) ([]Comment, error)
}
