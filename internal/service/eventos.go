package service

import (
	"context"
	"log"
	"time"

	"github.com/ArSteven/devconnect/internal/repository"
)

const (
	EventoInicioSesion  = "inicio_sesion"
	EventoContactoVisto = "contacto_visto"
	EventoSuscripcion   = "suscripcion"
)

// Eventos registra hechos del modelo de negocio para las métricas. Nunca demora ni hace fallar
// la operación que los origina: se guardan en segundo plano y, si fallan, solo quedan en el log.
type Eventos struct {
	repo *repository.EventoRepo
}

func NuevoEventos(r *repository.EventoRepo) *Eventos {
	return &Eventos{repo: r}
}

// InicioSesion cuenta como mucho un inicio de sesión por usuario y día (login o renovación del token).
func (e *Eventos) InicioSesion(ctx context.Context, usuarioID string) {
	e.enSegundoPlano(ctx, func(ctx context.Context) error {
		return e.repo.RegistrarUnoPorDia(ctx, usuarioID, EventoInicioSesion, nil)
	})
}

// ContactosVistos: una empresa suscrita recibió el correo de esos estudiantes. Una vez por día y estudiante.
func (e *Eventos) ContactosVistos(ctx context.Context, empresaID string, estudiantes []string) {
	if len(estudiantes) == 0 {
		return
	}
	e.enSegundoPlano(ctx, func(ctx context.Context) error {
		return e.repo.RegistrarUnoPorDia(ctx, empresaID, EventoContactoVisto, estudiantes)
	})
}

// Suscripcion: una empresa activó un plan. Cada activación cuenta.
func (e *Eventos) Suscripcion(ctx context.Context, empresaID, suscripcionID string) {
	e.enSegundoPlano(ctx, func(ctx context.Context) error {
		return e.repo.Registrar(ctx, empresaID, EventoSuscripcion, suscripcionID)
	})
}

func (e *Eventos) enSegundoPlano(ctx context.Context, guardar func(context.Context) error) {
	if e == nil {
		return
	}
	// La petición puede terminar antes que el registro: se conserva el contexto sin su cancelación.
	ctx = context.WithoutCancel(ctx)
	go func() {
		ctx, cancelar := context.WithTimeout(ctx, 5*time.Second)
		defer cancelar()
		if err := guardar(ctx); err != nil {
			log.Printf("métricas: no se pudo registrar un evento: %v", err)
		}
	}()
}
