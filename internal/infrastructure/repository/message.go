package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type messageRepository struct {
	db *pgxpool.Pool
}

// NewMessageRepository crea un repositorio para mensajería.
func NewMessageRepository(db *pgxpool.Pool) domain.MessageRepository {
	return &messageRepository{
		db: db,
	}
}

func (r *messageRepository) Create(ctx context.Context, msg *domain.Message) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando nuevo mensaje", "sender_id", msg.SenderID, "receiver_id", msg.ReceiverID)

	query := `
		INSERT INTO messages (sender_id, receiver_id, content, read_at, created_at)
		VALUES ($1, $2, $3, NULL, NOW())
		RETURNING id, created_at;
	`
	err := r.db.QueryRow(dbCtx, query, msg.SenderID, msg.ReceiverID, msg.Content).Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		slog.Error("Fallo al guardar mensaje en PostgreSQL", "error", err)
		return fmt.Errorf("error al guardar mensaje: %w", err)
	}

	return nil
}

func (r *messageRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Message, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Recuperando mensajes de usuario", "user_id", userID)

	// 1. Obtener mensajes donde el usuario es el remitente o el destinatario
	query := `
		SELECT id, sender_id, receiver_id, content, read_at, created_at
		FROM messages
		WHERE sender_id = $1 OR receiver_id = $1
		ORDER BY created_at ASC;
	`
	rows, err := r.db.Query(dbCtx, query, userID)
	if err != nil {
		slog.Error("Error al listar mensajes desde DB", "error", err, "user_id", userID)
		return nil, fmt.Errorf("error al listar mensajes de la base de datos: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var m domain.Message
		err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.ReadAt, &m.CreatedAt)
		if err != nil {
			slog.Error("Error al escanear fila de mensaje", "error", err)
			return nil, fmt.Errorf("error al decodificar mensaje de la base de datos: %w", err)
		}
		messages = append(messages, m)
	}

	if messages == nil {
		messages = []domain.Message{}
	}

	// 2. Marcar automáticamente los recibidos no leídos como leídos en la base de datos
	updateQuery := `
		UPDATE messages
		SET read_at = NOW()
		WHERE receiver_id = $1 AND read_at IS NULL;
	`
	_, err = r.db.Exec(dbCtx, updateQuery, userID)
	if err != nil {
		slog.Warn("Fallo al actualizar read_at para mensajes recibidos", "error", err, "user_id", userID)
	}

	// 3. Modificar la respuesta cargada localmente para marcar los no leídos como leídos (consistencia inmediata en la respuesta)
	now := time.Now().UTC()
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

	slog.Debug("Contando mensajes no leídos para usuario", "user_id", userID)

	query := `
		SELECT COUNT(*)
		FROM messages
		WHERE receiver_id = $1 AND read_at IS NULL;
	`
	var count int
	err := r.db.QueryRow(dbCtx, query, userID).Scan(&count)
	if err != nil {
		slog.Error("Fallo al contar mensajes no leídos en PostgreSQL", "error", err, "user_id", userID)
		return 0, fmt.Errorf("error al obtener mensajes no leídos: %w", err)
	}

	return count, nil
}
