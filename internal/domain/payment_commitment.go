package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// PaymentCommitment representa un compromiso o recordatorio de pago.
type PaymentCommitment struct {
	ID         int64           `json:"id"`
	UserID     int64           `json:"user_id"`
	Amount     decimal.Decimal `json:"amount"`
	CurrencyID int64           `json:"currency_id"`
	DueDate    time.Time       `json:"due_date"`
	Status     string          `json:"status"` // Ej: PENDIENTE, CUMPLIDO, VENCIDO
	CreatedAt  time.Time       `json:"created_at"`
}

// PaymentCommitmentRepository define los métodos de persistencia para los compromisos de pago.
type PaymentCommitmentRepository interface {
	Create(ctx context.Context, pc *PaymentCommitment) error
	FindByID(ctx context.Context, id int64) (*PaymentCommitment, error)
	FindByUserID(ctx context.Context, userID int64) ([]PaymentCommitment, error)
	Update(ctx context.Context, pc *PaymentCommitment) error
	Delete(ctx context.Context, id int64, userID int64) error
}

// PaymentCommitmentUseCase define la lógica de negocio para los compromisos de pago.
type PaymentCommitmentUseCase interface {
	Create(ctx context.Context, userID int64, amount decimal.Decimal, currencyID int64, dueDate time.Time, status string) (*PaymentCommitment, error)
	GetSegmentedCommitments(ctx context.Context, userID int64) (map[string][]PaymentCommitment, error)
	Update(ctx context.Context, id, userID int64, amount decimal.Decimal, currencyID int64, dueDate time.Time, status string) (*PaymentCommitment, error)
	Delete(ctx context.Context, id int64, userID int64) error
}
