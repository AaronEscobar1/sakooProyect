-- Crear esquemas si no existen
CREATE SCHEMA IF NOT EXISTS catalogs;

-- ============================================================================
-- ESQUEMA: catalogs (response_codes)
-- ============================================================================

CREATE TABLE catalogs.response_codes (
    code VARCHAR(50) PRIMARY KEY,
    http_status INT NOT NULL,
    default_message TEXT NOT NULL,
    description TEXT
);

-- Sembrado inicial de códigos de respuesta estandarizados
INSERT INTO catalogs.response_codes (code, http_status, default_message, description) VALUES
    ('SUCCESS', 200, 'Operación completada con éxito', 'Respuesta estándar para operaciones exitosas'),
    ('CREATED', 201, 'Recurso creado exitosamente', 'Respuesta estándar para inserciones exitosas'),
    ('INVALID_JSON', 400, 'El formato de los datos de entrada es incorrecto', 'Error al decodificar payload JSON'),
    ('BAD_REQUEST', 400, 'La solicitud contiene parámetros incorrectos', 'Error de validación o parámetros inválidos'),
    ('UNAUTHORIZED', 401, 'Credenciales de acceso incorrectas o token ausente', 'Fallo de autenticación o JWT inválido'),
    ('USER_ALREADY_EXISTS', 409, 'El correo electrónico ingresado ya se encuentra registrado', 'Intento de registro con email duplicado'),
    ('INTERNAL_ERROR', 500, 'Ha ocurrido un error interno en el servidor. Por favor, intente más tarde.', 'Error genérico del sistema no controlado')
ON CONFLICT (code) DO NOTHING;


-- ============================================================================
-- ESQUEMA: public (api_logs)
-- ============================================================================

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

-- Índice optimizado para búsquedas por código de rastreo (trazabilidad rápida)
CREATE INDEX idx_api_logs_track_code ON public.api_logs(track_code);
