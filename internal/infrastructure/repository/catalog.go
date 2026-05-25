package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type catalogRepository struct {
	db *pgxpool.Pool
}

// NewCatalogRepository crea un repositorio para catálogos del sistema.
func NewCatalogRepository(db *pgxpool.Pool) domain.CatalogRepository {
	return &catalogRepository{
		db: db,
	}
}

func (r *catalogRepository) GetDocumentTypes(ctx context.Context) ([]domain.DocumentType, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Recuperando tipos de documento de la base de datos")

	query := `
		SELECT id, code, name, display_order, created_at
		FROM catalogs.document_type
		ORDER BY display_order ASC, name ASC;
	`
	rows, err := r.db.Query(dbCtx, query)
	if err != nil {
		slog.Error("Fallo al listar tipos de documento en catalogs", "error", err)
		return nil, fmt.Errorf("error al listar tipos de documento: %w", err)
	}
	defer rows.Close()

	var docTypes []domain.DocumentType
	for rows.Next() {
		var dt domain.DocumentType
		err := rows.Scan(&dt.ID, &dt.Code, &dt.Name, &dt.DisplayOrder, &dt.CreatedAt)
		if err != nil {
			slog.Error("Error al escanear fila de tipo de documento", "error", err)
			return nil, fmt.Errorf("error al decodificar tipo de documento: %w", err)
		}
		docTypes = append(docTypes, dt)
	}

	if docTypes == nil {
		docTypes = []domain.DocumentType{}
	}

	return docTypes, nil
}

func (r *catalogRepository) GetCurrencies(ctx context.Context) ([]domain.Currency, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Recuperando monedas de la base de datos")

	query := `
		SELECT id, code, name, display_order, created_at, updated_at
		FROM catalogs.currency
		WHERE "show" = TRUE
		ORDER BY display_order ASC, name ASC;
	`
	rows, err := r.db.Query(dbCtx, query)
	if err != nil {
		slog.Error("Fallo al listar monedas en catalogs", "error", err)
		return nil, fmt.Errorf("error al listar monedas: %w", err)
	}
	defer rows.Close()

	var currencies []domain.Currency
	for rows.Next() {
		var c domain.Currency
		err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			slog.Error("Error al escanear fila de moneda", "error", err)
			return nil, fmt.Errorf("error al decodificar moneda: %w", err)
		}
		currencies = append(currencies, c)
	}

	if currencies == nil {
		currencies = []domain.Currency{}
	}

	return currencies, nil
}

func (r *catalogRepository) GetBanks(ctx context.Context) ([]domain.Bank, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Recuperando bancos de la base de datos")

	query := `
		SELECT id, code, name, show, created_at
		FROM catalogs.banks
		WHERE show = TRUE
		ORDER BY code ASC;
	`
	rows, err := r.db.Query(dbCtx, query)
	if err != nil {
		slog.Error("Fallo al listar bancos en catalogs", "error", err)
		return nil, fmt.Errorf("error al listar bancos: %w", err)
	}
	defer rows.Close()

	var banks []domain.Bank
	for rows.Next() {
		var b domain.Bank
		err := rows.Scan(&b.ID, &b.Code, &b.Name, &b.Show, &b.CreatedAt)
		if err != nil {
			slog.Error("Error al escanear fila de banco", "error", err)
			return nil, fmt.Errorf("error al decodificar banco: %w", err)
		}
		banks = append(banks, b)
	}

	if banks == nil {
		banks = []domain.Bank{}
	}

	return banks, nil
}
