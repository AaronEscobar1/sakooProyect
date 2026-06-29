-- Migración: Agregar ventana de visibilidad y duración de visualización a los banners
-- visible_from / visible_until: rango de fechas en que el banner debe mostrarse (NULL = sin límite).
-- duration_ms: cuánto se muestra cada banner en pantalla, en milisegundos.
ALTER TABLE market.banners ADD COLUMN IF NOT EXISTS visible_from  TIMESTAMP WITH TIME ZONE NULL;
ALTER TABLE market.banners ADD COLUMN IF NOT EXISTS visible_until TIMESTAMP WITH TIME ZONE NULL;
ALTER TABLE market.banners ADD COLUMN IF NOT EXISTS duration_ms   INT NOT NULL DEFAULT 5000;
