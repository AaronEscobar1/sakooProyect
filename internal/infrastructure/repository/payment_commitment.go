package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type paymentCommitmentRepository struct {
	db *pgxpool.Pool
}

// NewPaymentCommitmentRepository crea una nueva instancia del repositorio de compromisos de pago.
func NewPaymentCommitmentRepository(db *pgxpool.Pool) domain.PaymentCommitmentRepository {
	return &paymentCommitmentRepository{
		db: db,
	}
}

// Create registra un compromiso de pago en PostgreSQL.
func (r *paymentCommitmentRepository) Create(ctx context.Context, pc *domain.PaymentCommitment) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando compromiso de pago en base de datos", "user_id", pc.UserID, "amount", pc.Amount)

	query := `
		INSERT INTO public.payment_commitments (
			user_id, 
			amount, 
			currency_id, 
			due_date, 
			status, 
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, created_at;
	`

	err := r.db.QueryRow(dbCtx, query,
		pc.UserID,
		pc.Amount,
		pc.CurrencyID,
		pc.DueDate,
		pc.Status,
	).Scan(&pc.ID, &pc.CreatedAt)

	if err != nil {
		slog.Error("Fallo al insertar compromiso de pago en PostgreSQL", "error", err, "user_id", pc.UserID)
		return fmt.Errorf("error al crear compromiso de pago: %w", err)
	}

	return nil
}

// FindByID obtiene un compromiso de pago por su ID.
func (r *paymentCommitmentRepository) FindByID(ctx context.Context, id int64) (*domain.PaymentCommitment, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Buscando compromiso de pago por ID", "id", id)

	query := `
		SELECT id, user_id, amount, currency_id, due_date, status, created_at
		FROM public.payment_commitments
		WHERE id = $1;
	`

	var pc domain.PaymentCommitment
	err := r.db.QueryRow(dbCtx, query, id).Scan(
		&pc.ID,
		&pc.UserID,
		&pc.Amount,
		&pc.CurrencyID,
		&pc.DueDate,
		&pc.Status,
		&pc.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("compromiso de pago no encontrado: %w", pgx.ErrNoRows)
		}
		slog.Error("Error al consultar compromiso de pago por ID", "error", err, "id", id)
		return nil, fmt.Errorf("error al buscar compromiso de pago: %w", err)
	}

	return &pc, nil
}

// FindByUserID obtiene el listado de compromisos de pago asociados a un usuario.
func (r *paymentCommitmentRepository) FindByUserID(ctx context.Context, userID int64) ([]domain.PaymentCommitment, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando compromisos de pago por usuario en base de datos", "user_id", userID)

	query := `
		SELECT id, user_id, amount, currency_id, due_date, status, created_at
		FROM public.payment_commitments
		WHERE user_id = $1
		ORDER BY due_date ASC;
	`

	rows, err := r.db.Query(dbCtx, query, userID)
	if err != nil {
		slog.Error("Fallo al consultar compromisos de pago", "error", err, "user_id", userID)
		return nil, fmt.Errorf("error al listar compromisos de pago: %w", err)
	}
	defer rows.Close()

	var commitments []domain.PaymentCommitment
	for rows.Next() {
		var pc domain.PaymentCommitment
		err := rows.Scan(
			&pc.ID,
			&pc.UserID,
			&pc.Amount,
			&pc.CurrencyID,
			&pc.DueDate,
			&pc.Status,
			&pc.CreatedAt,
		)
		if err != nil {
			slog.Error("Fallo al escanear compromiso de pago", "error", err)
			return nil, fmt.Errorf("error al escanear compromiso de pago: %w", err)
		}
		commitments = append(commitments, pc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de compromisos de pago: %w", err)
	}

	if commitments == nil {
		commitments = []domain.PaymentCommitment{}
	}

	return commitments, nil
}

// Update actualiza los datos de un compromiso de pago.
func (r *paymentCommitmentRepository) Update(ctx context.Context, pc *domain.PaymentCommitment) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Actualizando compromiso de pago en base de datos", "id", pc.ID, "user_id", pc.UserID)

	query := `
		UPDATE public.payment_commitments
		SET amount = $1, currency_id = $2, due_date = $3, status = $4
		WHERE id = $5 AND user_id = $6;
	`

	res, err := r.db.Exec(dbCtx, query,
		pc.Amount,
		pc.CurrencyID,
		pc.DueDate,
		pc.Status,
		pc.ID,
		pc.UserID,
	)

	if err != nil {
		slog.Error("Fallo al actualizar compromiso de pago en PostgreSQL", "error", err, "id", pc.ID)
		return fmt.Errorf("error al actualizar compromiso de pago: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("compromiso de pago no encontrado o no pertenece al usuario")
	}

	return nil
}

// Delete elimina un compromiso de pago de la base de datos.
func (r *paymentCommitmentRepository) Delete(ctx context.Context, id int64, userID int64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Eliminando compromiso de pago de base de datos", "id", id, "user_id", userID)

	query := `
		DELETE FROM public.payment_commitments
		WHERE id = $1 AND user_id = $2;
	`

	res, err := r.db.Exec(dbCtx, query, id, userID)
	if err != nil {
		slog.Error("Fallo al eliminar compromiso de pago en PostgreSQL", "error", err, "id", id)
		return fmt.Errorf("error al eliminar compromiso de pago: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("compromiso de pago no encontrado o no pertenece al usuario")
	}

	return nil
}
