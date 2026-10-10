package handler

import (
	"net/http"

	"github.com/ArSteven/devconnect/internal/middleware"
	"github.com/ArSteven/devconnect/internal/model"
	"github.com/gin-gonic/gin"
)

// verificar: POST /propuestas/:id/verificacion — quien probó la mejora en «Pruébalo» deja constancia.
func (h *PublicacionHandler) verificar(c *gin.Context) {
	id, ok := idValido(c)
	if !ok {
		return
	}
	en, err := h.svc.Verificar(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), id)
	if err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "verificada_en": en})
}

// invitarDefensa: POST /propuestas/:id/defensa — la empresa cita a defender una solución de su reto.
func (h *PublicacionHandler) invitarDefensa(c *gin.Context) {
	id, ok := idValido(c)
	if !ok {
		return
	}
	var in model.NuevaDefensaInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "indica la fecha y la hora de la defensa")
		return
	}
	d, err := h.svc.InvitarDefensa(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), id, in)
	if err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"defensa": d})
}

// cambiarDefensa: PATCH /defensas/:id — realizada (y aprobada, si la empresa la aprueba) o cancelada.
func (h *PublicacionHandler) cambiarDefensa(c *gin.Context) {
	id, ok := idValido(c)
	if !ok {
		return
	}
	var in model.EstadoDefensaInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "el estado debe ser realizada o cancelada")
		return
	}
	d, err := h.svc.CambiarDefensa(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), id, in)
	if err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"defensa": d})
}

// defensas: GET /defensas — las defensas en vivo de quien pregunta (empresa o estudiante).
func (h *PublicacionHandler) defensas(c *gin.Context) {
	lista, err := h.svc.Defensas(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID))
	if err != nil {
		errorInterno(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"defensas": lista})
}
