-- 002_perfil_profesional.sql
-- Datos de hoja de vida para que el portafolio sirva a un reclutador.

ALTER TABLE perfiles_estudiante
    ADD COLUMN titular          TEXT CHECK (char_length(titular) <= 120),
    ADD COLUMN fecha_nacimiento DATE,
    ADD COLUMN semestre         SMALLINT CHECK (semestre BETWEEN 1 AND 12),
    ADD COLUMN estado_academico TEXT CHECK (estado_academico IN ('cursando', 'egresado')),
    ADD COLUMN anio_inicio      SMALLINT CHECK (anio_inicio BETWEEN 1990 AND 2040),
    ADD COLUMN anio_fin         SMALLINT CHECK (anio_fin BETWEEN 1990 AND 2045),
    ADD COLUMN disponibilidad   TEXT CHECK (disponibilidad IN ('practicas', 'medio_tiempo', 'tiempo_completo', 'freelance', 'no_disponible')),
    ADD COLUMN modalidad        TEXT CHECK (modalidad IN ('presencial', 'remoto', 'hibrido')),
    ADD COLUMN github_url       TEXT CHECK (char_length(github_url) <= 200),
    ADD COLUMN linkedin_url     TEXT CHECK (char_length(linkedin_url) <= 200),
    ADD COLUMN sitio_url        TEXT CHECK (char_length(sitio_url) <= 200),
    ADD COLUMN idiomas          TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN experiencia      JSONB NOT NULL DEFAULT '[]';
