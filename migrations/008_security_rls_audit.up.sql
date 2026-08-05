-- ============================================================================
-- SEGURIDAD: Auditoría y Row-Level Security (RLS)
-- ============================================================================

-- 1. Crear esquema de auditoría
CREATE SCHEMA IF NOT EXISTS audit;

-- 2. Tabla de historial de tasas de cambio
CREATE TABLE IF NOT EXISTS audit.exchange_rates_history (
    audit_id BIGSERIAL PRIMARY KEY,
    rate_id BIGINT NOT NULL,
    old_rate_from NUMERIC(18,10),
    new_rate_from NUMERIC(18,10),
    old_rate_to NUMERIC(18,10),
    new_rate_to NUMERIC(18,10),
    old_rate_average NUMERIC(18,10),
    new_rate_average NUMERIC(18,10),
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    action VARCHAR(10) NOT NULL -- 'UPDATE' o 'DELETE'
);

-- Función de trigger para tasas de cambio
CREATE OR REPLACE FUNCTION audit.audit_exchange_rates()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'UPDATE') THEN
        INSERT INTO audit.exchange_rates_history (rate_id, old_rate_from, new_rate_from, old_rate_to, new_rate_to, old_rate_average, new_rate_average, action)
        VALUES (OLD.id, OLD.rate_from, NEW.rate_from, OLD.rate_to, NEW.rate_to, OLD.rate_average, NEW.rate_average, 'UPDATE');
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO audit.exchange_rates_history (rate_id, old_rate_from, old_rate_to, old_rate_average, action)
        VALUES (OLD.id, OLD.rate_from, OLD.rate_to, OLD.rate_average, 'DELETE');
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Asignar el trigger
DROP TRIGGER IF EXISTS trg_audit_exchange_rates ON market.exchange_rates;
CREATE TRIGGER trg_audit_exchange_rates
AFTER UPDATE OR DELETE ON market.exchange_rates
FOR EACH ROW EXECUTE FUNCTION audit.audit_exchange_rates();


-- 3. Row-Level Security (RLS) para Cuentas Bancarias de Terceros
ALTER TABLE finance.third_party_accounts ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS user_isolation_policy ON finance.third_party_accounts;
CREATE POLICY user_isolation_policy ON finance.third_party_accounts
    FOR ALL
    USING (user_id = current_setting('sakoo.current_user_id', true)::BIGINT);

-- Nota: Para que el RLS funcione en el backend con pgx/ent, se debe ejecutar 
-- SET LOCAL sakoo.current_user_id = 'X' al inicio de cada transacción.
