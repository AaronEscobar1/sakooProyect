package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/comment"
	"github.com/aaron/sakoo-backend/ent/user"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type commentRepository struct {
	client *ent.Client
}

// NewCommentRepository crea un repositorio para comentarios en tasas usando Ent.
func NewCommentRepository(client *ent.Client) domain.CommentRepository {
	return &commentRepository{
		client: client,
	}
}

func (r *commentRepository) Create(ctx context.Context, c *domain.Comment) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando nuevo comentario de tasa en Ent", "user_id", c.UserID, "rate_id", c.RateID)

	builder := r.client.Comment.Create().
		SetRateID(c.RateID).
		SetContent(c.Content)

	if c.UserID != 0 {
		builder.SetUserID(c.UserID)
	}

	created, err := builder.Save(dbCtx)
	if err != nil {
		slog.Error("Fallo al guardar comentario en Ent", "error", err)
		return fmt.Errorf("error al guardar comentario: %w", err)
	}

	c.ID = int64(created.ID)
	c.CreatedAt = created.CreatedAt

	if c.UserID != 0 {
		u, err := r.client.User.Query().Where(user.IDEQ(int(c.UserID))).Only(dbCtx)
		if err == nil {
			if u.Username != "" {
				c.Username = u.Username
			} else {
				c.Username = fmt.Sprintf("%s %s", u.FirstName, u.LastName)
			}
		} else {
			c.Username = "Usuario Anónimo"
		}
	} else {
		c.Username = "Usuario Anónimo"
	}

	return nil
}

func (r *commentRepository) HasCommentedOnRate(ctx context.Context, userID, rateID int64) (bool, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Verificando si el usuario ya comentó la tasa", "user_id", userID, "rate_id", rateID)

	return r.client.Comment.Query().
		Where(comment.UserID(userID), comment.RateID(rateID)).
		Exist(dbCtx)
}

func (r *commentRepository) ListByRateID(ctx context.Context, rateID int64) ([]domain.Comment, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Listando opiniones para la tasa", "rate_id", rateID)

	comments, err := r.client.Comment.Query().
		Where(comment.RateID(rateID)).
		Order(ent.Desc(comment.FieldCreatedAt)).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al listar comentarios en Ent", "error", err, "rate_id", rateID)
		return nil, fmt.Errorf("error al listar comentarios: %w", err)
	}

	var result []domain.Comment
	for _, c := range comments {
		username := "Usuario Anónimo"
		if c.UserID != 0 {
			u, err := r.client.User.Query().Where(user.IDEQ(int(c.UserID))).Only(dbCtx)
			if err == nil {
				if u.Username != "" {
					username = u.Username
				} else {
					username = fmt.Sprintf("%s %s", u.FirstName, u.LastName)
				}
			}
		}

		result = append(result, domain.Comment{
			ID:        int64(c.ID),
			UserID:    c.UserID,
			Username:  username,
			RateID:    c.RateID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
		})
	}

	if result == nil {
		result = []domain.Comment{}
	}

	return result, nil
}
