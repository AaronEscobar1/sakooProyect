-- Añadir columna rate_id para asociar los comentarios con las tasas de cambio específicas
ALTER TABLE public.comments ADD COLUMN rate_id BIGINT NOT NULL REFERENCES public.exchange_rates(id) ON DELETE CASCADE;
