# Decisiones de arquitectura

Las decisiones de diseño de DevConnect que más pesan, qué alternativas había y qué se sacrifica con cada una. Cada entrada señala dónde está en el código.

---

## 1. El servidor nunca ejecuta código de usuarios

**Contexto.** La función central de DevConnect («Pruébalo») es ejecutar el código original y la mejora lado a lado. Ejecutar código arbitrario de desconocidos en el servidor es la forma más directa de que te tomen la máquina.

**Decisión.** JavaScript, TypeScript, Python (Pyodide) y SQL (sql.js) corren en el navegador de quien ejecuta, dentro de un iframe con `sandbox="allow-scripts"` (sin `allow-same-origin`) y una CSP que solo permite el CDN de los intérpretes. Go se reenvía al Go Playground oficial. La API solo recibe y guarda la *constancia* de que la mejora corrió y cambió la salida.

**Alternativas descartadas.** Contenedores aislados por ejecución (Docker/Firecracker): seguros, pero caros de operar para un prototipo y una superficie de ataque que habría que mantener.

**Lo que se sacrifica.** Depende de internet y de servicios externos (jsDelivr, go.dev). La verificación ocurre en el navegador de un usuario, así que es una constancia reproducible, no una prueba criptográfica: cualquiera puede volver a ejecutarla para comprobarla.

**Código.** `internal/service/ejecucion.go`, `internal/service/verificacion.go`; en el frontend `public/ejecutor.html`, `src/app/core/ejecucion.ts` y la CSP en `vercel.json`.

---

## 2. Sesión: token de acceso corto en memoria + refresh token opaco de un solo uso

**Contexto.** Un JWT de larga duración no se puede revocar; guardar tokens en `localStorage` los expone a cualquier XSS.

**Decisión.**
- Token de acceso JWT HS256 de **15 minutos**, que el frontend guarda solo en memoria (un signal), nunca en `localStorage`.
- Refresh token **aleatorio y opaco** (32 bytes), en una cookie `HttpOnly`, `Secure`, `SameSite=Strict` y con `Path=/api/v1/auth`, así que JavaScript no lo puede leer y el navegador solo lo envía a las rutas de autenticación.
- En la base se guarda **solo el SHA-256** del refresh token.
- Al renovar, el token se consume con un único `UPDATE … WHERE revocado_en IS NULL AND expira_en > now() RETURNING`: dos peticiones simultáneas con el mismo token no pueden usarlo las dos.
- La validación del JWT fija el algoritmo (`HS256`), el emisor y exige vencimiento, lo que cierra los ataques de `alg: none` y de cambio de algoritmo.

**Lo que se sacrifica.** Un token de acceso robado sigue sirviendo hasta 15 minutos. Cada renovación escribe en la base.

**Código.** `internal/service/auth.go`, `internal/repository/tokens.go`, `internal/handler/auth.go`. Pruebas: `internal/service/auth_test.go` y `auth_integracion_test.go` (incluye el caso concurrente).

---

## 3. El login no revela qué correos existen

**Decisión.** Correo inexistente y contraseña incorrecta devuelven el mismo error. Cuando el correo no existe se compara igual contra un hash bcrypt de relleno, para que el tiempo de respuesta tampoco lo delate. Además, 5 intentos por minuto por correo (token bucket con `x/time/rate`).

**Lo que se sacrifica.** El limitador vive en memoria: con más de una instancia de la API cada una tendría su propio contador. Para escalar habría que moverlo a Redis o a la base.

**Código.** `internal/service/auth.go` (`hashFalso`), `internal/middleware/seguridad.go`.

---

## 4. Tres capas sin interfaces: handler → service → repository

**Decisión.** Los handlers validan la entrada HTTP, los servicios tienen las reglas de negocio y los repositorios el SQL. Los servicios dependen de los repositorios concretos (`*repository.UsuarioRepo`), sin interfaces intermedias.

**Por qué.** Menos código ceremonial en un proyecto de una persona; las interfaces se agregan cuando aparece una segunda implementación real.

**Lo que se sacrifica.** Los servicios no se pueden probar con dobles en memoria: las pruebas que tocan datos usan un PostgreSQL real (en CI lo levanta el workflow). Es más lento, pero prueba también el SQL, que es donde están las garantías importantes (el consumo atómico del refresh token, por ejemplo).

---

## 5. Migraciones SQL propias, embebidas en el binario

**Decisión.** Los archivos `internal/database/migrations/*.sql` se embeben con `go:embed` y se aplican al arrancar, en orden, cada uno en su transacción, registrando la versión en `schema_migrations`. `/health` devuelve la última migración aplicada.

**Alternativas descartadas.** golang-migrate o goose: más funciones (bajar versiones), pero una dependencia y un paso de despliegue más.

**Lo que se sacrifica.** No hay migraciones hacia atrás; un error se corrige con una migración nueva.

---

## 6. Sin CORS: el frontend llama a la API por el mismo dominio

**Decisión.** Vercel reescribe `/api/*` hacia la API en Railway. Para el navegador todo es mismo origen, así que la API no necesita CORS y la cookie `SameSite=Strict` funciona.

**Lo que se sacrifica.** Cada petición pasa por un salto más (Vercel → Railway).

**Código.** `devconnect-web/vercel.json`, `proxy.conf.json` para desarrollo local.

---

## 7. Defensa en vivo con Jitsi y nombre de sala aleatorio

**Contexto.** Una solución a un reto puede venir de una IA. La forma más simple de comprobar autoría es pedirle a la persona que la explique.

**Decisión.** La empresa cita a una sala de Jitsi Meet de 15 minutos con un nombre generado con `crypto/rand`. No hay infraestructura de video propia.

**Lo que se sacrifica.** La privacidad de la sala depende de que el nombre no se filtre, y la llamada depende de un servicio externo.

**Código.** `internal/service/defensas.go`, `devconnect-web/src/app/core/jitsi.ts`.

---

## 8. Métricas del negocio sin frenar al usuario

**Decisión.** Los eventos (inicios de sesión, contactos vistos, suscripciones) se guardan en segundo plano, con su propio tiempo límite y sin la cancelación de la petición original. Si fallan, solo quedan en el log.

**Lo que se sacrifica.** Se puede perder algún evento si la API se reinicia justo en ese momento. Para métricas de validación del modelo de negocio es aceptable.

**Código.** `internal/service/eventos.go`.

---

## 9. Ley 1581 de 2012 en el modelo de datos

**Decisión.** Sin la autorización expresa no se crea la cuenta, y se rechaza antes de calcular el bcrypt. La base guarda cuándo se aceptó y qué versión de los términos; la versión la fija el servidor, no el cliente. El estudiante puede ocultar su correo a las empresas.

**Código.** `internal/service/auth.go` (`VersionTerminos`), migración `006_terminos_y_datos.sql`.
