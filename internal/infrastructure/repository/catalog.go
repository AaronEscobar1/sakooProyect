package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/bank"
	"github.com/aaron/sakoo-backend/ent/currency"
	"github.com/aaron/sakoo-backend/ent/documenttype"
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

	docTypes, err := r.client.DocumentType.Query().
		Order(ent.Asc(documenttype.FieldDisplayOrder), ent.Asc(documenttype.FieldName)).
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
			DisplayOrder: dt.DisplayOrder,
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

	slog.Debug("Recuperando monedas activas de la base de datos usando Ent")

	currencies, err := r.client.Currency.Query().
		Where(currency.ShowEQ(true)).
		Order(ent.Asc(currency.FieldDisplayOrder), ent.Asc(currency.FieldName)).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al listar monedas en catalogs", "error", err)
		return nil, fmt.Errorf("error al listar monedas: %w", err)
	}

	var result []domain.Currency
	for _, c := range currencies {
		name := c.Name
		if name == "" || strings.EqualFold(name, c.Code) {
			switch c.Code {
			case "COP":
				name = "PESO COLOMBIANO"
			case "USDT":
				name = "TETHER"
			case "USDC":
				name = "USD COIN"
			case "USD":
				name = "DÓLAR ESTADOUNIDENSE"
			case "EUR":
				name = "EURO"
			case "VES":
				name = "BOLÍVAR VENEZOLANO"
			case "BRL":
				name = "REAL BRASILEÑO"
			case "ARS":
				name = "PESO ARGENTINO"
			case "CLP":
				name = "PESO CHILENO"
			case "PEN":
				name = "SOL PERUANO"
			case "CRC":
				name = "COLÓN COSTARRICENSE"
			case "CNY":
				name = "YUAN CHINO"
			case "TRY":
				name = "LIRA TURCA"
			case "RUB":
				name = "RUBLO RUSO"
			case "UDI":
				name = "DÓLAR INTERVENCIÓN"
			default:
				name = c.Code
			}
		}

		result = append(result, domain.Currency{
			ID:           int64(c.ID),
			Code:         c.Code,
			Name:         name,
			Description:  name,
			DisplayOrder: c.DisplayOrder,
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
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Recuperando bancos de la base de datos usando Ent")

	banks, err := r.client.Bank.Query().
		Where(bank.ShowEQ(true)).
		Order(ent.Asc(bank.FieldCode)).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al listar bancos en catalogs", "error", err)
		return nil, fmt.Errorf("error al listar bancos: %w", err)
	}

	var result []domain.Bank
	for _, b := range banks {
		result = append(result, domain.Bank{
			ID:        int64(b.ID),
			Code:      b.Code,
			Name:      b.Name,
			Show:      b.Show,
			CreatedAt: b.CreatedAt,
		})
	}

	if result == nil {
		result = []domain.Bank{}
	}

	return result, nil
}
