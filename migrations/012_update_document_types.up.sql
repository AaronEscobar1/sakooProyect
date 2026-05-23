-- 1. Actualizar los registros existentes para reutilizar los IDs 1 y 2, evitando romper llaves foráneas de usuarios existentes
UPDATE catalogs.document_type 
SET code = 'V', name = 'Venezolano' 
WHERE id = 1 OR code = 'DNI';

UPDATE catalogs.document_type 
SET code = 'P', name = 'Pasaporte' 
WHERE id = 2 OR code = 'PASSPORT';

-- 2. Eliminar cualquier residuo con códigos antiguos si quedaron duplicados
DELETE FROM catalogs.document_type WHERE code IN ('DNI', 'PASSPORT');

-- 3. Insertar/Actualizar todos los tipos de documento requeridos con IDs fijos del 1 al 6 de forma segura
INSERT INTO catalogs.document_type (id, code, name) VALUES
    (1, 'V', 'Venezolano'),
    (2, 'P', 'Pasaporte'),
    (3, 'E', 'Extranjero'),
    (4, 'C', 'Comuna'),
    (5, 'J', 'Jurídico'),
    (6, 'G', 'Gubernamental')
ON CONFLICT (id) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name;

-- 4. Sincronizar el secuenciador de PostgreSQL para evitar colisiones en futuras inserciones automáticas
SELECT setval('catalogs.document_type_id_seq', COALESCE((SELECT MAX(id) FROM catalogs.document_type), 1));
