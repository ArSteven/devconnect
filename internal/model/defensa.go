package model

import "time"

// DuracionDefensaMin: las defensas en vivo son de 15 minutos.
const DuracionDefensaMin = 15

// Defensa es una cita en vivo de 15 minutos, en una sala privada de Jitsi, entre la empresa dueña
// de un reto y quien envió una solución. Estado: invitada, realizada o cancelada.
type Defensa struct {
	ID               string    `json:"id"`
	PropuestaID      string    `json:"propuesta_id"`
	PublicacionID    string    `json:"publicacion_id"`
	Titulo           string    `json:"titulo"` // el reto
	EmpresaID        string    `json:"empresa_id"`
	EmpresaNombre    string    `json:"empresa_nombre"`
	EstudianteID     string    `json:"estudiante_id"`
	EstudianteNombre string    `json:"estudiante_nombre"`
	EstudianteGithub string    `json:"estudiante_github"`
	IniciaEn         time.Time `json:"inicia_en"`
	DuracionMin      int       `json:"duracion_min"`
	Sala             string    `json:"sala"`
	Estado           string    `json:"estado"`
	Aprobada         bool      `json:"aprobada"`
}

type NuevaDefensaInput struct {
	IniciaEn time.Time `json:"inicia_en" binding:"required"`
}

// EstadoDefensaInput: realizada (con aprobada, si la empresa la aprueba) o cancelada.
type EstadoDefensaInput struct {
	Estado   string `json:"estado" binding:"required,oneof=realizada cancelada"`
	Aprobada bool   `json:"aprobada"`
}
