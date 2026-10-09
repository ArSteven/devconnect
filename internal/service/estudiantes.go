package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

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
// empresa con suscripción vigente, y a la empresa solo si el estudiante lo permite.
func (s *EstudianteService) Portafolio(ctx context.Context, visitanteID, visitanteRol, estudianteID string) (*model.Portafolio, error) {
	perfil, nacimiento, correo, err := s.estudiantes.Perfil(ctx, estudianteID)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return nil, ErrNoEncontrada
	}
	if err != nil {
		return nil, err
	}
	// La edad se calcula y se muestra; la fecha de nacimiento solo la ve el dueño.
	if nacimiento != nil {
		edad := calcularEdad(*nacimiento, time.Now())
		perfil.Edad = &edad
		if visitanteID == estudianteID {
			perfil.FechaNacimiento = nacimiento.Format("2006-01-02")
		}
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

	esDueno := visitanteID == estudianteID
	suscrita := false
	if !esDueno && visitanteRol == "empresa" {
		if suscrita, err = s.tieneSuscripcion(ctx, visitanteID); err != nil {
			return nil, err
		}
		if p.Guardado, err = s.estudiantes.EsCandidatoGuardado(ctx, visitanteID, estudianteID); err != nil {
			return nil, err
		}
	}
	switch {
	case esDueno || suscrita && perfil.ContactoVisible:
		p.Contacto = &model.Contacto{Correo: correo}
	case visitanteRol == "empresa" && !perfil.ContactoVisible:
		// El estudiante decidió no compartirlo: a ninguna empresa se le ofrece
		// suscribirse para ver un correo que de todos modos no vería.
		p.ContactoOculto = true
	default:
		p.ContactoBloqueado = visitanteRol == "empresa"
	}
	return p, nil
}

var ErrPerfilInvalido = errors.New("perfil inválido")

func (s *EstudianteService) ActualizarPerfil(ctx context.Context, id string, in model.ActualizarPerfilInput) error {
	limpiar := func(v *string) { *v = strings.TrimSpace(*v) }
	for _, campo := range []*string{&in.Titular, &in.Programa, &in.Institucion, &in.Ciudad, &in.Biografia,
		&in.GithubURL, &in.LinkedinURL, &in.SitioURL} {
		limpiar(campo)
	}
	in.Stack = normalizarLista(in.Stack, 15)
	in.Idiomas = normalizarIdiomas(in.Idiomas)
	if in.Experiencia == nil {
		in.Experiencia = []model.Experiencia{}
	}

	// Los enlaces solo pueden ser https y del sitio que dicen ser: así nadie mete
	// un enlace "javascript:" ni disfraza otro sitio como su LinkedIn.
	if !enlaceValido(in.GithubURL, "github.com") || !enlaceValido(in.LinkedinURL, "linkedin.com") || !enlaceValido(in.SitioURL, "") {
		return fmt.Errorf("%w: los enlaces deben empezar por https:// y apuntar al sitio correcto", ErrPerfilInvalido)
	}
	if in.FechaNacimiento != "" {
		f, err := time.Parse("2006-01-02", in.FechaNacimiento)
		if err != nil {
			return fmt.Errorf("%w: fecha de nacimiento inválida", ErrPerfilInvalido)
		}
		if edad := calcularEdad(f, time.Now()); edad < 15 || edad > 80 {
			return fmt.Errorf("%w: revisa la fecha de nacimiento", ErrPerfilInvalido)
		}
	}
	if in.AnioInicio != nil && in.AnioFin != nil && *in.AnioFin < *in.AnioInicio {
		return fmt.Errorf("%w: el año de finalización no puede ser anterior al de inicio", ErrPerfilInvalido)
	}
	for i := range in.Experiencia {
		e := &in.Experiencia[i]
		limpiar(&e.Cargo)
		limpiar(&e.Empresa)
		limpiar(&e.Descripcion)
		if e.Fin != "" && e.Fin < e.Inicio { // AAAA-MM se compara bien como texto
			return fmt.Errorf("%w: en experiencia, la fecha de fin es anterior a la de inicio", ErrPerfilInvalido)
		}
	}
	return s.estudiantes.ActualizarPerfil(ctx, id, in)
}

func enlaceValido(u, dominio string) bool {
	if u == "" {
		return true
	}
	p, err := url.Parse(u)
	if err != nil || p.Scheme != "https" || p.Host == "" {
		return false
	}
	if dominio == "" {
		return true
	}
	host := strings.ToLower(p.Host)
	return host == dominio || strings.HasSuffix(host, "."+dominio)
}

