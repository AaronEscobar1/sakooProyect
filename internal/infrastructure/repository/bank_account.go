package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type bankAccountRepository struct {
	db *pgxpool.Pool
}

// NewBankAccountRepository crea una nueva instancia del repositorio de cuentas bancarias.
func NewBankAccountRepository(db *pgxpool.Pool) domain.BankAccountRepository {
	return &bankAccountRepository{
		db: db,
	}
}

// CreateOwn registra una cuenta bancaria propia en PostgreSQL.
func (r *bankAccountRepository) CreateOwn(ctx context.Context, acc *domain.BankAccount) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando cuenta propia en base de datos", "user_id", acc.UserID, "bank_name", acc.BankName)

	query := `
		INSERT INTO public.bank_accounts (
			user_id, 
			bank_name, 
			account_number, 
			account_type, 
			holder_name, 
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at;
	`

	err := r.db.QueryRow(dbCtx, query,
		acc.UserID,
		acc.BankName,
		acc.AccountNumber,
		acc.AccountType,
		acc.HolderName,
	).Scan(&acc.ID, &acc.CreatedAt, &acc.UpdatedAt)

	if err != nil {
		slog.Error("Fallo al insertar cuenta propia en PostgreSQL", "error", err, "user_id", acc.UserID)
		return fmt.Errorf("error al crear cuenta propia: %w", err)
	}

	return nil
}

// ListOwn obtiene el listado de cuentas propias de un usuario.
func (r *bankAccountRepository) ListOwn(ctx context.Context, userID int64) ([]domain.BankAccount, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando cuentas propias en base de datos", "user_id", userID)

	query := `
		SELECT id, user_id, bank_name, account_number, account_type, holder_name, created_at, updated_at
		FROM public.bank_accounts
		WHERE user_id = $1
		ORDER BY created_at DESC;
	`

	rows, err := r.db.Query(dbCtx, query, userID)
	if err != nil {
		slog.Error("Fallo al consultar cuentas propias", "error", err, "user_id", userID)
		return nil, fmt.Errorf("error al listar cuentas propias: %w", err)
	}
	defer rows.Close()

	var accounts []domain.BankAccount
	for rows.Next() {
		var acc domain.BankAccount
		err := rows.Scan(
			&acc.ID,
			&acc.UserID,
			&acc.BankName,
			&acc.AccountNumber,
			&acc.AccountType,
			&acc.HolderName,
			&acc.CreatedAt,
			&acc.UpdatedAt,
		)
		if err != nil {
			slog.Error("Fallo al escanear cuenta propia", "error", err)
			return nil, fmt.Errorf("error al escanear cuenta propia: %w", err)
		}
		accounts = append(accounts, acc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de cuentas propias: %w", err)
	}

	if accounts == nil {
		accounts = []domain.BankAccount{}
	}

	return accounts, nil
}

// UpdateOwn actualiza los datos de una cuenta propia.
func (r *bankAccountRepository) UpdateOwn(ctx context.Context, acc *domain.BankAccount) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Actualizando cuenta propia en base de datos", "id", acc.ID, "user_id", acc.UserID)

	query := `
		UPDATE public.bank_accounts
		SET bank_name = $1, account_number = $2, account_type = $3, holder_name = $4, updated_at = NOW()
		WHERE id = $5 AND user_id = $6
		RETURNING updated_at;
	`

	err := r.db.QueryRow(dbCtx, query,
		acc.BankName,
		acc.AccountNumber,
		acc.AccountType,
		acc.HolderName,
		acc.ID,
		acc.UserID,
	).Scan(&acc.UpdatedAt)

	if err != nil {
		slog.Error("Fallo al actualizar cuenta propia en PostgreSQL", "error", err, "id", acc.ID)
		return fmt.Errorf("error al actualizar cuenta propia: %w", err)
	}

	return nil
}

