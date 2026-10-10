package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/repository"
)

const (
	porPagina    = 20
	maxActividad = 20 // mejoras aceptadas que el feed intercala entre las publicaciones
	// MinExplicacionReto: una solución de reto explica sus decisiones, no solo el resultado.
	MinExplicacionReto = 100
)

var (
	ErrNoEncontrada        = errors.New("no existe")
	ErrPropiaPublicacion   = errors.New("no puedes proponer mejoras a tu propia publicación")
	ErrNoEsAutor           = errors.New("solo el autor de la publicación puede decidir sobre sus propuestas")
	ErrYaDecidida          = errors.New("esta propuesta ya fue aceptada o rechazada")
	ErrRequiereSuscripcion = errors.New("esta función requiere una suscripción de empresa activa")
	ErrRetoCerrado         = errors.New("este reto ya cerró: la fecha límite pasó")
	ErrFechaLimiteReto     = errors.New("la fecha límite debe estar entre una hora y 90 días desde ahora")
	ErrExplicacionCorta    = errors.New("explica tus decisiones en al menos 100 caracteres: qué cambiaste, qué alternativas descartaste y qué harías con más tiempo")
	ErrUsoIA               = errors.New("indica si usaste inteligencia artificial para resolver el reto")
)

type PublicacionService struct {
	repo          *repository.PublicacionRepo
	suscripciones *repository.SuscripcionRepo
}

func NuevoPublicacionService(r *repository.PublicacionRepo, s *repository.SuscripcionRepo) *PublicacionService {
	return &PublicacionService{repo: r, suscripciones: s}
}

// Listar normaliza la búsqueda y la institución para que "Girón" encuentre "giron".
func (s *PublicacionService) Listar(ctx context.Context, f model.FiltroPublicaciones) ([]model.Publicacion, error) {
	if f.Pagina < 1 {
		f.Pagina = 1
	}
	f.Q = normalizarTexto(f.Q)
	f.Institucion = normalizarTexto(f.Institucion)
	lista, err := s.repo.Listar(ctx, f, porPagina, (f.Pagina-1)*porPagina)
	if err != nil || len(lista) == 0 {
		return lista, err
	}
	if err := s.agregarMejoras(ctx, lista); err != nil {
		return nil, err
	}
	return lista, nil
}

// agregarMejoras pone en cada publicación mejorada su mejora aceptada más reciente,
// recortada a lo que muestra la tarjeta del feed.
func (s *PublicacionService) agregarMejoras(ctx context.Context, lista []model.Publicacion) error {
	ids := make([]string, len(lista))
	for i, p := range lista {
		ids[i] = p.ID
	}
	mejoras, err := s.repo.MejorasAceptadas(ctx, ids)
	if err != nil {
		return err
	}
	ultima := map[string]model.MejoraCompleta{}
	for _, m := range mejoras { // llegan de la más reciente a la más antigua
		if _, ya := ultima[m.PublicacionID]; !ya {
			ultima[m.PublicacionID] = m
		}
	}
	for i := range lista {
		if m, ok := ultima[lista[i].ID]; ok {
			lista[i].Mejora = &model.MejoraAceptada{Autor: m.Contribuyente, Ventana: ventanaTarjeta(m.Original, m.Codigo)}
		}
	}
	return nil
}

// Actividad devuelve las últimas mejoras aceptadas, cada una con el tramo de su primer cambio.
func (s *PublicacionService) Actividad(ctx context.Context, f model.FiltroActividad) ([]model.Actividad, error) {
	f.Institucion = normalizarTexto(f.Institucion)
	mejoras, err := s.repo.MejorasRecientes(ctx, f, maxActividad)
	if err != nil {
		return nil, err
	}
	lista := make([]model.Actividad, len(mejoras))
	for i, m := range mejoras {
		lista[i] = model.Actividad{
			PropuestaID: m.PropuestaID, PublicacionID: m.PublicacionID,
			Titulo: m.Titulo, Lenguaje: m.Lenguaje, Tipo: m.Tipo,
			Autor: m.AutorPublicacion, Contribuyente: m.Contribuyente, AceptadaEn: m.AceptadaEn,
			Cambio: ventanaCambio(m.Original, m.Codigo),
		}
	}
	return lista, nil
}

