package domain

import (
	"context"
	"time"
)

// Banner representa un banner publicitario o informativo para la aplicación.
type Banner struct {
	ID        int64     `json:"id"`
	ImageURL  string    `json:"image_url"`
	Link      string    `json:"link"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BannerRepository define las operaciones de persistencia para banners.
type BannerRepository interface {
	ListActive(ctx context.Context) ([]Banner, error)
}

// BannerUseCase define la lógica de negocio para banners.
type BannerUseCase interface {
	GetActiveBanners(ctx context.Context) ([]Banner, error)
}
