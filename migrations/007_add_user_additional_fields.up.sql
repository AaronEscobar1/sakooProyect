-- Agregar columnas adicionales requeridas por el MVP a la tabla public.users
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS username VARCHAR(100) UNIQUE;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS second_name VARCHAR(100);
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS second_last_name VARCHAR(100);
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS registration_ip VARCHAR(50);
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS country VARCHAR(100);
