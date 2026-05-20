-- Añadir columna read_at para el control de mensajes leídos/no leídos
ALTER TABLE public.messages ADD COLUMN read_at TIMESTAMP WITH TIME ZONE NULL;
