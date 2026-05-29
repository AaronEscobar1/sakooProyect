package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type commentRepository struct {
	db *pgxpool.Pool
}

// NewCommentRepository crea un repositorio para comentarios en tasas.
func NewCommentRepository(db *pgxpool.Pool) domain.CommentRepository {
	return &commentRepository{
		db: db,
	}
}

func (r *commentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando nuevo comentario de tasa", "user_id", comment.UserID, "rate_id", comment.RateID)

	query := `
		WITH inserted AS (
			INSERT INTO comments (user_id, rate_id, content, created_at)
			VALUES ($1, $2, $3, NOW())
			RETURNING id, user_id, created_at
		)
		SELECT i.id, i.created_at, COALESCE(u.username, u.first_name || ' ' || u.last_name, 'Usuario Anónimo') AS username
		FROM inserted i
		LEFT JOIN users u ON i.user_id = u.id;
	`
	err := r.db.QueryRow(dbCtx, query, comment.UserID, comment.RateID, comment.Content).Scan(
		&comment.ID,
		&comment.CreatedAt,
		&comment.Username,
	)
	if err != nil {
		slog.Error("Fallo al guardar comentario en PostgreSQL", "error", err)
		return fmt.Errorf("error al guardar comentario: %w", err)
	}

	return nil
}

func (r *commentRepository) HasCommentedOnRate(ctx context.Context, userID, rateID int64) (bool, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Verificando si el usuario ya comentó la tasa", "user_id", userID, "rate_id", rateID)

	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM comments 
			WHERE user_id = $1 AND rate_id = $2
		);
	`
	var exists bool
	err := r.db.QueryRow(dbCtx, query, userID, rateID).Scan(&exists)
	if err != nil {
		slog.Error("Fallo al verificar si el usuario ya comentó", "error", err, "user_id", userID, "rate_id", rateID)
		return false, fmt.Errorf("error al verificar comentario previo: %w", err)
	}

	return exists, nil
}

func (r *commentRepository) ListByRateIDAndDate(ctx context.Context, rateID int64, date time.Time) ([]domain.Comment, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Listando opiniones del día para la tasa", "rate_id", rateID, "date", date)

	// Listar comentarios para un rate_id específico creados en un día específico.
	// Hacemos un JOIN con la tabla de usuarios para recuperar su username oficial con fallback al nombre completo para pintar en el front.
	query := `
		SELECT c.id, c.user_id, COALESCE(u.username, u.first_name || ' ' || u.last_name, 'Usuario Anónimo') AS username, c.rate_id, c.content, c.created_at
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		WHERE c.rate_id = $1 AND c.created_at::date = $2::date
		ORDER BY c.created_at DESC;
	`
	rows, err := r.db.Query(dbCtx, query, rateID, date)
	if err != nil {
		slog.Error("Fallo al listar comentarios en PostgreSQL", "error", err, "rate_id", rateID)
		return nil, fmt.Errorf("error al listar comentarios: %w", err)
	}
	defer rows.Close()

	var comments []domain.Comment
	for rows.Next() {
		var c domain.Comment
		err := rows.Scan(&c.ID, &c.UserID, &c.Username, &c.RateID, &c.Content, &c.CreatedAt)
		if err != nil {
			slog.Error("Error al escanear fila de comentario", "error", err)
			return nil, fmt.Errorf("error al decodificar comentario: %w", err)
		}
		comments = append(comments, c)
	}

	if comments == nil {
		comments = []domain.Comment{}
	}

	return comments, nil
}
