-- ====================================================
-- Migración: Deduplicación de notificaciones push de tasas
-- ====================================================
-- Marca de tiempo de cuándo se envió la notificación push de cambio de tasa
-- para una fila de exchange_rates. Permite garantizar una única notificación
-- por (moneda, value_date) aunque el scraper reprocese la misma fila en cada ciclo.
ALTER TABLE market.exchange_rates ADD COLUMN IF NOT EXISTS notified_at TIMESTAMP WITH TIME ZONE NULL;
