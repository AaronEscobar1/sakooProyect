package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/bankaccount"
	"github.com/aaron/sakoo-backend/ent/thirdpartyaccount"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type bankAccountRepository struct {
	client *ent.Client
}

// NewBankAccountRepository crea una nueva instancia del repositorio de cuentas bancarias con Ent.
func NewBankAccountRepository(client *ent.Client) domain.BankAccountRepository {
	return &bankAccountRepository{
		client: client,
	}
}

func (r *bankAccountRepository) CreateOwn(ctx context.Context, acc *domain.BankAccount) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando cuenta propia en Ent", "user_id", acc.UserID, "bank_id", acc.BankID)

	created, err := r.client.BankAccount.Create().
		SetUserID(acc.UserID).
		SetAccountNumber(acc.AccountNumber).
		SetAccountType(acc.AccountType).
		SetHolderName(acc.HolderName).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al insertar cuenta propia en Ent", "error", err, "user_id", acc.UserID)
		return fmt.Errorf("error al crear cuenta propia: %w", err)
	}

	acc.ID = int64(created.ID)
	acc.CreatedAt = created.CreatedAt
	acc.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *bankAccountRepository) ListOwn(ctx context.Context, userID int64) ([]domain.BankAccount, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando cuentas propias en Ent", "user_id", userID)

	accounts, err := r.client.BankAccount.Query().
		Where(bankaccount.UserID(userID)).
		Order(ent.Desc(bankaccount.FieldCreatedAt)).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al consultar cuentas propias", "error", err, "user_id", userID)
		return nil, fmt.Errorf("error al listar cuentas propias: %w", err)
	}

	var result []domain.BankAccount
	for _, acc := range accounts {
		result = append(result, domain.BankAccount{
			ID:            int64(acc.ID),
			UserID:        acc.UserID,
			AccountNumber: acc.AccountNumber,
			AccountType:   acc.AccountType,
			HolderName:    acc.HolderName,
			CreatedAt:     acc.CreatedAt,
			UpdatedAt:     acc.UpdatedAt,
		})
	}

	if result == nil {
		result = []domain.BankAccount{}
	}

	return result, nil
}

func (r *bankAccountRepository) UpdateOwn(ctx context.Context, acc *domain.BankAccount) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Actualizando cuenta propia en Ent", "id", acc.ID, "user_id", acc.UserID)

	updated, err := r.client.BankAccount.UpdateOneID(int(acc.ID)).
		Where(bankaccount.UserID(acc.UserID)).
		SetAccountNumber(acc.AccountNumber).
		SetAccountType(acc.AccountType).
		SetHolderName(acc.HolderName).
		SetUpdatedAt(time.Now()).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al actualizar cuenta propia en Ent", "error", err, "id", acc.ID)
		return fmt.Errorf("error al actualizar cuenta propia: %w", err)
	}

	acc.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *bankAccountRepository) DeleteOwn(ctx context.Context, id int64, userID int64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Eliminando cuenta propia en Ent", "id", id, "user_id", userID)

	count, err := r.client.BankAccount.Delete().
		Where(
			bankaccount.IDEQ(int(id)),
			bankaccount.UserID(userID),
		).
		Exec(dbCtx)

	if err != nil {
		slog.Error("Fallo al eliminar cuenta propia en Ent", "error", err, "id", id)
		return fmt.Errorf("error al eliminar cuenta propia: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("cuenta propia no encontrada o no pertenece al usuario")
	}

	return nil
}

func (r *bankAccountRepository) CreateThirdParty(ctx context.Context, acc *domain.ThirdPartyAccount) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando cuenta de terceros en Ent", "user_id", acc.UserID)

	created, err := r.client.ThirdPartyAccount.Create().
		SetUserID(acc.UserID).
		SetAccountNumber(acc.AccountNumber).
		SetAccountType(acc.AccountType).
		SetHolderName(acc.HolderName).
		SetAlias(acc.Alias).
		SetDocumentNumber(acc.DocumentNumber).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al insertar cuenta de terceros en Ent", "error", err, "user_id", acc.UserID)
		return fmt.Errorf("error al crear cuenta de terceros: %w", err)
	}

	acc.ID = int64(created.ID)
	acc.CreatedAt = created.CreatedAt
	acc.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *bankAccountRepository) ListThirdParty(ctx context.Context, userID int64) ([]domain.ThirdPartyAccount, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando cuentas de terceros en Ent", "user_id", userID)

	accounts, err := r.client.ThirdPartyAccount.Query().
		Where(thirdpartyaccount.UserID(userID)).
		Order(ent.Desc(thirdpartyaccount.FieldCreatedAt)).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al consultar cuentas de terceros", "error", err, "user_id", userID)
		return nil, fmt.Errorf("error al listar cuentas de terceros: %w", err)
	}

	var result []domain.ThirdPartyAccount
	for _, acc := range accounts {
		result = append(result, domain.ThirdPartyAccount{
			ID:             int64(acc.ID),
			UserID:         acc.UserID,
			AccountNumber:  acc.AccountNumber,
			AccountType:    acc.AccountType,
			HolderName:     acc.HolderName,
			Alias:          acc.Alias,
			DocumentNumber: acc.DocumentNumber,
			CreatedAt:      acc.CreatedAt,
			UpdatedAt:      acc.UpdatedAt,
		})
	}

	if result == nil {
		result = []domain.ThirdPartyAccount{}
	}

	return result, nil
}

func (r *bankAccountRepository) UpdateThirdParty(ctx context.Context, acc *domain.ThirdPartyAccount) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Actualizando cuenta de terceros en Ent", "id", acc.ID, "user_id", acc.UserID)

	updated, err := r.client.ThirdPartyAccount.UpdateOneID(int(acc.ID)).
		Where(thirdpartyaccount.UserID(acc.UserID)).
		SetAccountNumber(acc.AccountNumber).
		SetAccountType(acc.AccountType).
		SetHolderName(acc.HolderName).
		SetAlias(acc.Alias).
		SetDocumentNumber(acc.DocumentNumber).
		SetUpdatedAt(time.Now()).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al actualizar cuenta de terceros en Ent", "error", err, "id", acc.ID)
		return fmt.Errorf("error al actualizar cuenta de terceros: %w", err)
	}

	acc.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *bankAccountRepository) DeleteThirdParty(ctx context.Context, id int64, userID int64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Eliminando cuenta de terceros en Ent", "id", id, "user_id", userID)

	count, err := r.client.ThirdPartyAccount.Delete().
		Where(
			thirdpartyaccount.IDEQ(int(id)),
			thirdpartyaccount.UserID(userID),
		).
		Exec(dbCtx)

	if err != nil {
		slog.Error("Fallo al eliminar cuenta de terceros en Ent", "error", err, "id", id)
		return fmt.Errorf("error al eliminar cuenta de terceros: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("cuenta de terceros no encontrada o no pertenece al usuario")
	}

	return nil
}
