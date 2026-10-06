package service

import (
	"context"
	"errors"
	"strings"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/repository"
)

var ErrYaSuscrito = errors.New("ya tienes una suscripción activa")

type EstudianteService struct {
	estudiantes   *repository.EstudianteRepo
	suscripciones *repository.SuscripcionRepo
}

func NuevoEstudianteService(e *repository.EstudianteRepo, s *repository.SuscripcionRepo) *EstudianteService {
	return &EstudianteService{estudiantes: e, suscripciones: s}
}

// Portafolio aplica la regla central del modelo freemium: cualquiera con sesión
// ve el portafolio, pero el contacto solo lo ve el propio estudiante o una
// empresa con suscripción vigente.
func (s *EstudianteService) Portafolio(ctx context.Context, visitanteID, visitanteRol, estudianteID string) (*model.Portafolio, error) {
	perfil, correo, err := s.estudiantes.Perfil(ctx, estudianteID)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return nil, ErrNoEncontrada
	}
	if err != nil {
		return nil, err
	}
	totales, err := s.estudiantes.Totales(ctx, estudianteID)
	if err != nil {
		return nil, err
	}
	historial, err := s.estudiantes.Historial(ctx, estudianteID)
	if err != nil {
		return nil, err
	}
	habilidades, err := s.estudiantes.Habilidades(ctx, estudianteID)
	if err != nil {
		return nil, err
	}
	destacados, err := s.estudiantes.Destacados(ctx, estudianteID)
	if err != nil {
		return nil, err
	}
	actividad, err := s.estudiantes.Actividad(ctx, estudianteID)
	if err != nil {
		return nil, err
	}

	p := &model.Portafolio{
		Perfil: *perfil, Totales: totales, Historial: historial,
		Habilidades: habilidades, Destacados: destacados, Actividad: actividad,
	}
	// La tasa de aceptación es la señal de calidad: no cuenta cuánto propone, sino cuánto le aceptan.
	if totales.PropuestasHechas > 0 {
		tasa := (totales.MejorasAportadas*100 + totales.PropuestasHechas/2) / totales.PropuestasHechas
		p.TasaAceptacion = &tasa
	}

	puedeVer := visitanteID == estudianteID
	if !puedeVer && visitanteRol == "empresa" {
		_, err := s.suscripciones.Activa(ctx, visitanteID)
		if err != nil && !errors.Is(err, repository.ErrNoEncontrado) {
			return nil, err
		}
		puedeVer = err == nil
	}
	if puedeVer {
		p.Contacto = &model.Contacto{Correo: correo}
	} else {
		p.ContactoBloqueado = visitanteRol == "empresa"
	}
	return p, nil
}

func (s *EstudianteService) ActualizarPerfil(ctx context.Context, id string, in model.ActualizarPerfilInput) error {
	in.Programa = strings.TrimSpace(in.Programa)
	in.Institucion = strings.TrimSpace(in.Institucion)
	in.Ciudad = strings.TrimSpace(in.Ciudad)
	in.Biografia = strings.TrimSpace(in.Biografia)
	in.Stack = normalizarLista(in.Stack, 15)
	return s.estudiantes.ActualizarPerfil(ctx, id, in)
}

func (s *EstudianteService) BuscarTalento(ctx context.Context, f model.FiltroTalento) ([]model.TarjetaTalento, error) {
	f.Lenguajes = normalizarLista(f.Lenguajes, 10)
	f.Ciudad = strings.TrimSpace(f.Ciudad)
	return s.estudiantes.BuscarTalento(ctx, f)
}

func (s *EstudianteService) SuscripcionActual(ctx context.Context, empresaID string) (*model.Suscripcion, error) {
	sus, err := s.suscripciones.Activa(ctx, empresaID)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return nil, nil
	}
	return sus, err
}

// Suscribirse simula el pago: no hay pasarela en el prototipo.
func (s *EstudianteService) Suscribirse(ctx context.Context, empresaID, periodo string) (*model.Suscripcion, error) {
	actual, err := s.SuscripcionActual(ctx, empresaID)
	if err != nil {
		return nil, err
	}
	if actual != nil {
		return nil, ErrYaSuscrito
	}
	return s.suscripciones.Crear(ctx, empresaID, periodo)
}

// normalizarLista pasa a minúsculas, quita vacíos y repetidos, y limita la cantidad.
func normalizarLista(items []string, maximo int) []string {
	vistos := map[string]bool{}
	lista := []string{}
	for _, it := range items {
		it = strings.ToLower(strings.TrimSpace(it))
		if it == "" || vistos[it] {
			continue
		}
		vistos[it] = true
		lista = append(lista, it)
		if len(lista) == maximo {
			break
		}
	}
	return lista
}