// DeleteOwn elimina lógicamente o físicamente una cuenta propia de la base de datos.
func (r *bankAccountRepository) DeleteOwn(ctx context.Context, id int64, userID int64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Eliminando cuenta propia de base de datos", "id", id, "user_id", userID)

	query := `
		DELETE FROM public.bank_accounts
		WHERE id = $1 AND user_id = $2;
	`

	res, err := r.db.Exec(dbCtx, query, id, userID)
	if err != nil {
		slog.Error("Fallo al eliminar cuenta propia en PostgreSQL", "error", err, "id", id)
		return fmt.Errorf("error al eliminar cuenta propia: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("cuenta propia no encontrada o no pertenece al usuario")
	}

	return nil
}

// CreateThirdParty registra una cuenta de terceros en PostgreSQL.
func (r *bankAccountRepository) CreateThirdParty(ctx context.Context, acc *domain.ThirdPartyAccount) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando cuenta de terceros en base de datos", "user_id", acc.UserID, "bank_name", acc.BankName)

	query := `
		INSERT INTO public.third_party_accounts (
			user_id, 
			bank_name, 
			account_number, 
			account_type, 
			holder_name, 
			alias,
			document_number,
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at;
	`

	err := r.db.QueryRow(dbCtx, query,
		acc.UserID,
		acc.BankName,
		acc.AccountNumber,
		acc.AccountType,
		acc.HolderName,
		acc.Alias,
		acc.DocumentNumber,
	).Scan(&acc.ID, &acc.CreatedAt, &acc.UpdatedAt)

	if err != nil {
		slog.Error("Fallo al insertar cuenta de terceros en PostgreSQL", "error", err, "user_id", acc.UserID)
		return fmt.Errorf("error al crear cuenta de terceros: %w", err)
	}

	return nil
}

// ListThirdParty obtiene el listado de cuentas de terceros asociadas a un usuario.
func (r *bankAccountRepository) ListThirdParty(ctx context.Context, userID int64) ([]domain.ThirdPartyAccount, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando cuentas de terceros en base de datos", "user_id", userID)

	query := `
		SELECT id, user_id, bank_name, account_number, account_type, holder_name, alias, document_number, created_at, updated_at
		FROM public.third_party_accounts
		WHERE user_id = $1
		ORDER BY created_at DESC;
	`

	rows, err := r.db.Query(dbCtx, query, userID)
	if err != nil {
		slog.Error("Fallo al consultar cuentas de terceros", "error", err, "user_id", userID)
		return nil, fmt.Errorf("error al listar cuentas de terceros: %w", err)
	}
	defer rows.Close()

	var accounts []domain.ThirdPartyAccount
	for rows.Next() {
		var acc domain.ThirdPartyAccount
		err := rows.Scan(
			&acc.ID,
			&acc.UserID,
			&acc.BankName,
			&acc.AccountNumber,
			&acc.AccountType,
			&acc.HolderName,
			&acc.Alias,
			&acc.DocumentNumber,
			&acc.CreatedAt,
			&acc.UpdatedAt,
		)
		if err != nil {
			slog.Error("Fallo al escanear cuenta de terceros", "error", err)
			return nil, fmt.Errorf("error al escanear cuenta de terceros: %w", err)
		}
		accounts = append(accounts, acc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de cuentas de terceros: %w", err)
	}

	if accounts == nil {
		accounts = []domain.ThirdPartyAccount{}
	}

	return accounts, nil
}

// UpdateThirdParty actualiza los datos de una cuenta de terceros.
func (r *bankAccountRepository) UpdateThirdParty(ctx context.Context, acc *domain.ThirdPartyAccount) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Actualizando cuenta de terceros en base de datos", "id", acc.ID, "user_id", acc.UserID)

	query := `
		UPDATE public.third_party_accounts
		SET bank_name = $1, account_number = $2, account_type = $3, holder_name = $4, alias = $5, document_number = $6, updated_at = NOW()
		WHERE id = $7 AND user_id = $8
		RETURNING updated_at;
	`

	err := r.db.QueryRow(dbCtx, query,
		acc.BankName,
		acc.AccountNumber,
		acc.AccountType,
		acc.HolderName,
		acc.Alias,
		acc.DocumentNumber,
		acc.ID,
		acc.UserID,
	).Scan(&acc.UpdatedAt)

	if err != nil {
		slog.Error("Fallo al actualizar cuenta de terceros en PostgreSQL", "error", err, "id", acc.ID)
		return fmt.Errorf("error al actualizar cuenta de terceros: %w", err)
	}

	return nil
}

// DeleteThirdParty elimina una cuenta de terceros de la base de datos.
func (r *bankAccountRepository) DeleteThirdParty(ctx context.Context, id int64, userID int64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Eliminando cuenta de terceros de base de datos", "id", id, "user_id", userID)

	query := `
		DELETE FROM public.third_party_accounts
		WHERE id = $1 AND user_id = $2;
	`

	res, err := r.db.Exec(dbCtx, query, id, userID)
	if err != nil {
		slog.Error("Fallo al eliminar cuenta de terceros en PostgreSQL", "error", err, "id", id)
		return fmt.Errorf("error al eliminar cuenta de terceros: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("cuenta de terceros no encontrada o no pertenece al usuario")
	}

	return nil
}
