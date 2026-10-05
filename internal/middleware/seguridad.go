package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// CabecerasSeguridad agrega cabeceras básicas a todas las respuestas de la API.
func CabecerasSeguridad() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		c.Next()
	}
}

// LimiteCuerpo corta cualquier petición cuyo cuerpo supere maxBytes.
func LimiteCuerpo(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// Limitador cuenta intentos por clave (por ejemplo, por correo en el login)
// para frenar ataques de fuerza bruta. Vive en memoria: suficiente para una
// sola instancia de la API, que es como corre el prototipo.
type Limitador struct {
	mu      sync.Mutex
	claves  map[string]*entrada
	tasa    rate.Limit
	rafaga  int
}

type entrada struct {
	lim   *rate.Limiter
	visto time.Time
}

// NuevoLimitador permite `intentos` por `periodo` para cada clave.
func NuevoLimitador(intentos int, periodo time.Duration) *Limitador {
	l := &Limitador{
		claves: make(map[string]*entrada),
		tasa:   rate.Every(periodo / time.Duration(intentos)),
		rafaga: intentos,
	}
	go l.limpiar()
	return l
}

func (l *Limitador) Permitir(clave string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.claves[clave]
	if !ok {
		e = &entrada{lim: rate.NewLimiter(l.tasa, l.rafaga)}
		l.claves[clave] = e
	}
	e.visto = time.Now()
	return e.lim.Allow()
}

// limpiar borra cada minuto las claves sin actividad reciente para no acumular memoria.
func (l *Limitador) limpiar() {
	for range time.Tick(time.Minute) {
		l.mu.Lock()
		for k, e := range l.claves {
			if time.Since(e.visto) > 10*time.Minute {
				delete(l.claves, k)
			}
		}
		l.mu.Unlock()
	}
}
