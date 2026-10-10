package model

import "time"

type Experiencia struct {
	Cargo       string `json:"cargo" binding:"required,max=100"`
	Empresa     string `json:"empresa" binding:"required,max=100"`
	Inicio      string `json:"inicio" binding:"required,datetime=2006-01"` // AAAA-MM
	Fin         string `json:"fin" binding:"omitempty,datetime=2006-01"`   // vacío = actual
	Descripcion string `json:"descripcion" binding:"max=500"`
}

type PerfilEstudiante struct {
	ID              string        `json:"id"`
	Nombre          string        `json:"nombre"`
	Titular         string        `json:"titular"`
	Edad            *int          `json:"edad"`                       // se calcula; la fecha no se expone
	FechaNacimiento string        `json:"fecha_nacimiento,omitempty"` // solo la ve el dueño
	Programa        string        `json:"programa"`
	Institucion     string        `json:"institucion"`
	Semestre        *int          `json:"semestre"`
	EstadoAcademico string        `json:"estado_academico"`
	AnioInicio      *int          `json:"anio_inicio"`
	AnioFin         *int          `json:"anio_fin"`
	Ciudad          string        `json:"ciudad"`
	Disponibilidad  string        `json:"disponibilidad"`
	Modalidad       string        `json:"modalidad"`
	GithubURL       string        `json:"github_url"`
	LinkedinURL     string        `json:"linkedin_url"`
	SitioURL        string        `json:"sitio_url"`
	Stack           []string      `json:"stack"`
	Idiomas         []string      `json:"idiomas"`
	Experiencia     []Experiencia `json:"experiencia"`
	Biografia       string        `json:"biografia"`
	ContactoVisible bool          `json:"contacto_visible"` // false = no muestra el correo a las empresas
}

type TotalesPortafolio struct {
	Publicaciones    int `json:"publicaciones"`
	PropuestasHechas int `json:"propuestas_hechas"`
	MejorasAportadas int `json:"mejoras_aportadas"`
	MejorasRecibidas int `json:"mejoras_recibidas"`
	Colaboradores    int `json:"colaboradores"`
	PersonasAyudadas int `json:"personas_ayudadas"` // autores distintos cuyo código mejoró
	Sesiones         int `json:"sesiones"`
	RetosResueltos   int `json:"retos_resueltos"`
}

// Habilidad: evidencia real por lenguaje, no lo que el estudiante dice saber.
type Habilidad struct {
	Lenguaje      string `json:"lenguaje"`
	Publicaciones int    `json:"publicaciones"`
	Aportes       int    `json:"aportes"`
}

// Destacado: una mejora suya que otra persona aceptó. Si EsReto, AutorOriginal es la empresa.
type Destacado struct {
	PropuestaID     string    `json:"propuesta_id"`
	PublicacionID   string    `json:"publicacion_id"`
	Titulo          string    `json:"titulo"`
	Lenguaje        string    `json:"lenguaje"`
	AutorOriginal   string    `json:"autor_original"`
	EsReto          bool      `json:"es_reto"`
	Explicacion     string    `json:"explicacion"`
	Fecha           time.Time `json:"fecha"`
	Verificada      bool      `json:"verificada"`       // «Mejora verificada por ejecución»
	DefensaAprobada bool      `json:"defensa_aprobada"` // la empresa del reto aprobó su defensa en vivo
}

type DiaActividad struct {
	Fecha string `json:"fecha"` // AAAA-MM-DD
	Total int    `json:"total"`
}

// EventoHistorial es un "commit" del portafolio.
// Tipo: publicacion, aporte (mejora aceptada en código ajeno), reto (solución aceptada
// en el reto de una empresa), recibida o sesion.
type EventoHistorial struct {
	Tipo          string    `json:"tipo"`
	ID            string    `json:"id"`
	PublicacionID string    `json:"publicacion_id"`
	Titulo        string    `json:"titulo"`
	Lenguaje      string    `json:"lenguaje"`
	ConQuien      string    `json:"con_quien"`
	Fecha         time.Time `json:"fecha"`
}

type Contacto struct {
	Correo string `json:"correo"`
}

type Portafolio struct {
	Perfil            PerfilEstudiante  `json:"perfil"`
	Totales           TotalesPortafolio `json:"totales"`
	TasaAceptacion    *int              `json:"tasa_aceptacion"` // % de propuestas aceptadas; null si no ha propuesto
	Habilidades       []Habilidad       `json:"habilidades"`
	Destacados        []Destacado       `json:"destacados"`
	Actividad         []DiaActividad    `json:"actividad"` // últimos 6 meses
	Historial         []EventoHistorial `json:"historial"`
	Contacto          *Contacto         `json:"contacto"`           // null si no tiene permiso
	ContactoBloqueado bool              `json:"contacto_bloqueado"` // true = la empresa debe suscribirse
	ContactoOculto    bool              `json:"contacto_oculto"`    // true = el estudiante decidió no compartir su correo
	Guardado          bool              `json:"guardado"`           // la empresa que lo ve ya lo tiene en su lista
}

