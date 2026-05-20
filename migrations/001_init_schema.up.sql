-- Crear esquemas
CREATE SCHEMA IF NOT EXISTS catalogs;

-- ============================================================================
-- ESQUEMA: catalogs
-- ============================================================================

-- Tabla de Monedas
CREATE TABLE catalogs.currency (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(3) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Tipos de Usuario
CREATE TABLE catalogs.user_type (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Tipos de Documento
CREATE TABLE catalogs.document_type (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insertar valores iniciales por defecto en catálogos (se auto-generan IDs 1, 2, 3...)
INSERT INTO catalogs.currency (code, name) VALUES
    ('USD', 'United States Dollar'),
    ('EUR', 'Euro'),
    ('CRC', 'Costa Rican Colón')
ON CONFLICT (code) DO NOTHING;

INSERT INTO catalogs.user_type (code, name) VALUES
    ('ADMIN', 'Administrator'),
    ('CUSTOMER', 'Customer')
ON CONFLICT (code) DO NOTHING;

INSERT INTO catalogs.document_type (code, name) VALUES
    ('DNI', 'National Identity Document'),
    ('PASSPORT', 'Passport')
ON CONFLICT (code) DO NOTHING;


-- ============================================================================
-- ESQUEMA: public
-- ============================================================================

-- Tabla de Usuarios
CREATE TABLE public.users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    avatar_index INT NOT NULL DEFAULT 0,
    user_type_id BIGINT NOT NULL REFERENCES catalogs.user_type(id) ON DELETE RESTRICT,
    document_type_id BIGINT REFERENCES catalogs.document_type(id) ON DELETE SET NULL,
    document_number VARCHAR(50),
    password_hash VARCHAR(255) NOT NULL, -- Almacenar hash bcrypt
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Historial de Contraseñas
CREATE TABLE public.user_passwords_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Sesiones de Usuario
CREATE TABLE public.user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Tasas de Cambio (Exchange Rates)
CREATE TABLE public.exchange_rates (
    id BIGSERIAL PRIMARY KEY,
    currency_id BIGINT NOT NULL REFERENCES catalogs.currency(id) ON DELETE CASCADE,
    rate_from NUMERIC(18,10) NOT NULL,
    rate_to NUMERIC(18,10) NOT NULL,
    rate_average NUMERIC(18,10) NOT NULL,
    value_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_exchange_rates_currency_date UNIQUE (currency_id, value_date)
);

-- Tabla de Comentarios (Comments)
CREATE TABLE public.comments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES public.users(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Mensajes (Messages)
CREATE TABLE public.messages (
    id BIGSERIAL PRIMARY KEY,
    sender_id BIGINT REFERENCES public.users(id) ON DELETE SET NULL,
    receiver_id BIGINT REFERENCES public.users(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Compromisos de Pago (Payment Commitments)
CREATE TABLE public.payment_commitments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES public.users(id) ON DELETE SET NULL,
    amount NUMERIC(18,2) NOT NULL,
    currency_id BIGINT REFERENCES catalogs.currency(id) ON DELETE RESTRICT,
    due_date DATE NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Notificaciones de Pago (Payment Notifications)
CREATE TABLE public.payment_notifications (
    id BIGSERIAL PRIMARY KEY,
    payment_commitment_id BIGINT REFERENCES public.payment_commitments(id) ON DELETE SET NULL,
    amount_paid NUMERIC(18,2) NOT NULL,
    transaction_reference VARCHAR(100) NOT NULL,
    notification_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Configuraciones (Configurations)
CREATE TABLE public.configurations (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) UNIQUE NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
