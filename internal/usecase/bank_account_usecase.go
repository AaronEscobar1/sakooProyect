package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/aaron/sakoo-backend/internal/domain"
)

type bankAccountUseCase struct {
	repo domain.BankAccountRepository
}

// NewBankAccountUseCase crea una nueva instancia de la lógica de negocio de cuentas bancarias.
func NewBankAccountUseCase(repo domain.BankAccountRepository) domain.BankAccountUseCase {
	return &bankAccountUseCase{
		repo: repo,
	}
}

// CreateOwn valida y crea una cuenta bancaria propia.
func (u *bankAccountUseCase) CreateOwn(ctx context.Context, userID int64, bankID int64, accNum, accType, holder string) (*domain.BankAccount, error) {
	accNum = strings.TrimSpace(accNum)
	accType = strings.TrimSpace(accType)
	holder = strings.TrimSpace(holder)

	if userID <= 0 {
		return nil, errors.New("ID de usuario inválido")
	}
	if bankID <= 0 || accNum == "" || accType == "" || holder == "" {
		return nil, errors.New("Los campos bank_id, account_number, account_type y holder_name son requeridos")
	}

	if len(accNum) < 10 {
		return nil, errors.New("El número de cuenta debe tener al menos 10 caracteres")
	}

	acc := &domain.BankAccount{
		UserID:        userID,
		BankID:        bankID,
		AccountNumber: accNum,
		AccountType:   accType,
		HolderName:    holder,
	}

	if err := u.repo.CreateOwn(ctx, acc); err != nil {
		return nil, err
	}

	return acc, nil
}

// ListOwn obtiene el listado de cuentas bancarias propias del usuario.
func (u *bankAccountUseCase) ListOwn(ctx context.Context, userID int64) ([]domain.BankAccount, error) {
	if userID <= 0 {
		return nil, errors.New("ID de usuario inválido")
	}
	return u.repo.ListOwn(ctx, userID)
}

// UpdateOwn valida y actualiza una cuenta bancaria propia.
func (u *bankAccountUseCase) UpdateOwn(ctx context.Context, id, userID int64, bankID int64, accNum, accType, holder string) (*domain.BankAccount, error) {
	accNum = strings.TrimSpace(accNum)
	accType = strings.TrimSpace(accType)
	holder = strings.TrimSpace(holder)

	if id <= 0 || userID <= 0 {
		return nil, errors.New("IDs de cuenta o usuario inválidos")
	}
	if bankID <= 0 || accNum == "" || accType == "" || holder == "" {
		return nil, errors.New("Los campos bank_id, account_number, account_type y holder_name son requeridos")
	}

	acc := &domain.BankAccount{
		ID:            id,
		UserID:        userID,
		BankID:        bankID,
		AccountNumber: accNum,
		AccountType:   accType,
		HolderName:    holder,
	}

	if err := u.repo.UpdateOwn(ctx, acc); err != nil {
		return nil, err
	}

	return acc, nil
}

// DeleteOwn elimina una cuenta propia si pertenece al usuario.
func (u *bankAccountUseCase) DeleteOwn(ctx context.Context, id int64, userID int64) error {
	if id <= 0 || userID <= 0 {
		return errors.New("IDs de cuenta o usuario inválidos")
	}
	return u.repo.DeleteOwn(ctx, id, userID)
}

// CreateThirdParty valida y crea una cuenta bancaria de terceros.
func (u *bankAccountUseCase) CreateThirdParty(ctx context.Context, userID int64, bankID int64, accNum, accType, holder, alias, docNum, phone string) (*domain.ThirdPartyAccount, error) {
	accNum = strings.TrimSpace(accNum)
	accType = strings.TrimSpace(accType)
	holder = strings.TrimSpace(holder)
	alias = strings.TrimSpace(alias)
	docNum = strings.TrimSpace(docNum)
	phone = strings.TrimSpace(phone)

	if userID <= 0 {
		return nil, errors.New("ID de usuario inválido")
	}
	if bankID <= 0 || accNum == "" || accType == "" || holder == "" || alias == "" || docNum == "" {
		return nil, errors.New("Los campos bank_id, account_number, account_type, holder_name, alias y document_number son requeridos")
	}

	if len(accNum) < 10 {
		return nil, errors.New("El número de cuenta debe tener al menos 10 caracteres")
	}

	acc := &domain.ThirdPartyAccount{
		UserID:         userID,
		BankID:         bankID,
		AccountNumber:  accNum,
		AccountType:    accType,
		HolderName:     holder,
		Alias:          alias,
		DocumentNumber: docNum,
		PhoneNumber:    phone,
	}

	if err := u.repo.CreateThirdParty(ctx, acc); err != nil {
		return nil, err
	}

	return acc, nil
}

// ListThirdParty obtiene las cuentas de terceros de un usuario.
func (u *bankAccountUseCase) ListThirdParty(ctx context.Context, userID int64) ([]domain.ThirdPartyAccount, error) {
	if userID <= 0 {
		return nil, errors.New("ID de usuario inválido")
	}
	return u.repo.ListThirdParty(ctx, userID)
}

// UpdateThirdParty valida y actualiza una cuenta de terceros.
func (u *bankAccountUseCase) UpdateThirdParty(ctx context.Context, id, userID int64, bankID int64, accNum, accType, holder, alias, docNum, phone string) (*domain.ThirdPartyAccount, error) {
	accNum = strings.TrimSpace(accNum)
	accType = strings.TrimSpace(accType)
	holder = strings.TrimSpace(holder)
	alias = strings.TrimSpace(alias)
	docNum = strings.TrimSpace(docNum)
	phone = strings.TrimSpace(phone)

	if id <= 0 || userID <= 0 {
		return nil, errors.New("IDs de cuenta o usuario inválidos")
	}
	if bankID <= 0 || accNum == "" || accType == "" || holder == "" || alias == "" || docNum == "" {
		return nil, errors.New("Los campos bank_id, account_number, account_type, holder_name, alias y document_number son requeridos")
	}

	acc := &domain.ThirdPartyAccount{
		ID:             id,
		UserID:         userID,
		BankID:         bankID,
		AccountNumber:  accNum,
		AccountType:    accType,
		HolderName:     holder,
		Alias:          alias,
		DocumentNumber: docNum,
		PhoneNumber:    phone,
	}

	if err := u.repo.UpdateThirdParty(ctx, acc); err != nil {
		return nil, err
	}

	return acc, nil
}

// DeleteThirdParty elimina una cuenta de terceros si pertenece al usuario.
func (u *bankAccountUseCase) DeleteThirdParty(ctx context.Context, id int64, userID int64) error {
	if id <= 0 || userID <= 0 {
		return errors.New("IDs de cuenta o usuario inválidos")
	}
	return u.repo.DeleteThirdParty(ctx, id, userID)
}
