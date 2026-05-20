-- ============================================================================
-- MIGRACIÓN: 003_add_scraper_currencies.up.sql
-- ============================================================================

-- Insertar nuevas monedas necesarias para el Web Scraper del BCV
INSERT INTO catalogs.currency (code, name) VALUES
    ('CNY', 'Yuan Chino'),
    ('TRY', 'Lira Turca'),
    ('RUB', 'Rublo Ruso')
ON CONFLICT (code) DO NOTHING;
