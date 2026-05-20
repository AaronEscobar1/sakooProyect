INSERT INTO catalogs.currency (code, name) VALUES
    ('UDI', 'Dólar Intervención')
ON CONFLICT (code) DO NOTHING;
