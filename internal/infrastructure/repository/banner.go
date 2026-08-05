package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/banner"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type bannerRepository struct {
	client *ent.Client
}

// NewBannerRepository crea un repositorio para banners publicitarios usando Ent.
func NewBannerRepository(client *ent.Client) domain.BannerRepository {
	return &bannerRepository{
		client: client,
	}
}

func (r *bannerRepository) ListActive(ctx context.Context) ([]domain.Banner, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Recuperando banners activos de la base de datos con Ent")

	banners, err := r.client.Banner.Query().
		Where(banner.IsActiveEQ(true)).
		Order(ent.Asc(banner.FieldDisplayOrder), ent.Asc(banner.FieldID)).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al listar banners activos en Ent", "error", err)
		return nil, fmt.Errorf("error al listar banners activos: %w", err)
	}

	var result []domain.Banner
	for _, b := range banners {
		result = append(result, domain.Banner{
			ID:           int64(b.ID),
			ImageURL:     b.ImageURL,
			Link:         b.Link,
			IsActive:     b.IsActive,
			DisplayOrder: b.DisplayOrder,
			CreatedAt:    b.CreatedAt,
			UpdatedAt:    b.UpdatedAt,
		})
	}

	if result == nil {
		result = []domain.Banner{}
	}

	return result, nil
}
