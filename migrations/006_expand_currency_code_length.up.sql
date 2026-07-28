-- Ampliar la longitud de la columna code en la tabla catalogs.currency para permitir nombres o códigos concatenados largos
ALTER TABLE catalogs.currency ALTER COLUMN code TYPE VARCHAR(100);
