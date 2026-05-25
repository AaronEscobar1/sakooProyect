-- ====================================================
-- Migración: Cambios Aprobados y Catálogo de Bancos
-- Generado el: 2026-05-25 12:31:00
-- ====================================================

-- 1. Catálogo Centralizado de Bancos
CREATE TABLE IF NOT EXISTS catalogs.banks (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    show BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Limpiar la tabla antes de re-sembrar para evitar duplicados locales y limpiar códigos de letras legados
TRUNCATE TABLE catalogs.banks RESTART IDENTITY CASCADE;

-- Registrar la lista oficial completa de 33 bancos suministrada
INSERT INTO catalogs.banks (code, name) VALUES 
('0001', 'Banco Central de Venezuela'),
('0102', 'Banco de Venezuela'),
('0104', 'Venezolano de Crédito'),
('0105', 'Mercantil'),
('0108', 'Provincial'),
('0114', 'Bancaribe'),
('0115', 'Banco Exterior'),
('0128', 'Banco Caroní'),
('0134', 'Banesco'),
('0137', 'Sofitasa'),
('0138', 'Banco Plaza'),
('0145', 'BANCO DE COMERCIO EXTERIOR'),
('0146', 'Bangente'),
('0151', 'BFC Banco Fondo Común'),
('0152', 'BANDES'),
('0156', '100% Banco'),
('0157', 'Delsur'),
('0163', 'Banco del Tesoro'),
('0166', 'Banco Agrícola de Venezuela'),
('0168', 'Bancrecer'),
('0169', 'R4'),
('0171', 'Activo'),
('0172', 'Bancamiga'),
('0173', 'BANCO INTERNACIONAL DE DESARROLLO,'),
('0174', 'Banplus'),
('0175', 'Banco Digital de los Trabajadores'),
('0177', 'BANFANB'),
('0178', 'N58 Banco Digital'),
('0191', 'BNC Banco Nacional de Crédito'),
('0601', 'I.M.C.P'),
('0732', 'FONDEN'),
('2017', 'ONT'),
('6000', 'BANAVIH')
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
