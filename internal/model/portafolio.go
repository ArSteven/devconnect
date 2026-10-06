package model

import "time"

type Experiencia struct {
	Cargo       string `json:"cargo" binding:"required,max=100"`
	Empresa     string `json:"empresa" binding:"required,max=100"`
	Inicio      string `json:"inicio" binding:"required,datetime=2006-01"`  // AAAA-MM
	Fin         string `json:"fin" binding:"omitempty,datetime=2006-01"`     // vacío = actual
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
}

type TotalesPortafolio struct {
	Publicaciones    int `json:"publicaciones"`
	PropuestasHechas int `json:"propuestas_hechas"`
	MejorasAportadas int `json:"mejoras_aportadas"`
	MejorasRecibidas int `json:"mejoras_recibidas"`
	Colaboradores    int `json:"colaboradores"`
	Sesiones         int `json:"sesiones"`
}

// Habilidad: evidencia real por lenguaje, no lo que el estudiante dice saber.
type Habilidad struct {
	Lenguaje      string `json:"lenguaje"`
	Publicaciones int    `json:"publicaciones"`
	Aportes       int    `json:"aportes"`
}

// Destacado: una mejora suya que otra persona aceptó.
type Destacado struct {
	PropuestaID   string    `json:"propuesta_id"`
	PublicacionID string    `json:"publicacion_id"`
	Titulo        string    `json:"titulo"`
	Lenguaje      string    `json:"lenguaje"`
	AutorOriginal string    `json:"autor_original"`
	Explicacion   string    `json:"explicacion"`
	Fecha         time.Time `json:"fecha"`
}

type DiaActividad struct {
	Fecha string `json:"fecha"` // AAAA-MM-DD
	Total int    `json:"total"`
}

// EventoHistorial es un "commit" del portafolio.
// Tipo: publicacion, aporte (mejora aceptada en código ajeno), recibida o sesion.
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
}

type TarjetaTalento struct {
	ID               string   `json:"id"`
	Nombre           string   `json:"nombre"`
	Ciudad           string   `json:"ciudad"`
	Institucion      string   `json:"institucion"`
	Stack            []string `json:"stack"`
	Publicaciones    int      `json:"publicaciones"`
	MejorasAportadas int      `json:"mejoras_aportadas"`
}

type FiltroTalento struct {
	Lenguajes   []string
	Ciudad      string
	ConMejoras  bool
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
}

type SuscribirseInput struct {
	Periodo string `json:"periodo" binding:"required,oneof=mensual anual"`
}
