package service

import (
	"context"
	"errors"
	"strings"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/repository"
)

const porPagina = 20

var (
	ErrNoEncontrada      = errors.New("no existe")
	ErrPropiaPublicacion = errors.New("no puedes proponer mejoras a tu propia publicación")
	ErrNoEsAutor         = errors.New("solo el autor de la publicación puede decidir sobre sus propuestas")
	ErrYaDecidida        = errors.New("esta propuesta ya fue aceptada o rechazada")
)

type PublicacionService struct {
	repo *repository.PublicacionRepo
}

func NuevoPublicacionService(r *repository.PublicacionRepo) *PublicacionService {
	return &PublicacionService{repo: r}
}

func (s *PublicacionService) Listar(ctx context.Context, lenguaje string, pagina int) ([]model.Publicacion, error) {
	if pagina < 1 {
		pagina = 1
	}
	return s.repo.Listar(ctx, strings.ToLower(lenguaje), porPagina, (pagina-1)*porPagina)
}

func (s *PublicacionService) Crear(ctx context.Context, autorID string, in model.NuevaPublicacionInput) (*model.Publicacion, error) {
	p := &model.Publicacion{
		AutorID:     autorID,
		Titulo:      strings.TrimSpace(in.Titulo),
		Descripcion: strings.TrimSpace(in.Descripcion),
		Lenguaje:    in.Lenguaje,
		Codigo:      in.Codigo, // el código se guarda tal cual: es texto, nunca se ejecuta
	}
	if err := s.repo.Crear(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PublicacionService) Detalle(ctx context.Context, id string) (*model.DetallePublicacion, error) {
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
	comentarios, err := s.repo.ListarComentarios(ctx, id)
	if err != nil {
		return nil, err
	}
	return &model.DetallePublicacion{Publicacion: *p, ListaPropuestas: propuestas, ListaComentarios: comentarios}, nil
}

// Proponer: cualquier estudiante puede proponer una mejora, excepto el autor.
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
	p := &model.Propuesta{
		PublicacionID: publicacionID,
		AutorID:       autorID,
		Codigo:        in.Codigo,
		Explicacion:   strings.TrimSpace(in.Explicacion),
	}
	if err := s.repo.CrearPropuesta(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Decidir: solo el autor de la publicación acepta o rechaza, y solo una vez.
// Una propuesta aceptada es la que después cuenta en el portafolio de quien la hizo.
func (s *PublicacionService) Decidir(ctx context.Context, usuarioID, propuestaID, estado string) error {
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
