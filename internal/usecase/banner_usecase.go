package usecase

import (
	"context"
	"log/slog"

	"github.com/aaron/sakoo-backend/internal/domain"
)

type bannerUseCase struct {
	repo domain.BannerRepository
}

// NewBannerUseCase crea un caso de uso para banners publicitarios.
func NewBannerUseCase(repo domain.BannerRepository) domain.BannerUseCase {
	return &bannerUseCase{
		repo: repo,
	}
}

func (uc *bannerUseCase) GetActiveBanners(ctx context.Context) ([]domain.Banner, error) {
	slog.Info("Procesando listado de banners publicitarios activos")
	return uc.repo.ListActive(ctx)
}
