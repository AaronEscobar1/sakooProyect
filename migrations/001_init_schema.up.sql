-- 1. Crear esquemas si no existen
CREATE SCHEMA IF NOT EXISTS catalogs;

-- ============================================================================
-- ESQUEMA: catalogs (Tablas y Catálogos)
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

-- Tabla de Tipos de Documento de Identidad
CREATE TABLE catalogs.document_type (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Códigos de Respuesta Estandarizados (para API)
CREATE TABLE catalogs.response_codes (
    code VARCHAR(50) PRIMARY KEY,
    http_status INT NOT NULL,
    default_message TEXT NOT NULL,
    description TEXT
);

-- ============================================================================
-- ESQUEMA: public (Tablas del Negocio y Core)
-- ============================================================================

-- Tabla de Usuarios
CREATE TABLE public.users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100),
    last_name VARCHAR(100) NOT NULL,
    second_last_name VARCHAR(100),
    avatar_index INT NOT NULL DEFAULT 0,
    user_type_id BIGINT NOT NULL REFERENCES catalogs.user_type(id) ON DELETE RESTRICT,
    document_type_id BIGINT REFERENCES catalogs.document_type(id) ON DELETE SET NULL,
    document_number VARCHAR(50),
    password_hash VARCHAR(255) NOT NULL,
    registration_ip VARCHAR(50),
    country VARCHAR(100),
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
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
    rate_id BIGINT NOT NULL REFERENCES public.exchange_rates(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Mensajes (Messages)
CREATE TABLE public.messages (
    id BIGSERIAL PRIMARY KEY,
    sender_id BIGINT REFERENCES public.users(id) ON DELETE SET NULL,
    receiver_id BIGINT REFERENCES public.users(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    read_at TIMESTAMP WITH TIME ZONE NULL,
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

-- Tabla de Cuentas Bancarias Propias
CREATE TABLE public.bank_accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    bank_name VARCHAR(100) NOT NULL,
    account_number VARCHAR(100) NOT NULL,
    account_type VARCHAR(50) NOT NULL,
    holder_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Cuentas Bancarias de Terceros
CREATE TABLE public.third_party_accounts (
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

-- Tabla de Banners Publicitarios e Informativos
CREATE TABLE public.banners (
    id BIGSERIAL PRIMARY KEY,
    image_url VARCHAR(255) NOT NULL,
    link VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Logs de Auditoría de API (Trazabilidad)
CREATE TABLE public.api_logs (
    id BIGSERIAL PRIMARY KEY,
    track_code VARCHAR(50) NOT NULL,
    user_id BIGINT REFERENCES public.users(id) ON DELETE SET NULL,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(255) NOT NULL,
    http_status INT NOT NULL,
    response_code VARCHAR(50),
    latency_ms BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Gestión de OTPs (One-Time Passwords)
CREATE TABLE public.user_otps (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    otp_code VARCHAR(10) NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'REGISTER', 'RECOVER', 'DELETE'
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Tabla de Configuraciones (Configurations)
CREATE TABLE public.configurations (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) UNIQUE NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Tokens de Dispositivos (FCM Tokens)
CREATE TABLE public.user_device_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    platform VARCHAR(50) NOT NULL, -- 'android', 'ios', 'web'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Historial / Inbox de Notificaciones
CREATE TABLE public.notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    data JSONB, -- Datos adicionales adjuntos en formato clave-valor
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- ÍNDICES OPTIMIZADOS
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_api_logs_track_code ON public.api_logs(track_code);
CREATE INDEX IF NOT EXISTS idx_user_otps_email_code_action ON public.user_otps(email, otp_code, action);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON public.notifications(user_id);

-- ============================================================================
-- SEMBRADO DE DATOS (SEEDS)
-- ============================================================================

-- Tipos de Usuario
INSERT INTO catalogs.user_type (code, name) VALUES
    ('ADMIN', 'Administrator'),
    ('CUSTOMER', 'Customer')
ON CONFLICT (code) DO NOTHING;

-- Tipos de Documento de Identidad (IDs Fijos del 1 al 6)
INSERT INTO catalogs.document_type (id, code, name) VALUES
    (1, 'V', 'Venezolano'),
    (2, 'P', 'Pasaporte'),
    (3, 'E', 'Extranjero'),
    (4, 'C', 'Comuna'),
    (5, 'J', 'Jurídico'),
    (6, 'G', 'Gubernamental')
ON CONFLICT (id) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name;

SELECT setval('catalogs.document_type_id_seq', COALESCE((SELECT MAX(id) FROM catalogs.document_type), 1));

-- Monedas Core y Web Scraper (BCV + Mercantil)
INSERT INTO catalogs.currency (code, name) VALUES
    ('USD', 'Dólar Estadounidense'),
    ('EUR', 'Euro'),
    ('CRC', 'Colón Costarricense'),
    ('CNY', 'Yuan Chino'),
    ('TRY', 'Lira Turca'),
    ('RUB', 'Rublo Ruso'),
    ('UDI', 'Dólar Intervención')
ON CONFLICT (code) DO NOTHING;

-- Banners de Prueba Iniciales
INSERT INTO public.banners (image_url, link, is_active) VALUES
    ('https://images.unsplash.com/photo-1611974789855-9c2a0a7236a3?auto=format&fit=crop&w=800&q=80', 'https://sakoo.com/promo/tasa-preferencial', TRUE),
    ('https://images.unsplash.com/photo-1559526324-4b87b5e36e44?auto=format&fit=crop&w=800&q=80', 'https://sakoo.com/promo/nueva-calculadora', TRUE)
ON CONFLICT DO NOTHING;

-- Códigos de Respuesta del Servidor
INSERT INTO catalogs.response_codes (code, http_status, default_message, description) VALUES
    ('SUCCESS', 200, 'Operación completada con éxito', 'Respuesta estándar para operaciones exitosas'),
    ('CREATED', 201, 'Recurso creado exitosamente', 'Respuesta estándar para inserciones exitosas'),
    ('INVALID_JSON', 400, 'El formato de los datos de entrada es incorrecto', 'Error al decodificar payload JSON'),
    ('BAD_REQUEST', 400, 'La solicitud contiene parámetros incorrectos', 'Error de validación o parámetros inválidos'),
    ('UNAUTHORIZED', 401, 'Credenciales de acceso incorrectas o token ausente', 'Fallo de autenticación o JWT inválido'),
    ('USER_ALREADY_EXISTS', 409, 'El correo electrónico ingresado ya se encuentra registrado', 'Intento de registro con email duplicado'),
    ('INTERNAL_ERROR', 500, 'Ha ocurrido un error interno en el servidor. Por favor, intente más tarde.', 'Error genérico del sistema no controlado')
ON CONFLICT (code) DO NOTHING;
