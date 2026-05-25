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

-- Registrar bancos semilla básicos por defecto para evitar llaves vacías
INSERT INTO catalogs.banks (code, name) VALUES 
('MERCANTIL', 'Banco Mercantil'),
('BANESCO', 'Banesco Banco Universal'),
('PROVINCIAL', 'BBVA Provincial'),
('BDV', 'Banco de Venezuela')
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
