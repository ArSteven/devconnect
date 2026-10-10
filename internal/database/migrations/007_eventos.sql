-- 007_eventos.sql
-- Hechos del modelo de negocio para las métricas: quién hizo qué y cuándo, sin contenido.
-- Si se borra una cuenta (Ley 1581 de 2012), sus eventos se borran con ella.

CREATE TABLE eventos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id  UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    tipo        TEXT NOT NULL CHECK (tipo IN ('inicio_sesion', 'contacto_visto', 'suscripcion')),
    objetivo_id UUID,                               -- contacto_visto: el estudiante; suscripcion: la suscripción
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_eventos_tipo_fecha ON eventos (tipo, creado_en);
