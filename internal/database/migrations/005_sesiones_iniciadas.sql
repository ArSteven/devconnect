-- 005_sesiones_iniciadas.sql
-- Distingue una sesión dictada de una cancelada: las dos terminan en 'finalizada',
-- pero solo la dictada pasó por 'en_vivo'. También permite ocultar salas olvidadas.

ALTER TABLE sesiones_vivo ADD COLUMN iniciada_en TIMESTAMPTZ;

-- Las sesiones anteriores no guardaban este dato: se conserva como se contaban hasta hoy.
UPDATE sesiones_vivo SET iniciada_en = inicia_en WHERE estado IN ('en_vivo', 'finalizada');
