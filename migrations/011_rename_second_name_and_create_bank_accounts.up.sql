-- 1. Renombrar la columna second_name a middle_name en la tabla public.users si existe
ALTER TABLE public.users RENAME COLUMN second_name TO middle_name;

-- 2. Crear tabla de cuentas bancarias propias
CREATE TABLE IF NOT EXISTS public.bank_accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    bank_name VARCHAR(100) NOT NULL,
    account_number VARCHAR(100) NOT NULL,
    account_type VARCHAR(50) NOT NULL, -- Ej: Corriente, Ahorros
    holder_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Crear tabla de cuentas de terceros
CREATE TABLE IF NOT EXISTS public.third_party_accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    bank_name VARCHAR(100) NOT NULL,
    account_number VARCHAR(100) NOT NULL,
    account_type VARCHAR(50) NOT NULL,
    holder_name VARCHAR(100) NOT NULL,
    alias VARCHAR(100) NOT NULL,
    document_number VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
