-- 003_retos.sql
-- Retos técnicos de empresas. Un reto es una publicación más: reutiliza las
-- propuestas de mejora y la aceptación, así que no hay lógica duplicada.

ALTER TABLE publicaciones
    ADD COLUMN tipo         TEXT NOT NULL DEFAULT 'pregunta' CHECK (tipo IN ('pregunta', 'reto')),
    ADD COLUMN fecha_limite TIMESTAMPTZ,
    ADD CONSTRAINT publicaciones_reto_con_fecha CHECK (tipo = 'pregunta' OR fecha_limite IS NOT NULL);

CREATE INDEX idx_publicaciones_tipo_fecha ON publicaciones (tipo, creado_en DESC);
CREATE INDEX idx_publicaciones_estado     ON publicaciones (estado);
