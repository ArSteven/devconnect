package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/repository"
)

var (
	ErrNoEsAnfitrion = errors.New("solo quien creó la sesión puede cambiar su estado")
	ErrFechaSesion   = errors.New("la fecha debe ser desde ahora hasta 90 días adelante")
	ErrTransicion    = errors.New("la sesión no puede pasar a ese estado")
)

type SesionService struct {
	repo *repository.SesionRepo
}

func NuevoSesionService(r *repository.SesionRepo) *SesionService {
	return &SesionService{repo: r}
}

func (s *SesionService) Listar(ctx context.Context) ([]model.SesionVivo, error) {
	return s.repo.Listar(ctx)
}

func (s *SesionService) Obtener(ctx context.Context, id string) (*model.SesionVivo, error) {
	ses, err := s.repo.Obtener(ctx, id)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return nil, ErrNoEncontrada
	}
	return ses, err
}

func (s *SesionService) Crear(ctx context.Context, anfitrionID string, in model.NuevaSesionInput) (*model.SesionVivo, error) {
	ahora := time.Now()
	if in.IniciaEn.Before(ahora.Add(-10*time.Minute)) || in.IniciaEn.After(ahora.Add(90*24*time.Hour)) {
		return nil, ErrFechaSesion
	}
	// Nombre de sala aleatorio: nadie puede adivinarlo para colarse a una sesión.
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	ses := &model.SesionVivo{
		AnfitrionID: anfitrionID,
		Titulo:      strings.TrimSpace(in.Titulo),
		Descripcion: strings.TrimSpace(in.Descripcion),
		IniciaEn:    in.IniciaEn,
		Sala:        "devconnect-" + hex.EncodeToString(b),
	}
	if err := s.repo.Crear(ctx, ses); err != nil {
		return nil, err
	}
	return ses, nil
}

// CambiarEstado: programada -> en_vivo -> finalizada. Una programada también puede cancelarse (finalizada).
func (s *SesionService) CambiarEstado(ctx context.Context, usuarioID, id, estado string) error {
	ses, err := s.Obtener(ctx, id)
	if err != nil {
		return err
	}
	if ses.AnfitrionID != usuarioID {
		return ErrNoEsAnfitrion
	}
	valida := (ses.Estado == "programada" && (estado == "en_vivo" || estado == "finalizada")) ||
		(ses.Estado == "en_vivo" && estado == "finalizada")
	if !valida {
		return ErrTransicion
	}
	return s.repo.CambiarEstado(ctx, id, estado)
}
