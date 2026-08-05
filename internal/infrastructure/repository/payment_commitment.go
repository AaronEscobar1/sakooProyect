package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/paymentcommitment"
	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/shopspring/decimal"
)

type paymentCommitmentRepository struct {
	client *ent.Client
}

// NewPaymentCommitmentRepository crea una nueva instancia del repositorio de compromisos de pago usando Ent.
func NewPaymentCommitmentRepository(client *ent.Client) domain.PaymentCommitmentRepository {
	return &paymentCommitmentRepository{
		client: client,
	}
}

func (r *paymentCommitmentRepository) Create(ctx context.Context, pc *domain.PaymentCommitment) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando compromiso de pago en Ent", "user_id", pc.UserID, "amount", pc.Amount)

	amtFloat, _ := pc.Amount.Float64()

	builder := r.client.PaymentCommitment.Create().
		SetAmount(amtFloat).
		SetDueDate(pc.DueDate).
		SetStatus(pc.Status)

	if pc.UserID != 0 {
		builder.SetUserID(pc.UserID)
	}
	if pc.CurrencyID != 0 {
		builder.SetCurrencyID(pc.CurrencyID)
	}

	created, err := builder.Save(dbCtx)
	if err != nil {
		slog.Error("Fallo al insertar compromiso de pago en Ent", "error", err)
		return fmt.Errorf("error al crear compromiso de pago: %w", err)
	}

	pc.ID = int64(created.ID)
	pc.CreatedAt = created.CreatedAt
	return nil
}

func (r *paymentCommitmentRepository) FindByID(ctx context.Context, id int64) (*domain.PaymentCommitment, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Buscando compromiso de pago por ID", "id", id)

	pc, err := r.client.PaymentCommitment.Query().
		Where(paymentcommitment.IDEQ(int(id))).
		Only(dbCtx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("compromiso de pago no encontrado: %w", err)
		}
		slog.Error("Error al consultar compromiso de pago por ID", "error", err, "id", id)
		return nil, fmt.Errorf("error al buscar compromiso de pago: %w", err)
	}

	return &domain.PaymentCommitment{
		ID:         int64(pc.ID),
		UserID:     pc.UserID,
		Amount:     decimal.NewFromFloat(pc.Amount),
		CurrencyID: pc.CurrencyID,
		DueDate:    pc.DueDate,
		Status:     pc.Status,
		CreatedAt:  pc.CreatedAt,
	}, nil
}

func (r *paymentCommitmentRepository) FindByUserID(ctx context.Context, userID int64) ([]domain.PaymentCommitment, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando compromisos de pago por usuario en Ent", "user_id", userID)

	pcs, err := r.client.PaymentCommitment.Query().
		Where(paymentcommitment.UserID(userID)).
		Order(ent.Asc(paymentcommitment.FieldDueDate)).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al consultar compromisos de pago", "error", err, "user_id", userID)
		return nil, fmt.Errorf("error al listar compromisos de pago: %w", err)
	}

	var commitments []domain.PaymentCommitment
	for _, pc := range pcs {
		commitments = append(commitments, domain.PaymentCommitment{
			ID:         int64(pc.ID),
			UserID:     pc.UserID,
			Amount:     decimal.NewFromFloat(pc.Amount),
			CurrencyID: pc.CurrencyID,
			DueDate:    pc.DueDate,
			Status:     pc.Status,
			CreatedAt:  pc.CreatedAt,
		})
	}

	if commitments == nil {
		commitments = []domain.PaymentCommitment{}
	}

	return commitments, nil
}

func (r *paymentCommitmentRepository) Update(ctx context.Context, pc *domain.PaymentCommitment) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Actualizando compromiso de pago en Ent", "id", pc.ID)

	amtFloat, _ := pc.Amount.Float64()

	builder := r.client.PaymentCommitment.UpdateOneID(int(pc.ID)).
		SetAmount(amtFloat).
		SetDueDate(pc.DueDate).
		SetStatus(pc.Status)

	if pc.UserID != 0 {
		builder.Where(paymentcommitment.UserID(pc.UserID))
	}

	_, err := builder.Save(dbCtx)
	if err != nil {
		slog.Error("Fallo al actualizar compromiso de pago en Ent", "error", err, "id", pc.ID)
		return fmt.Errorf("error al actualizar compromiso de pago: %w", err)
	}

	return nil
}

func (r *paymentCommitmentRepository) Delete(ctx context.Context, id int64, userID int64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Eliminando compromiso de pago en Ent", "id", id, "user_id", userID)

	count, err := r.client.PaymentCommitment.Delete().
		Where(
			paymentcommitment.IDEQ(int(id)),
			paymentcommitment.UserID(userID),
		).
		Exec(dbCtx)

	if err != nil {
		slog.Error("Fallo al eliminar compromiso de pago en Ent", "error", err, "id", id)
		return fmt.Errorf("error al eliminar compromiso de pago: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("compromiso de pago no encontrado o no pertenece al usuario")
	}

	return nil
}
