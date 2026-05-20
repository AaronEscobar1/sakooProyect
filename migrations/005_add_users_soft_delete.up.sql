-- Agregar columna deleted_at para borrado lógico de usuarios
ALTER TABLE public.users ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE NULL;
