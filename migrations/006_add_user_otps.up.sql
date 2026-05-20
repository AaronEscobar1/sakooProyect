-- Migración para crear la tabla de gestión de OTPs (One-Time Passwords)
CREATE TABLE IF NOT EXISTS public.user_otps (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    otp_code VARCHAR(10) NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'REGISTER', 'RECOVER', 'DELETE'
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Índice compuesto para búsquedas ultra rápidas de OTPs no usados y vigentes
CREATE INDEX IF NOT EXISTS idx_user_otps_email_code_action ON public.user_otps (email, otp_code, action);
