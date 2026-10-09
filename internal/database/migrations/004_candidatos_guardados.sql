-- 004_candidatos_guardados.sql
-- Lista de candidatos de cada empresa: el puente con su proceso de selección.

CREATE TABLE candidatos_guardados (
    empresa_id    UUID NOT NULL REFERENCES empresas(usuario_id) ON DELETE CASCADE,
    estudiante_id UUID NOT NULL REFERENCES perfiles_estudiante(usuario_id) ON DELETE CASCADE,
    creado_en     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (empresa_id, estudiante_id)
);
CREATE INDEX idx_candidatos_estudiante ON candidatos_guardados (estudiante_id);
