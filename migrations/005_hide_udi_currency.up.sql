-- ====================================================
-- Migración: Retiro de la tasa de intervención (UDI / Mercantil)
-- ====================================================
-- La tasa "Dólar Intervención" (UDI, raspada del Banco Mercantil) deja de utilizarse.
-- Se oculta la moneda sin borrar el histórico ya registrado (datos preservados).
UPDATE catalogs.currency SET "show" = FALSE, display_order = 0 WHERE code = 'UDI';
