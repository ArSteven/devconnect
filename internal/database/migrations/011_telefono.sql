-- 011_telefono.sql
-- Teléfono del estudiante: opcional y lo escribe él mismo en su perfil. Sigue la regla del correo:
-- solo lo ven él y las empresas con suscripción vigente, y a ellas solo si tiene el contacto visible.
-- Se guarda sin espacios ni guiones: dígitos, con un + inicial si trae indicativo.

ALTER TABLE perfiles_estudiante
    ADD COLUMN telefono TEXT CHECK (telefono ~ '^\+?[0-9]{7,15}$');
