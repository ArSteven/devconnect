package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/ArSteven/devconnect/internal/repository"
)

var (
	ErrNoAceptada   = errors.New("solo se verifica una mejora aceptada")
	ErrPropiaMejora = errors.New("tu mejora la verifica otra persona cuando la ejecuta")
	ErrNoEjecutable = errors.New("este lenguaje todavía no se puede ejecutar en DevConnect")
)

// LenguajesEjecutables son los que «Pruébalo» sabe correr: en el navegador, o Go en el Go Playground.
var LenguajesEjecutables = []string{"javascript", "typescript", "python", "sql", "go"}

// Verificar: otra persona ejecutó en «Pruébalo» las dos versiones de una mejora aceptada, la mejora
// corrió sin errores y su salida cambió. La ejecución ocurre en su navegador; aquí queda la constancia
// para el portafolio de quien hizo la mejora, que cualquiera puede volver a comprobar ejecutándola.
func (s *PublicacionService) Verificar(ctx context.Context, usuarioID, propuestaID string) (time.Time, error) {
	info, err := s.repo.InfoPropuesta(ctx, propuestaID)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return time.Time{}, ErrNoEncontrada
	}
	if err != nil {
		return time.Time{}, err
	}
	switch {
	case info.Estado != "aceptada":
		return time.Time{}, ErrNoAceptada
	case info.AutorPropuesta == usuarioID:
		return time.Time{}, ErrPropiaMejora
	case !slices.Contains(LenguajesEjecutables, info.Lenguaje):
		return time.Time{}, ErrNoEjecutable
	}
	return s.repo.Verificar(ctx, propuestaID, usuarioID)
}
