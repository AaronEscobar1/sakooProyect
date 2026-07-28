package usecase

import (
	"context"
	"log/slog"

	"github.com/aaron/sakoo-backend/internal/domain"
)

type catalogUseCase struct {
	repo domain.CatalogRepository
}

// NewCatalogUseCase crea un caso de uso para catálogos del sistema.
func NewCatalogUseCase(repo domain.CatalogRepository) domain.CatalogUseCase {
	return &catalogUseCase{
		repo: repo,
	}
}

func (uc *catalogUseCase) GetDocumentTypes(ctx context.Context) ([]domain.DocumentType, error) {
	slog.Info("Procesando listado de tipos de documento")
	return uc.repo.GetDocumentTypes(ctx)
}

func (uc *catalogUseCase) GetCurrencies(ctx context.Context) ([]domain.Currency, error) {
	slog.Info("Procesando listado de monedas")
	return uc.repo.GetCurrencies(ctx)
}

func (uc *catalogUseCase) GetBanks(ctx context.Context) ([]domain.Bank, error) {
	slog.Info("Procesando listado de bancos")
	return uc.repo.GetBanks(ctx)
}
