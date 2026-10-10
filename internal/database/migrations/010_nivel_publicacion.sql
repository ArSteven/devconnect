-- 010_nivel_publicacion.sql
-- Nivel opcional de un código publicado, para que quien empieza encuentre qué practicar.
-- Los retos no lo usan.

ALTER TABLE publicaciones
    ADD COLUMN nivel TEXT NULL CHECK (nivel IN ('principiante', 'intermedio'));