func (s *PublicacionService) Crear(ctx context.Context, autorID string, in model.NuevaPublicacionInput) (*model.Publicacion, error) {
	p := &model.Publicacion{
		AutorID:     autorID,
		Titulo:      strings.TrimSpace(in.Titulo),
		Descripcion: strings.TrimSpace(in.Descripcion),
		Lenguaje:    in.Lenguaje,
		Codigo:      in.Codigo, // el código se guarda tal cual: es texto, nunca se ejecuta
		Tipo:        "pregunta",
	}
	if in.Nivel != "" {
		p.Nivel = &in.Nivel
	}
	if err := s.repo.Crear(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// CrearReto: solo una empresa con suscripción vigente publica retos. La suscripción
// se consulta aquí, en el momento, no se confía en lo que diga el frontend.
func (s *PublicacionService) CrearReto(ctx context.Context, empresaID string, in model.NuevoRetoInput) (*model.Publicacion, error) {
	if err := s.exigirSuscripcion(ctx, empresaID); err != nil {
		return nil, err
	}
	ahora := time.Now()
	if in.FechaLimite.Before(ahora.Add(time.Hour)) || in.FechaLimite.After(ahora.Add(90*24*time.Hour)) {
		return nil, ErrFechaLimiteReto
	}
	limite := in.FechaLimite
	p := &model.Publicacion{
		AutorID:     empresaID,
		Titulo:      strings.TrimSpace(in.Titulo),
		Descripcion: strings.TrimSpace(in.Descripcion),
		Lenguaje:    in.Lenguaje,
		Codigo:      in.Codigo,
		Tipo:        "reto",
		FechaLimite: &limite,
	}
	if err := s.repo.Crear(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Detalle: la declaración de uso de IA y la defensa en vivo de una solución solo las ven la empresa
// dueña del reto y quien envió la solución.
func (s *PublicacionService) Detalle(ctx context.Context, id, visitanteID string) (*model.DetallePublicacion, error) {
	p, err := s.repo.Obtener(ctx, id)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return nil, ErrNoEncontrada
	}
	if err != nil {
		return nil, err
	}
	propuestas, err := s.repo.ListarPropuestas(ctx, id)
	if err != nil {
		return nil, err
	}
	defensas := map[string]model.Defensa{}
	if p.Tipo == "reto" {
		lista, err := s.repo.DefensasVigentes(ctx, id)
		if err != nil {
			return nil, err
		}
		for _, d := range lista {
			defensas[d.PropuestaID] = d
		}
	}
	for i := range propuestas {
		pr := &propuestas[i]
		if visitanteID != p.AutorID && visitanteID != pr.AutorID {
			pr.UsoIA, pr.UsoIADetalle = "", ""
			continue
		}
		if d, ok := defensas[pr.ID]; ok {
			pr.Defensa = &d
		}
	}
	comentarios, err := s.repo.ListarComentarios(ctx, id)
	if err != nil {
		return nil, err
	}
	return &model.DetallePublicacion{Publicacion: *p, ListaPropuestas: propuestas, ListaComentarios: comentarios}, nil
}

// Proponer: cualquier estudiante puede proponer una mejora, excepto el autor.
// En un reto, solo mientras no haya pasado la fecha límite, explicando sus decisiones en al menos
// 100 caracteres y declarando si usó inteligencia artificial.
func (s *PublicacionService) Proponer(ctx context.Context, autorID, publicacionID string, in model.NuevaPropuestaInput) (*model.Propuesta, error) {
	pub, err := s.repo.Obtener(ctx, publicacionID)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return nil, ErrNoEncontrada
	}
	if err != nil {
		return nil, err
	}
	if pub.AutorID == autorID {
		return nil, ErrPropiaPublicacion
	}
	if pub.Tipo == "reto" && pub.FechaLimite != nil && time.Now().After(*pub.FechaLimite) {
		return nil, ErrRetoCerrado
	}
	p := &model.Propuesta{
		PublicacionID: publicacionID,
		AutorID:       autorID,
		Codigo:        in.Codigo,
		Explicacion:   strings.TrimSpace(in.Explicacion),
	}
	if pub.Tipo == "reto" {
		if utf8.RuneCountInString(p.Explicacion) < MinExplicacionReto {
			return nil, ErrExplicacionCorta
		}
		if in.UsoIA == "" {
			return nil, ErrUsoIA
		}
		p.UsoIA, p.UsoIADetalle = in.UsoIA, strings.TrimSpace(in.UsoIADetalle)
	}
	if err := s.repo.CrearPropuesta(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Decidir: solo el autor de la publicación acepta o rechaza, y solo una vez.
// Una propuesta aceptada es la que después cuenta en el portafolio de quien la hizo.
// Si quien decide es una empresa (su reto), elegir ganador exige suscripción vigente.
func (s *PublicacionService) Decidir(ctx context.Context, usuarioID, rol, propuestaID, estado string) error {
	info, err := s.repo.InfoPropuesta(ctx, propuestaID)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return ErrNoEncontrada
	}
	if err != nil {
		return err
	}
	if info.AutorPublicacion != usuarioID {
		return ErrNoEsAutor
	}
	if rol == "empresa" {
		if err := s.exigirSuscripcion(ctx, usuarioID); err != nil {
			return err
		}
	}
	if info.Estado != "pendiente" {
		return ErrYaDecidida
	}
	err = s.repo.Decidir(ctx, propuestaID, estado)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return ErrYaDecidida
	}
	return err
}

func (s *PublicacionService) Comentar(ctx context.Context, autorID, publicacionID string, in model.NuevoComentarioInput) (*model.Comentario, error) {
	if _, err := s.repo.Obtener(ctx, publicacionID); err != nil {
		if errors.Is(err, repository.ErrNoEncontrado) {
			return nil, ErrNoEncontrada
		}
		return nil, err
	}
	c := &model.Comentario{AutorID: autorID, Texto: strings.TrimSpace(in.Texto)}
	if err := s.repo.CrearComentario(ctx, publicacionID, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *PublicacionService) exigirSuscripcion(ctx context.Context, empresaID string) error {
	_, err := s.suscripciones.Activa(ctx, empresaID)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return ErrRequiereSuscripcion
	}
	return err
}
