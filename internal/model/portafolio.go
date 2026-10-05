package model

import "time"

type PerfilEstudiante struct {
	ID          string   `json:"id"`
	Nombre      string   `json:"nombre"`
	Programa    string   `json:"programa"`
	Institucion string   `json:"institucion"`
	Ciudad      string   `json:"ciudad"`
	Stack       []string `json:"stack"`
	Biografia   string   `json:"biografia"`
}

type TotalesPortafolio struct {
	Publicaciones    int `json:"publicaciones"`
	MejorasAportadas int `json:"mejoras_aportadas"`
	MejorasRecibidas int `json:"mejoras_recibidas"`
	Sesiones         int `json:"sesiones"`
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
	Programa    string   `json:"programa" binding:"max=150"`
	Institucion string   `json:"institucion" binding:"max=150"`
	Ciudad      string   `json:"ciudad" binding:"max=100"`
	Stack       []string `json:"stack" binding:"max=15,dive,min=1,max=30"`
	Biografia   string   `json:"biografia" binding:"max=500"`
}

type SuscribirseInput struct {
	Periodo string `json:"periodo" binding:"required,oneof=mensual anual"`
}