func calcularEdad(nacimiento, hoy time.Time) int {
	edad := hoy.Year() - nacimiento.Year()
	if hoy.YearDay() < nacimiento.YearDay() {
		edad--
	}
	return edad
}

// normalizarIdiomas conserva mayúsculas ("Inglés B2") pero quita vacíos y repetidos.
func normalizarIdiomas(items []string) []string {
	vistos := map[string]bool{}
	lista := []string{}
	for _, it := range items {
		it = strings.TrimSpace(it)
		clave := strings.ToLower(it)
		if it == "" || vistos[clave] {
			continue
		}
		vistos[clave] = true
		lista = append(lista, it)
		if len(lista) == 6 {
			break
		}
	}
	return lista
}

// AreaMetropolitana es el valor especial del filtro de ciudad: Bucaramanga y su área
// metropolitana, que es donde las empresas buscan practicantes presenciales.
const AreaMetropolitana = "area_metropolitana"

var municipiosAMB = []string{"bucaramanga", "floridablanca", "giron", "piedecuesta"}

func (s *EstudianteService) BuscarTalento(ctx context.Context, empresaID string, q model.FiltroTalentoQuery) ([]model.TarjetaTalento, error) {
	f := model.FiltroTalento{
		Lenguajes:      normalizarLista(strings.Split(q.Lenguaje, ","), 10),
		Institucion:    normalizarTexto(q.Institucion),
		Nivel:          q.Nivel,
		Disponibilidad: q.Disponibilidad,
		Modalidad:      q.Modalidad,
		ConMejoras:     q.ConMejoras,
		EmpresaID:      empresaID,
	}
	switch ciudad := normalizarTexto(q.Ciudad); ciudad {
	case "":
	case AreaMetropolitana:
		f.Ciudades = municipiosAMB
	default:
		f.Ciudades = []string{ciudad}
	}
	return s.estudiantes.BuscarTalento(ctx, f)
}

// maxCandidatos evita que una sola cuenta llene la tabla.
const maxCandidatos = 300

var ErrDemasiadosCandidatos = fmt.Errorf("tu lista ya tiene %d candidatos: quita alguno antes de guardar otro", maxCandidatos)

// Candidatos devuelve la lista guardada. El correo solo sale con suscripción activa
// y si el estudiante permite mostrarlo: la regla se aplica aquí, no en el frontend.
func (s *EstudianteService) Candidatos(ctx context.Context, empresaID string) ([]model.Candidato, bool, error) {
	lista, err := s.estudiantes.ListarCandidatos(ctx, empresaID)
	if err != nil {
		return nil, false, err
	}
	activa, err := s.tieneSuscripcion(ctx, empresaID)
	if err != nil {
		return nil, false, err
	}
	for i := range lista {
		if !activa || !lista[i].ContactoVisible {
			lista[i].Correo = ""
		}
	}
	return lista, activa, nil
}

func (s *EstudianteService) GuardarCandidato(ctx context.Context, empresaID, estudianteID string) error {
	err := s.estudiantes.GuardarCandidato(ctx, empresaID, estudianteID, maxCandidatos)
	switch {
	case errors.Is(err, repository.ErrNoEncontrado):
		return ErrNoEncontrada
	case errors.Is(err, repository.ErrLimite):
		return ErrDemasiadosCandidatos
	}
	return err
}

func (s *EstudianteService) QuitarCandidato(ctx context.Context, empresaID, estudianteID string) error {
	return s.estudiantes.QuitarCandidato(ctx, empresaID, estudianteID)
}

// DestacadosSemana: si en 7 días nadie logró mejoras aceptadas, amplía a 30 para no mostrar un vacío.
func (s *EstudianteService) DestacadosSemana(ctx context.Context) ([]model.EstudianteDestacado, int, error) {
	for _, dias := range []int{7, 30} {
		lista, err := s.estudiantes.DestacadosRecientes(ctx, dias, 5)
		if err != nil || len(lista) > 0 {
			return lista, dias, err
		}
	}
	return []model.EstudianteDestacado{}, 30, nil
}

func (s *EstudianteService) Catalogos(ctx context.Context) (*model.Catalogos, error) {
	return s.estudiantes.Catalogos(ctx)
}

func (s *EstudianteService) tieneSuscripcion(ctx context.Context, empresaID string) (bool, error) {
	_, err := s.suscripciones.Activa(ctx, empresaID)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return false, nil
	}
	return err == nil, err
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
