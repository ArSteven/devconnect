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
	TotalProponentes int        `json:"total_proponentes"` // personas distintas que propusieron mejoras
	Proponentes      []Persona  `json:"proponentes"`       // las primeras cuatro, para los avatares del feed
	// Mejora es la mejora aceptada más reciente, recortada a la zona del cambio. Solo la llena el feed.
	Mejora *MejoraAceptada `json:"mejora"`
}

// Persona es lo mínimo para mostrar a alguien en el feed: su avatar y su nombre.
type Persona struct {
	ID        string `json:"id"`
	Nombre    string `json:"nombre"`
	GithubURL string `json:"github_url"`
}

// Ventana es un tramo del código original y del mejorado alrededor de un cambio. El navegador
// hace el diff sobre ella: así no viajan los archivos completos en cada tarjeta.
type Ventana struct {
	Original    string `json:"original"`
	Codigo      string `json:"codigo"`       // la versión mejorada
	DesdeLinea  int    `json:"desde_linea"`  // número de la primera línea de los dos tramos (desde 1)
	TotalLineas int    `json:"total_lineas"` // líneas de la versión mejorada completa
}

// MejoraAceptada es lo que la tarjeta del feed muestra de una mejora: quién la hizo y el código resultante.
type MejoraAceptada struct {
	Autor Persona `json:"autor"`
	Ventana
}

// MejoraCompleta es una mejora aceptada con los dos códigos completos. El service la recorta a una Ventana.
type MejoraCompleta struct {
	PropuestaID      string
	PublicacionID    string
	Titulo           string
	Lenguaje         string
	Tipo             string
	AutorPublicacion Persona // en un reto, la empresa
	Contribuyente    Persona // quien propuso la mejora
	AceptadaEn       time.Time
	Original         string
	Codigo           string
	Explicacion      string
}

// Transformacion es una mejora aceptada para la sección «Antes y después»: el código original
// se transforma en el mejorado frente a quien mira.
type Transformacion struct {
	PropuestaID   string    `json:"propuesta_id"`
	PublicacionID string    `json:"publicacion_id"`
	Titulo        string    `json:"titulo"`
	Lenguaje      string    `json:"lenguaje"`
	Tipo          string    `json:"tipo"`
	Autor         Persona   `json:"autor"`         // autor de la publicación (en un reto, la empresa)
	Contribuyente Persona   `json:"contribuyente"` // quien hizo la mejora
	Original      string    `json:"original"`
	Codigo        string    `json:"codigo"`
	Explicacion   string    `json:"explicacion"`
	AceptadaEn    time.Time `json:"aceptada_en"`
}

// FiltroTransformaciones son los parámetros de GET /antes-y-despues.
type FiltroTransformaciones struct {
	Pagina int `form:"pagina" binding:"omitempty,min=1,max=1000"`
}

// Actividad es una mejora aceptada contada en el feed: quién mejoró el código de quién.
type Actividad struct {
	PropuestaID   string    `json:"propuesta_id"`
	PublicacionID string    `json:"publicacion_id"`
	Titulo        string    `json:"titulo"`
	Lenguaje      string    `json:"lenguaje"`
	Tipo          string    `json:"tipo"`
	Autor         Persona   `json:"autor"`         // autor de la publicación (en un reto, la empresa)
	Contribuyente Persona   `json:"contribuyente"` // quien hizo la mejora
	AceptadaEn    time.Time `json:"aceptada_en"`
	Cambio        Ventana   `json:"cambio"` // el tramo del primer cambio, para el diff corto
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

// FiltroActividad son los parámetros de GET /actividad: los del feed que tienen sentido para una mejora.
type FiltroActividad struct {
	Lenguaje    string `form:"lenguaje" binding:"omitempty,oneof=go angular typescript javascript python java php sql otro"`
	Institucion string `form:"institucion" binding:"max=150"`
	Tipo        string `form:"tipo" binding:"omitempty,oneof=pregunta reto"`
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