type TarjetaTalento struct {
	ID               string   `json:"id"`
	Nombre           string   `json:"nombre"`
	Titular          string   `json:"titular"`
	GithubURL        string   `json:"github_url"`
	Ciudad           string   `json:"ciudad"`
	Institucion      string   `json:"institucion"`
	Programa         string   `json:"programa"`
	Semestre         *int     `json:"semestre"`
	EstadoAcademico  string   `json:"estado_academico"`
	Disponibilidad   string   `json:"disponibilidad"`
	Modalidad        string   `json:"modalidad"`
	Stack            []string `json:"stack"`
	Verificadas      []string `json:"verificadas"` // lenguajes con evidencia en DevConnect
	Publicaciones    int      `json:"publicaciones"`
	PropuestasHechas int      `json:"propuestas_hechas"`
	MejorasAportadas int      `json:"mejoras_aportadas"`
	TasaAceptacion   *int     `json:"tasa_aceptacion"`
	SemanasActivas   int      `json:"semanas_activas"` // de las últimas 8: la constancia
	Guardado         bool     `json:"guardado"`
}

// Candidato es una tarjeta de talento guardada por una empresa. El correo solo
// viene si la empresa tiene suscripción activa.
type Candidato struct {
	TarjetaTalento
	Correo          string    `json:"correo,omitempty"`
	GuardadoEn      time.Time `json:"guardado_en"`
	ContactoVisible bool      `json:"-"` // el service lo usa para decidir si entrega el correo
}

// FiltroTalentoQuery son los parámetros de GET /talento, validados con binding.
type FiltroTalentoQuery struct {
	Lenguaje       string `form:"lenguaje" binding:"max=200"`
	Ciudad         string `form:"ciudad" binding:"max=100"`
	Institucion    string `form:"institucion" binding:"max=150"`
	Nivel          string `form:"nivel" binding:"omitempty,oneof=inicial medio avanzado egresado"`
	Disponibilidad string `form:"disponibilidad" binding:"omitempty,oneof=practicas medio_tiempo tiempo_completo freelance"`
	Modalidad      string `form:"modalidad" binding:"omitempty,oneof=presencial remoto hibrido"`
	ConMejoras     bool   `form:"con_mejoras"`
}

// FiltroTalento ya normalizado por el service.
type FiltroTalento struct {
	Lenguajes      []string
	Ciudades       []string // vacía = cualquier ciudad
	Institucion    string
	Nivel          string
	Disponibilidad string
	Modalidad      string
	ConMejoras     bool
	EmpresaID      string // para marcar los que ya guardó
}

// EstudianteDestacado: quien más mejoras le aceptaron en los últimos días.
type EstudianteDestacado struct {
	ID          string `json:"id"`
	Nombre      string `json:"nombre"`
	Institucion string `json:"institucion"`
	GithubURL   string `json:"github_url"`
	Mejoras     int    `json:"mejoras"`
}

type OpcionCatalogo struct {
	Valor string `json:"valor"`
	Total int    `json:"total"`
}

// Catalogos alimenta los filtros con lo que de verdad existe en la base.
type Catalogos struct {
	Instituciones []OpcionCatalogo `json:"instituciones"`
	Ciudades      []OpcionCatalogo `json:"ciudades"`
}

type Suscripcion struct {
	ID        string    `json:"id"`
	Periodo   string    `json:"periodo"`
	IniciaEn  time.Time `json:"inicia_en"`
	TerminaEn time.Time `json:"termina_en"`
	Estado    string    `json:"estado"`
}

type ActualizarPerfilInput struct {
	Titular         string        `json:"titular" binding:"max=120"`
	FechaNacimiento string        `json:"fecha_nacimiento" binding:"omitempty,datetime=2006-01-02"`
	Programa        string        `json:"programa" binding:"max=150"`
	Institucion     string        `json:"institucion" binding:"max=150"`
	Semestre        *int          `json:"semestre" binding:"omitempty,min=1,max=12"`
	EstadoAcademico string        `json:"estado_academico" binding:"omitempty,oneof=cursando egresado"`
	AnioInicio      *int          `json:"anio_inicio" binding:"omitempty,min=1990,max=2040"`
	AnioFin         *int          `json:"anio_fin" binding:"omitempty,min=1990,max=2045"`
	Ciudad          string        `json:"ciudad" binding:"max=100"`
	Disponibilidad  string        `json:"disponibilidad" binding:"omitempty,oneof=practicas medio_tiempo tiempo_completo freelance no_disponible"`
	Modalidad       string        `json:"modalidad" binding:"omitempty,oneof=presencial remoto hibrido"`
	GithubURL       string        `json:"github_url" binding:"max=200"`
	LinkedinURL     string        `json:"linkedin_url" binding:"max=200"`
	SitioURL        string        `json:"sitio_url" binding:"max=200"`
	Stack           []string      `json:"stack" binding:"max=15,dive,min=1,max=30"`
	Idiomas         []string      `json:"idiomas" binding:"max=6,dive,min=2,max=40"`
	Experiencia     []Experiencia `json:"experiencia" binding:"max=10,dive"`
	Biografia       string        `json:"biografia" binding:"max=1000"`
	ContactoVisible *bool         `json:"contacto_visible"` // nil = no lo cambia
}

type SuscribirseInput struct {
	Periodo string `json:"periodo" binding:"required,oneof=mensual anual"`
}
