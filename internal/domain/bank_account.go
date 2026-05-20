package domain

import (
	"context"
	"time"
)

// BankAccount representa una cuenta bancaria propia del usuario.
type BankAccount struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	BankName      string    `json:"bank_name"`
	AccountNumber string    `json:"account_number"`
	AccountType   string    `json:"account_type"`
	HolderName    string    `json:"holder_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ThirdPartyAccount representa una cuenta de terceros registrada por el usuario.
type ThirdPartyAccount struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	BankName       string    `json:"bank_name"`
	AccountNumber  string    `json:"account_number"`
	AccountType    string    `json:"account_type"`
	HolderName     string    `json:"holder_name"`
	Alias          string    `json:"alias"`
	DocumentNumber string    `json:"document_number"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// BankAccountRepository define los métodos de persistencia para las cuentas en la base de datos.
type BankAccountRepository interface {
	// Cuentas propias
	CreateOwn(ctx context.Context, acc *BankAccount) error
	ListOwn(ctx context.Context, userID int64) ([]BankAccount, error)
	UpdateOwn(ctx context.Context, acc *BankAccount) error
	DeleteOwn(ctx context.Context, id int64, userID int64) error

	// Cuentas de terceros
	CreateThirdParty(ctx context.Context, acc *ThirdPartyAccount) error
	ListThirdParty(ctx context.Context, userID int64) ([]ThirdPartyAccount, error)
	UpdateThirdParty(ctx context.Context, acc *ThirdPartyAccount) error
	DeleteThirdParty(ctx context.Context, id int64, userID int64) error
}

// BankAccountUseCase define la lógica de negocio para gestionar cuentas bancarias.
type BankAccountUseCase interface {
	// Cuentas propias
	CreateOwn(ctx context.Context, userID int64, bankName, accNum, accType, holder string) (*BankAccount, error)
	ListOwn(ctx context.Context, userID int64) ([]BankAccount, error)
	UpdateOwn(ctx context.Context, id, userID int64, bankName, accNum, accType, holder string) (*BankAccount, error)
	DeleteOwn(ctx context.Context, id int64, userID int64) error

	// Cuentas de terceros
	CreateThirdParty(ctx context.Context, userID int64, bankName, accNum, accType, holder, alias, docNum string) (*ThirdPartyAccount, error)
	ListThirdParty(ctx context.Context, userID int64) ([]ThirdPartyAccount, error)
	UpdateThirdParty(ctx context.Context, id, userID int64, bankName, accNum, accType, holder, alias, docNum string) (*ThirdPartyAccount, error)
	DeleteThirdParty(ctx context.Context, id int64, userID int64) error
}
