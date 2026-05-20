-- Crear la tabla de banners publicitarios activos
CREATE TABLE public.banners (
    id BIGSERIAL PRIMARY KEY,
    image_url VARCHAR(255) NOT NULL,
    link VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insertar algunos banners iniciales para demostración
INSERT INTO public.banners (image_url, link, is_active) VALUES
    ('https://images.unsplash.com/photo-1611974789855-9c2a0a7236a3?auto=format&fit=crop&w=800&q=80', 'https://sakoo.com/promo/tasa-preferencial', TRUE),
    ('https://images.unsplash.com/photo-1559526324-4b87b5e36e44?auto=format&fit=crop&w=800&q=80', 'https://sakoo.com/promo/nueva-calculadora', TRUE)
ON CONFLICT DO NOTHING;
