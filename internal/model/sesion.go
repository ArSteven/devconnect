package model

import "time"

type SesionVivo struct {
	ID              string    `json:"id"`
	AnfitrionID     string    `json:"anfitrion_id"`
	AnfitrionNombre string    `json:"anfitrion_nombre"`
	Titulo          string    `json:"titulo"`
	Descripcion     string    `json:"descripcion"`
	IniciaEn        time.Time `json:"inicia_en"`
	Sala            string    `json:"sala"`
	Estado          string    `json:"estado"`
}

type NuevaSesionInput struct {
	Titulo      string    `json:"titulo" binding:"required,min=5,max=150"`
	Descripcion string    `json:"descripcion" binding:"max=1000"`
	IniciaEn    time.Time `json:"inicia_en" binding:"required"`
}

type EstadoSesionInput struct {
	Estado string `json:"estado" binding:"required,oneof=en_vivo finalizada"`
}
