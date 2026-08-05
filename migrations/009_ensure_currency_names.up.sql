-- Migración: Garantizar presencia, descripciones completas y visibilidad para todas las monedas en catalogs.currency
INSERT INTO catalogs.currency (code, name, "show", display_order) VALUES
    ('USD', 'Dólar Estadounidense', TRUE, 1),
    ('EUR', 'Euro', TRUE, 2),
    ('USDT', 'Tether USDT', TRUE, 3),
    ('USDC', 'USD Coin', TRUE, 4),
    ('COP', 'Peso Colombiano', TRUE, 5),
    ('VES', 'Bolívar Venezolano', TRUE, 6),
    ('BRL', 'Real Brasileño', TRUE, 7),
    ('ARS', 'Peso Argentino', TRUE, 8),
    ('CLP', 'Peso Chileno', TRUE, 9),
    ('PEN', 'Sol Peruano', TRUE, 10),
    ('CRC', 'Colón Costarricense', TRUE, 11),
    ('CNY', 'Yuan Chino', TRUE, 12),
    ('TRY', 'Lira Turca', TRUE, 13),
    ('RUB', 'Rublo Ruso', TRUE, 14)
ON CONFLICT (code) DO UPDATE SET 
    name = EXCLUDED.name,
    "show" = TRUE,
    display_order = EXCLUDED.display_order;

UPDATE catalogs.currency SET "show" = TRUE WHERE code IN ('USD', 'EUR', 'USDT', 'USDC', 'COP', 'VES', 'BRL', 'ARS', 'CLP', 'PEN', 'CRC', 'CNY', 'TRY', 'RUB');
