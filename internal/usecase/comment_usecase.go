package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
)

type commentUseCase struct {
	repo domain.CommentRepository
}

// NewCommentUseCase crea un caso de uso para comentarios en tasas de cambio.
func NewCommentUseCase(repo domain.CommentRepository) domain.CommentUseCase {
	return &commentUseCase{
		repo: repo,
	}
}

func (uc *commentUseCase) AddComment(ctx context.Context, userID, rateID int64, content string) (*domain.Comment, error) {
	if content == "" {
		return nil, errors.New("el contenido del comentario no puede estar vacío")
	}
	if rateID <= 0 {
		return nil, errors.New("el ID de tasa 'rate_id' no es válido")
	}

	slog.Info("Procesando creación de comentario para tasa", "user_id", userID, "rate_id", rateID)

	// Validar la regla de negocio: el usuario solo puede comentar 1 vez por tasa de cambio
	alreadyCommented, err := uc.repo.HasCommentedOnRate(ctx, userID, rateID)
	if err != nil {
		return nil, err
	}
	if alreadyCommented {
		return nil, errors.New("el usuario ya ha comentado en esta tasa de cambio, solo se permite un comentario por tasa")
	}

	comment := &domain.Comment{
		UserID:  userID,
		RateID:  rateID,
		Content: content,
	}

	err = uc.repo.Create(ctx, comment)
	if err != nil {
		return nil, err
	}

	return comment, nil
}

func (uc *commentUseCase) GetCommentsByRate(ctx context.Context, rateID int64) ([]domain.Comment, error) {
	if rateID <= 0 {
		return nil, errors.New("el ID de tasa 'rate_id' no es válido")
	}

	slog.Info("Procesando listado de opiniones del día para tasa", "rate_id", rateID)

	// Obtener opiniones del día de hoy
	today := time.Now()
	return uc.repo.ListByRateIDAndDate(ctx, rateID, today)
}
