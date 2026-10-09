package model

import "time"

type Publicacion struct {
	ID               string     `json:"id"`
	AutorID          string     `json:"autor_id"`
	AutorNombre      string     `json:"autor_nombre"` // en un reto, la razón social de la empresa
	AutorInstitucion string     `json:"autor_institucion"`
	AutorGithub      string     `json:"autor_github"`
	Titulo           string     `json:"titulo"`
	Descripcion      string     `json:"descripcion"`
	Lenguaje         string     `json:"lenguaje"`
	Codigo           string     `json:"codigo"`
	TotalLineas      int        `json:"total_lineas"` // en el feed solo llegan las primeras líneas del código
	Estado           string     `json:"estado"`
	Tipo             string     `json:"tipo"` // pregunta o reto
	FechaLimite      *time.Time `json:"fecha_limite"`
	Propuestas       int        `json:"total_propuestas"`
	Comentarios      int        `json:"total_comentarios"`
	CreadoEn         time.Time  `json:"creado_en"`
}

type Propuesta struct {
	ID            string    `json:"id"`
	PublicacionID string    `json:"publicacion_id"`
	AutorID       string    `json:"autor_id"`
	AutorNombre   string    `json:"autor_nombre"`
	AutorGithub   string    `json:"autor_github"`
	Codigo        string    `json:"codigo"`
	Explicacion   string    `json:"explicacion"`
	Estado        string    `json:"estado"`
	CreadoEn      time.Time `json:"creado_en"`
}

type Comentario struct {
	ID          string    `json:"id"`
	AutorID     string    `json:"autor_id"`
	AutorNombre string    `json:"autor_nombre"`
	AutorGithub string    `json:"autor_github"`
	AutorRol    string    `json:"autor_rol"`
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
	TipoPublicacion  string
}

// FiltroPublicaciones son los parámetros de GET /publicaciones. Todos se combinan.
type FiltroPublicaciones struct {
	Q           string `form:"q" binding:"max=100"`
	Lenguaje    string `form:"lenguaje" binding:"omitempty,oneof=go angular typescript javascript python java php sql otro"`
	Institucion string `form:"institucion" binding:"max=150"`
	Estado      string `form:"estado" binding:"omitempty,oneof=abierta resuelta"`
	Tipo        string `form:"tipo" binding:"omitempty,oneof=pregunta reto"`
	Pagina      int    `form:"pagina" binding:"omitempty,min=1,max=1000"`
}

type NuevaPublicacionInput struct {
	Titulo      string `json:"titulo" binding:"required,min=5,max=150"`
	Descripcion string `json:"descripcion" binding:"max=2000"`
	Lenguaje    string `json:"lenguaje" binding:"required,oneof=go angular typescript javascript python java php sql otro"`
	Codigo      string `json:"codigo" binding:"required,max=20000"`
}

// NuevoRetoInput: el código inicial es opcional (puede ser una plantilla o nada).
type NuevoRetoInput struct {
	Titulo      string    `json:"titulo" binding:"required,min=5,max=150"`
	Descripcion string    `json:"descripcion" binding:"required,min=20,max=2000"`
	Lenguaje    string    `json:"lenguaje" binding:"required,oneof=go angular typescript javascript python java php sql otro"`
	Codigo      string    `json:"codigo" binding:"max=20000"`
	FechaLimite time.Time `json:"fecha_limite" binding:"required"`
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
