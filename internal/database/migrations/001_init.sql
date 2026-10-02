-- 001_init.sql
-- Esquema inicial de DevConnect: 9 tablas del dominio.
-- Claves UUID (gen_random_uuid viene incluido desde PostgreSQL 13).

CREATE TABLE usuarios (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    correo          TEXT NOT NULL UNIQUE,          -- se guarda siempre en minúsculas
    nombre          TEXT NOT NULL CHECK (char_length(nombre) BETWEEN 2 AND 100),
    hash_contrasena TEXT NOT NULL,                 -- bcrypt, nunca texto plano
    rol             TEXT NOT NULL CHECK (rol IN ('estudiante', 'empresa', 'admin')),
    creado_en       TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE perfiles_estudiante (
    usuario_id     UUID PRIMARY KEY REFERENCES usuarios(id) ON DELETE CASCADE,
    programa       TEXT,
    institucion    TEXT,
    ciudad         TEXT,
    stack          TEXT[] NOT NULL DEFAULT '{}',
    biografia      TEXT CHECK (char_length(biografia) <= 500),
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE empresas (
    usuario_id     UUID PRIMARY KEY REFERENCES usuarios(id) ON DELETE CASCADE,
    razon_social   TEXT NOT NULL,
    nit            TEXT,
    sector         TEXT,
    ciudad         TEXT,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE publicaciones (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    autor_id       UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    titulo         TEXT NOT NULL CHECK (char_length(titulo) BETWEEN 5 AND 150),
    descripcion    TEXT CHECK (char_length(descripcion) <= 2000),
    lenguaje       TEXT NOT NULL,
    codigo         TEXT NOT NULL CHECK (char_length(codigo) <= 20000),
    estado         TEXT NOT NULL DEFAULT 'abierta' CHECK (estado IN ('abierta', 'resuelta')),
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_publicaciones_autor    ON publicaciones (autor_id);
CREATE INDEX idx_publicaciones_lenguaje ON publicaciones (lenguaje);
CREATE INDEX idx_publicaciones_fecha    ON publicaciones (creado_en DESC);

CREATE TABLE propuestas_mejora (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publicacion_id UUID NOT NULL REFERENCES publicaciones(id) ON DELETE CASCADE,
    autor_id       UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    codigo         TEXT NOT NULL CHECK (char_length(codigo) <= 20000),
    explicacion    TEXT NOT NULL CHECK (char_length(explicacion) BETWEEN 10 AND 2000),
    estado         TEXT NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'aceptada', 'rechazada')),
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_propuestas_publicacion  ON propuestas_mejora (publicacion_id);
CREATE INDEX idx_propuestas_autor_estado ON propuestas_mejora (autor_id, estado);

CREATE TABLE comentarios (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publicacion_id UUID NOT NULL REFERENCES publicaciones(id) ON DELETE CASCADE,
    autor_id       UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    texto          TEXT NOT NULL CHECK (char_length(texto) BETWEEN 1 AND 2000),
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_comentarios_publicacion ON comentarios (publicacion_id);

CREATE TABLE sesiones_vivo (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    anfitrion_id   UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    titulo         TEXT NOT NULL CHECK (char_length(titulo) BETWEEN 5 AND 150),
    descripcion    TEXT,
    inicia_en      TIMESTAMPTZ NOT NULL,
    sala_jitsi     TEXT NOT NULL UNIQUE,
    estado         TEXT NOT NULL DEFAULT 'programada' CHECK (estado IN ('programada', 'en_vivo', 'finalizada')),
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sesiones_inicio ON sesiones_vivo (inicia_en);

CREATE TABLE suscripciones (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id     UUID NOT NULL REFERENCES empresas(usuario_id) ON DELETE CASCADE,
    periodo        TEXT NOT NULL CHECK (periodo IN ('mensual', 'anual')),
    inicia_en      TIMESTAMPTZ NOT NULL DEFAULT now(),
    termina_en     TIMESTAMPTZ NOT NULL,
    estado         TEXT NOT NULL DEFAULT 'activa' CHECK (estado IN ('activa', 'vencida', 'cancelada')),
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (termina_en > inicia_en)
);
CREATE INDEX idx_suscripciones_empresa ON suscripciones (empresa_id, estado);

CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id  UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,              -- SHA-256 del token, nunca el token
    expira_en   TIMESTAMPTZ NOT NULL,
    revocado_en TIMESTAMPTZ,
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_usuario ON refresh_tokens (usuario_id);
