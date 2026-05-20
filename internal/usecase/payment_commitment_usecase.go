package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/shopspring/decimal"
)

type paymentCommitmentUseCase struct {
	repo domain.PaymentCommitmentRepository
}

// NewPaymentCommitmentUseCase crea una nueva instancia del caso de uso de compromisos de pago.
func NewPaymentCommitmentUseCase(repo domain.PaymentCommitmentRepository) domain.PaymentCommitmentUseCase {
	return &paymentCommitmentUseCase{
		repo: repo,
	}
}

// Create valida y registra un nuevo compromiso de pago.
func (u *paymentCommitmentUseCase) Create(ctx context.Context, userID int64, amount decimal.Decimal, currencyID int64, dueDate time.Time, status string) (*domain.PaymentCommitment, error) {
	status = strings.TrimSpace(strings.ToUpper(status))

	if userID <= 0 {
		return nil, errors.New("ID de usuario inválido")
	}
	if amount.IsNegative() || amount.IsZero() {
		return nil, errors.New("el monto del compromiso de pago debe ser positivo")
	}
	if currencyID <= 0 {
		return nil, errors.New("ID de moneda inválido")
	}
	if dueDate.IsZero() {
		return nil, errors.New("la fecha límite de pago es requerida")
	}
	if status == "" {
		status = "PENDIENTE"
	}

	pc := &domain.PaymentCommitment{
		UserID:     userID,
		Amount:     amount,
		CurrencyID: currencyID,
		DueDate:    dueDate,
		Status:     status,
	}

	if err := u.repo.Create(ctx, pc); err != nil {
		return nil, err
	}

	return pc, nil
}

// GetSegmentedCommitments obtiene los compromisos del usuario y los divide en por_vencer, vencidos y cumplidos.
func (u *paymentCommitmentUseCase) GetSegmentedCommitments(ctx context.Context, userID int64) (map[string][]domain.PaymentCommitment, error) {
	if userID <= 0 {
		return nil, errors.New("ID de usuario inválido")
	}

	commitments, err := u.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	segmented := map[string][]domain.PaymentCommitment{
		"por_vencer": {},
		"vencidos":   {},
		"cumplidos":  {},
	}

	now := time.Now()
	// Obtener la fecha actual a las 00:00:00 para comparación justa basada únicamente en días
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, pc := range commitments {
		statusUpper := strings.ToUpper(pc.Status)

		// 1. Si está pagado/cumplido, se segmenta en "cumplidos"
		if statusUpper == "CUMPLIDO" || statusUpper == "PAGADO" || statusUpper == "COMPLETED" {
			segmented["cumplidos"] = append(segmented["cumplidos"], pc)
			continue
		}

		// 2. Si no está cumplido, comparar fecha límite vs hoy a medianoche
		dueMidnight := time.Date(pc.DueDate.Year(), pc.DueDate.Month(), pc.DueDate.Day(), 0, 0, 0, 0, pc.DueDate.Location())

		if dueMidnight.Before(todayMidnight) {
			segmented["vencidos"] = append(segmented["vencidos"], pc)
		} else {
			segmented["por_vencer"] = append(segmented["por_vencer"], pc)
		}
	}

	return segmented, nil
}

// Update valida y actualiza un compromiso de pago existente.
func (u *paymentCommitmentUseCase) Update(ctx context.Context, id, userID int64, amount decimal.Decimal, currencyID int64, dueDate time.Time, status string) (*domain.PaymentCommitment, error) {
	status = strings.TrimSpace(strings.ToUpper(status))

	if id <= 0 || userID <= 0 {
		return nil, errors.New("IDs de cuenta o usuario inválidos")
	}
	if amount.IsNegative() || amount.IsZero() {
		return nil, errors.New("el monto debe ser un valor positivo")
	}
	if currencyID <= 0 {
		return nil, errors.New("ID de moneda inválido")
	}
	if dueDate.IsZero() {
		return nil, errors.New("la fecha límite de pago es requerida")
	}
	if status == "" {
		status = "PENDIENTE"
	}

	pc := &domain.PaymentCommitment{
		ID:         id,
		UserID:     userID,
		Amount:     amount,
		CurrencyID: currencyID,
		DueDate:    dueDate,
		Status:     status,
	}

	if err := u.repo.Update(ctx, pc); err != nil {
		return nil, err
	}

	return pc, nil
}

// Delete elimina un compromiso de pago.
func (u *paymentCommitmentUseCase) Delete(ctx context.Context, id int64, userID int64) error {
	if id <= 0 || userID <= 0 {
		return errors.New("IDs de cuenta o usuario inválidos")
	}
	return u.repo.Delete(ctx, id, userID)
}
