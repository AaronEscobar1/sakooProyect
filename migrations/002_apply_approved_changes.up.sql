-- ====================================================
-- Migración: Cambios Aprobados y Catálogo de Bancos
-- Generado el: 2026-05-25 12:31:00
-- ====================================================

-- 1. Catálogo Centralizado de Bancos
CREATE TABLE IF NOT EXISTS catalogs.banks (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Limpiar la tabla antes de re-sembrar para evitar duplicados locales y limpiar códigos de letras legados
TRUNCATE TABLE catalogs.banks RESTART IDENTITY CASCADE;

-- Registrar la lista completa y oficial de bancos activos en Venezuela con sus códigos oficiales de Sudeban
INSERT INTO catalogs.banks (code, name) VALUES 
('0102', 'Banco de Venezuela, S.A. Banco Universal'),
('0104', 'Venezolano de Crédito, S.A. Banco Universal'),
('0105', 'Banco Mercantil, C.A. Banco Universal'),
('0108', 'Provincial, S.A. Banco Universal (BBVA Provincial)'),
('0114', 'Bancaribe, C.A. Banco Universal'),
('0115', 'Banco Exterior, C.A. Banco Universal'),
('0128', 'Banco Caroní, C.A. Banco Universal'),
('0134', 'Banesco Banco Universal, C.A.'),
('0138', 'Banco Plaza, C.A. Banco Universal'),
('0151', 'BFC Banco Fondo Común, C.A. Banco Universal'),
('0156', '100% Banco, Banco Universal, C.A.'),
('0157', 'DelSur, Banco Universal, C.A.'),
('0163', 'Banco del Tesoro, C.A. Banco Universal'),
('0166', 'Banco Agrícola de Venezuela, C.A. Banco Universal'),
('0168', 'Bancrecer, S.A. Banco Microfinanciero'),
('0169', 'Mi Banco, Banco Microfinanciero, C.A.'),
('0171', 'Banco Activo, C.A. Banco Universal'),
('0172', 'Bancamiga Banco Universal, C.A.'),
('0174', 'Banplus Banco Universal, C.A.'),
('0175', 'Banco Bicentenario del Pueblo, Banco Universal C.A.'),
('0177', 'BANFANB (Banco de la Fuerza Armada Nacional Bolivariana)'),
('0191', 'BNC Banco Nacional de Crédito, C.A. Banco Universal')
ON CONFLICT (code) DO NOTHING;

-- En finance.bank_accounts: quitar bank_name y agregar bank_id FK
ALTER TABLE finance.bank_accounts DROP COLUMN IF EXISTS bank_name;
ALTER TABLE finance.bank_accounts ADD COLUMN IF NOT EXISTS bank_id BIGINT REFERENCES catalogs.banks(id) ON DELETE RESTRICT;

-- En finance.third_party_accounts: quitar bank_name y agregar bank_id FK
ALTER TABLE finance.third_party_accounts DROP COLUMN IF EXISTS bank_name;
ALTER TABLE finance.third_party_accounts ADD COLUMN IF NOT EXISTS bank_id BIGINT REFERENCES catalogs.banks(id) ON DELETE RESTRICT;

-- 2. Teléfono en Cuentas
ALTER TABLE finance.third_party_accounts ADD COLUMN IF NOT EXISTS phone_number VARCHAR(50) NULL;

-- 3. Estados y Origen en Tasas de Cambio
ALTER TABLE market.exchange_rates ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'REGISTERED';
ALTER TABLE market.exchange_rates ADD COLUMN IF NOT EXISTS source VARCHAR(50) DEFAULT 'SCRAPING';

-- 4. Concepto en Compromisos de Pago
ALTER TABLE finance.payment_commitments ADD COLUMN IF NOT EXISTS concept VARCHAR(255) NULL;

-- 5. Ordenamiento en Catálogos
ALTER TABLE catalogs.currency ADD COLUMN IF NOT EXISTS display_order INT DEFAULT 0;
ALTER TABLE catalogs.document_type ADD COLUMN IF NOT EXISTS display_order INT DEFAULT 0;
