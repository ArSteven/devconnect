# DevConnect API

Backend de **DevConnect**, una red social donde estudiantes de programación aprenden mejorando el código de otros y construyen un portafolio que las empresas pueden verificar.

**Demo:** [devconnect-web-mauve.vercel.app](https://devconnect-web-mauve.vercel.app) · **Frontend:** [ArSteven/devconnect-web](https://github.com/ArSteven/devconnect-web)

Go · Gin · PostgreSQL (pgx) · JWT · desplegado en Railway

---

## Qué resuelve

Un estudiante sin experiencia no tiene cómo demostrar que sabe programar, y una empresa no tiene cómo saber si un portafolio lo hizo una persona o una IA. DevConnect mide lo que alguien aporta al código de otros:

1. Un estudiante publica código.
2. Otros proponen mejoras; el autor las acepta o rechaza.
3. Cualquiera puede **ejecutar las dos versiones** y la plataforma deja constancia de que la mejora corre y cambia el resultado.
4. Las empresas publican **retos**, revisan soluciones y citan a una **defensa en vivo** de 15 minutos para comprobar que quien la envió la entiende.
5. El portafolio resume todo con señales verificables: mejoras aceptadas, tasa de aceptación, mejoras verificadas por ejecución y defensas aprobadas.

## Arquitectura

```
cmd/api/main.go          arranque, inyección de dependencias y rutas
internal/
  config/                variables de entorno (.env en local, Railway en producción)
  database/              pool pgx y migraciones SQL versionadas (se aplican al arrancar)
  handler/               HTTP: validación de entrada y respuestas
  service/               reglas de negocio
  repository/            consultas SQL
  model/                 tipos compartidos
  middleware/            autenticación, roles, cabeceras de seguridad, límites
```

Capas estrictas: `handler → service → repository`. Los handlers no tocan SQL y los repositorios no conocen HTTP.

## Decisiones que vale la pena mirar

- **El servidor nunca ejecuta código de usuarios.** JavaScript, TypeScript, Python y SQL corren en el navegador dentro de un iframe aislado con CSP `sandbox`; Go se reenvía al Go Playground oficial (`internal/service/ejecucion.go`). La API solo guarda la constancia de la verificación.
- **Autenticación:** contraseñas con bcrypt, token de acceso JWT corto (se validan firma, algoritmo, emisor y vencimiento) y refresh token de un solo uso que se revoca al rotar o al cerrar sesión.
- **Login sin filtrar información:** se compara contra un hash falso cuando el correo no existe, para que el tiempo de respuesta no revele qué cuentas están registradas.
- **Límite de intentos** con token bucket (`x/time/rate`): 5 por minuto por correo en el login y 10 ejecuciones de Go por minuto por persona; cuerpo máximo de 1 MB y cabeceras de seguridad en todas las respuestas.
- **Ley 1581 de 2012:** el registro exige la autorización expresa de tratamiento de datos y queda registrada.
- **Diff mínimo para el feed** (`internal/service/ventana.go`): recorta el código al tramo que cambió para que la tarjeta muestre solo el aporte, con tests.
- **Métricas del modelo de negocio** a partir de una tabla de eventos (registro, publicaciones, propuestas, suscripciones), visibles solo para el rol admin.

## Modelo de datos

12 tablas en 11 migraciones: `usuarios`, `perfiles_estudiante`, `empresas`, `publicaciones` (incluye retos), `propuestas_mejora`, `comentarios`, `sesiones_vivo`, `suscripciones`, `refresh_tokens`, `candidatos_guardados`, `eventos` y `defensas`. El endpoint `/health` devuelve la última migración aplicada para confirmar qué versión de la base quedó desplegada.

## API

Todas las rutas van bajo `/api/v1`. Salvo las de `auth`, exigen `Authorization: Bearer <token>`.

| Área | Rutas |
|---|---|
| Autenticación | `POST /auth/registro` · `POST /auth/login` · `POST /auth/refresh` · `POST /auth/logout` · `GET /auth/yo` |
| Publicaciones | `GET /publicaciones` · `POST /publicaciones` · `GET /publicaciones/:id` · `GET /actividad` |
| Mejoras | `POST /publicaciones/:id/propuestas` · `PATCH /propuestas/:id` · `POST /propuestas/:id/verificacion` · `POST /publicaciones/:id/comentarios` |
| Ejecución | `POST /ejecutar/go` |
| Estudiantes | `GET /estudiantes/:id/portafolio` · `PUT /estudiantes/yo/perfil` · `GET /estudiantes/destacados` · `GET /catalogos` |
| Empresas | `POST /retos` · `GET /talento` · `GET/PUT/DELETE /candidatos` · `GET /suscripciones/actual` · `POST /suscripciones` |
| Defensas | `POST /propuestas/:id/defensa` · `PATCH /defensas/:id` · `GET /defensas` |
| Sesiones en vivo | `GET /sesiones` · `POST /sesiones` · `GET /sesiones/:id` · `PATCH /sesiones/:id` |
| Admin | `GET /admin/metricas` |

Roles: `estudiante`, `empresa` y `admin`. Cada ruta declara qué rol la puede usar.

## Correr en local

Requisitos: Go 1.26 y PostgreSQL 15 o superior.

```bash
git clone https://github.com/ArSteven/devconnect.git
cd devconnect
cat > .env <<'EOF'
DATABASE_URL=postgres://postgres:postgres@localhost:5432/devconnect?sslmode=disable
JWT_SECRET=cambia-esto-por-una-cadena-de-al-menos-32-caracteres
ENV=desarrollo
EOF
go run ./cmd/api
```

Las migraciones se aplican solas al arrancar. La API queda en `http://localhost:8080` y `GET /health` confirma la conexión a la base.

```bash
go test ./...
```

## Contexto

Trabajo de grado (modalidad emprendimiento) del programa de Tecnología en Desarrollo de Sistemas Informáticos de las Unidades Tecnológicas de Santander (UTS), línea de Arquitectura y Diseño de Software.

Autor: **Elver Steven Ariza González** · [github.com/ArSteven](https://github.com/ArSteven)
