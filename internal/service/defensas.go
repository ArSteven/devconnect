package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/repository"
)

var (
	ErrNoEsReto          = errors.New("la defensa en vivo es solo para soluciones de retos")
	ErrNoEsTuReto        = errors.New("solo la empresa dueña del reto puede citar a una defensa")
	ErrSolucionRechazada = errors.New("esta solución fue rechazada: ya no se puede citar a defensa")
	ErrFechaDefensa      = errors.New("la defensa debe ser desde ahora hasta 30 días adelante")
	ErrDefensaVigente    = errors.New("esta solución ya tiene una defensa en vivo: cancélala para proponer otra fecha")
	ErrTransicionDefensa = errors.New("la defensa no puede pasar a ese estado")
)

// InvitarDefensa: la empresa dueña del reto, con suscripción vigente, cita a quien envió una solución
// a una sala privada de Jitsi de 15 minutos con nombre aleatorio. El estudiante la ve en su feed,
// en «En vivo» y en su solución.
func (s *PublicacionService) InvitarDefensa(ctx context.Context, empresaID, propuestaID string, in model.NuevaDefensaInput) (*model.Defensa, error) {
	info, err := s.repo.InfoPropuesta(ctx, propuestaID)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return nil, ErrNoEncontrada
	}
	if err != nil {
		return nil, err
	}
	switch {
	case info.TipoPublicacion != "reto":
		return nil, ErrNoEsReto
	case info.AutorPublicacion != empresaID:
		return nil, ErrNoEsTuReto
	case info.Estado == "rechazada":
		return nil, ErrSolucionRechazada
	}
	if err := s.exigirSuscripcion(ctx, empresaID); err != nil {
		return nil, err
	}
	ahora := time.Now()
	if in.IniciaEn.Before(ahora.Add(-5*time.Minute)) || in.IniciaEn.After(ahora.Add(30*24*time.Hour)) {
		return nil, ErrFechaDefensa
	}
	// Nombre de sala aleatorio: nadie más puede adivinarlo para entrar.
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	d := &model.Defensa{
		PropuestaID:  propuestaID,
		EmpresaID:    empresaID,
		EstudianteID: info.AutorPropuesta,
		IniciaEn:     in.IniciaEn,
		Sala:         "devconnect-defensa-" + hex.EncodeToString(b),
	}
	if err := s.repo.CrearDefensa(ctx, d); err != nil {
		if errors.Is(err, repository.ErrDefensaVigente) {
			return nil, ErrDefensaVigente
		}
		return nil, err
	}
	return s.repo.ObtenerDefensa(ctx, d.ID)
}

// CambiarDefensa: invitada -> realizada (aprobada o no) o cancelada; una realizada todavía puede
// aprobarse. Una aprobación no se deshace.
func (s *PublicacionService) CambiarDefensa(ctx context.Context, empresaID, id string, in model.EstadoDefensaInput) (*model.Defensa, error) {
	d, err := s.repo.ObtenerDefensa(ctx, id)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return nil, ErrNoEncontrada
	}
	if err != nil {
		return nil, err
	}
	if d.EmpresaID != empresaID {
		return nil, ErrNoEsTuReto
	}
	valida := (d.Estado == "invitada" && (in.Estado == "realizada" || in.Estado == "cancelada" && !in.Aprobada)) ||
		(d.Estado == "realizada" && in.Estado == "realizada" && in.Aprobada && !d.Aprobada)
	if !valida {
		return nil, ErrTransicionDefensa
	}
	if err := s.repo.CambiarDefensa(ctx, id, d.Estado, in.Estado, in.Aprobada); err != nil {
		if errors.Is(err, repository.ErrNoEncontrado) {
			return nil, ErrTransicionDefensa // otra petición la cambió primero
		}
		return nil, err
	}
	d.Estado, d.Aprobada = in.Estado, in.Aprobada
	return d, nil
}

// Defensas: las de la empresa o el estudiante que pregunta, desde hace una semana en adelante.
func (s *PublicacionService) Defensas(ctx context.Context, usuarioID string) ([]model.Defensa, error) {
	return s.repo.DefensasDe(ctx, usuarioID)
}
