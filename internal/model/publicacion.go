package model

import "time"

type Publicacion struct {
	ID          string    `json:"id"`
	AutorID     string    `json:"autor_id"`
	AutorNombre string    `json:"autor_nombre"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	Lenguaje    string    `json:"lenguaje"`
	Codigo      string    `json:"codigo"`
	Estado      string    `json:"estado"`
	Propuestas  int       `json:"total_propuestas"`
	Comentarios int       `json:"total_comentarios"`
	CreadoEn    time.Time `json:"creado_en"`
}

type Propuesta struct {
	ID            string    `json:"id"`
	PublicacionID string    `json:"publicacion_id"`
	AutorID       string    `json:"autor_id"`
	AutorNombre   string    `json:"autor_nombre"`
	Codigo        string    `json:"codigo"`
	Explicacion   string    `json:"explicacion"`
	Estado        string    `json:"estado"`
	CreadoEn      time.Time `json:"creado_en"`
}

type Comentario struct {
	ID          string    `json:"id"`
	AutorID     string    `json:"autor_id"`
	AutorNombre string    `json:"autor_nombre"`
	Texto       string    `json:"texto"`
	CreadoEn    time.Time `json:"creado_en"`
}

// DetallePublicacion es lo que devuelve GET /publicaciones/:id.
type DetallePublicacion struct {
	Publicacion
	ListaPropuestas  []Propuesta  `json:"propuestas"`
	ListaComentarios []Comentario `json:"comentarios"`
}

// PropuestaInfo reúne lo necesario para decidir sobre una propuesta.
type PropuestaInfo struct {
	Estado           string
	PublicacionID    string
	AutorPublicacion string
}

type NuevaPublicacionInput struct {
	Titulo      string `json:"titulo" binding:"required,min=5,max=150"`
	Descripcion string `json:"descripcion" binding:"max=2000"`
	Lenguaje    string `json:"lenguaje" binding:"required,oneof=go angular typescript javascript python java php sql otro"`
	Codigo      string `json:"codigo" binding:"required,max=20000"`
}

type NuevaPropuestaInput struct {
	Codigo      string `json:"codigo" binding:"required,max=20000"`
	Explicacion string `json:"explicacion" binding:"required,min=10,max=2000"`
}

type DecisionInput struct {
	Estado string `json:"estado" binding:"required,oneof=aceptada rechazada"`
}

type NuevoComentarioInput struct {
	Texto string `json:"texto" binding:"required,min=1,max=2000"`
}
