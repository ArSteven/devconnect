-- 006_terminos_y_datos.sql
-- Ley 1581 de 2012: prueba de la autorización previa, expresa e informada (cuándo y
-- qué versión aceptó) y la opción del estudiante de no mostrar su correo a las empresas.
-- Las cuentas anteriores son de prueba y se borran antes del piloto: no se rellenan.

ALTER TABLE usuarios
    ADD COLUMN terminos_aceptados_en TIMESTAMPTZ,
    ADD COLUMN version_terminos      TEXT;

ALTER TABLE perfiles_estudiante
    ADD COLUMN contacto_visible BOOLEAN NOT NULL DEFAULT TRUE;
