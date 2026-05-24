package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type bannerRepository struct {
	db *pgxpool.Pool
}

// NewBannerRepository crea un repositorio para banners publicitarios.
func NewBannerRepository(db *pgxpool.Pool) domain.BannerRepository {
	return &bannerRepository{
		db: db,
	}
}

func (r *bannerRepository) ListActive(ctx context.Context) ([]domain.Banner, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Recuperando banners activos de la base de datos")

	query := `
		SELECT id, image_url, link, is_active, created_at, updated_at
		FROM banners
		WHERE is_active = TRUE
		ORDER BY created_at DESC;
	`
	rows, err := r.db.Query(dbCtx, query)
	if err != nil {
		slog.Error("Fallo al listar banners activos en PostgreSQL", "error", err)
		return nil, fmt.Errorf("error al listar banners activos: %w", err)
	}
	defer rows.Close()

	var banners []domain.Banner
	for rows.Next() {
		var b domain.Banner
		err := rows.Scan(&b.ID, &b.ImageURL, &b.Link, &b.IsActive, &b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			slog.Error("Error al escanear fila de banner", "error", err)
			return nil, fmt.Errorf("error al decodificar banner: %w", err)
		}
		banners = append(banners, b)
	}

	if banners == nil {
		banners = []domain.Banner{}
	}

	return banners, nil
}
