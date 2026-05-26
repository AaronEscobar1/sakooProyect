-- Migración: Agregar orden de visualización a los banners
ALTER TABLE market.banners ADD COLUMN IF NOT EXISTS display_order INT DEFAULT 0;
