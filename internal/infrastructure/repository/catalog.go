package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/documenttype"
	"github.com/aaron/sakoo-backend/ent/currency"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type catalogRepository struct {
	client *ent.Client
}

// NewCatalogRepository crea un repositorio para catálogos del sistema.
func NewCatalogRepository(client *ent.Client) domain.CatalogRepository {
	return &catalogRepository{
		client: client,
	}
}

func (r *catalogRepository) GetDocumentTypes(ctx context.Context) ([]domain.DocumentType, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Recuperando tipos de documento de la base de datos usando Ent")

	// Usamos Ent builder
	docTypes, err := r.client.DocumentType.Query().
		Order(ent.Asc(documenttype.FieldName)).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al listar tipos de documento en catalogs", "error", err)
		return nil, fmt.Errorf("error al listar tipos de documento: %w", err)
	}

	var result []domain.DocumentType
	for _, dt := range docTypes {
		result = append(result, domain.DocumentType{
			ID:           int64(dt.ID),
			Code:         dt.Code,
			Name:         dt.Name,
			DisplayOrder: int(dt.ID), // placeholder
			CreatedAt:    dt.CreatedAt,
		})
	}

	if result == nil {
		result = []domain.DocumentType{}
	}

	return result, nil
}

func (r *catalogRepository) GetCurrencies(ctx context.Context) ([]domain.Currency, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Recuperando monedas de la base de datos usando Ent")

	currencies, err := r.client.Currency.Query().
		Order(ent.Asc(currency.FieldName)).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al listar monedas en catalogs", "error", err)
		return nil, fmt.Errorf("error al listar monedas: %w", err)
	}

	var result []domain.Currency
	for _, c := range currencies {
		result = append(result, domain.Currency{
			ID:           int64(c.ID),
			Code:         c.Code,
			Name:         c.Name,
			DisplayOrder: int(c.ID),
			CreatedAt:    c.CreatedAt,
			UpdatedAt:    c.UpdatedAt,
		})
	}

	if result == nil {
		result = []domain.Currency{}
	}

	return result, nil
}

func (r *catalogRepository) GetBanks(ctx context.Context) ([]domain.Bank, error) {
	// Dummy para mantener compatibilidad en esta prueba de concepto.
	return []domain.Bank{}, nil
}
