-- 009_retos_ia_y_defensas.sql
-- Soluciones de retos frente al uso de IA: quien resuelve declara si usó inteligencia artificial y
-- la empresa puede citarlo a defender su solución en vivo.

-- Declaración de uso de IA (solo en soluciones de retos): no, para consultar dudas o para generar código.
ALTER TABLE propuestas_mejora
    ADD COLUMN uso_ia         TEXT CHECK (uso_ia IN ('no', 'consulta', 'codigo')),
    ADD COLUMN uso_ia_detalle TEXT CHECK (char_length(uso_ia_detalle) <= 500);

-- Defensa en vivo: una sala privada de Jitsi de 15 minutos entre la empresa dueña del reto y quien
-- envió la solución. Solo ellos dos la ven. Aprobada solo puede estar una defensa realizada.
CREATE TABLE defensas (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    propuesta_id   UUID NOT NULL REFERENCES propuestas_mejora(id) ON DELETE CASCADE,
    empresa_id     UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    estudiante_id  UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    inicia_en      TIMESTAMPTZ NOT NULL,
    sala_jitsi     TEXT NOT NULL UNIQUE,
    estado         TEXT NOT NULL DEFAULT 'invitada' CHECK (estado IN ('invitada', 'realizada', 'cancelada')),
    aprobada       BOOLEAN NOT NULL DEFAULT false,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NOT aprobada OR estado = 'realizada')
);
-- Una sola defensa vigente por solución; las canceladas quedan como historia.
CREATE UNIQUE INDEX idx_defensas_vigente ON defensas (propuesta_id) WHERE estado <> 'cancelada';
CREATE INDEX idx_defensas_estudiante ON defensas (estudiante_id, inicia_en);
CREATE INDEX idx_defensas_empresa ON defensas (empresa_id, inicia_en);
