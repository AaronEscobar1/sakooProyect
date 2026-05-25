package domain

import (
	"context"
	"time"
)

// Currency representa la moneda disponible en el catálogo (ej: USD, EUR, CRC).
type Currency struct {
	ID           int64     `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

// Bank representa un banco en el catálogo.
type Bank struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// CatalogRepository define la interfaz de persistencia para consultar catálogos.
type CatalogRepository interface {
	GetDocumentTypes(ctx context.Context) ([]DocumentType, error)
	GetCurrencies(ctx context.Context) ([]Currency, error)
}

// CatalogUseCase define la lógica de negocio para consultar catálogos.
type CatalogUseCase interface {
	GetDocumentTypes(ctx context.Context) ([]DocumentType, error)
	GetCurrencies(ctx context.Context) ([]Currency, error)
}
