package handler

import (
	"errors"
	"net/http"

	"github.com/ArSteven/devconnect/internal/middleware"
	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/service"
	"github.com/gin-gonic/gin"
)

type SesionHandler struct {
	svc *service.SesionService
}

func NuevoSesionHandler(s *service.SesionService) *SesionHandler {
	return &SesionHandler{svc: s}
}

// Rutas espera un grupo que ya pasa por RequiereAuth.
func (h *SesionHandler) Rutas(g *gin.RouterGroup) {
	soloEstudiante := middleware.RequiereRol("estudiante")
	g.GET("/sesiones", h.listar)
	g.POST("/sesiones", soloEstudiante, h.crear)
	g.GET("/sesiones/:id", h.obtener)
	g.PATCH("/sesiones/:id", soloEstudiante, h.cambiarEstado)
}

func (h *SesionHandler) listar(c *gin.Context) {
	lista, err := h.svc.Listar(c.Request.Context())
	if err != nil {
		errorInterno(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sesiones": lista})
}

func (h *SesionHandler) obtener(c *gin.Context) {
	id, ok := idValido(c)
	if !ok {
		return
	}
	s, err := h.svc.Obtener(c.Request.Context(), id)
	if err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sesion": s})
}

func (h *SesionHandler) crear(c *gin.Context) {
	var in model.NuevaSesionInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "revisa el título y la fecha")
		return
	}
	s, err := h.svc.Crear(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), in)
	if err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"sesion": s})
}

func (h *SesionHandler) cambiarEstado(c *gin.Context) {
	id, ok := idValido(c)
	if !ok {
		return
	}
	var in model.EstadoSesionInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "el estado debe ser en_vivo o finalizada")
		return
	}
	if err := h.svc.CambiarEstado(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), id, in.Estado); err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "estado": in.Estado})
}

func (h *SesionHandler) fallo(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNoEncontrada):
		responderError(c, http.StatusNotFound, "NO_ENCONTRADO", "no existe")
	case errors.Is(err, service.ErrNoEsAnfitrion):
		responderError(c, http.StatusForbidden, "NO_ES_ANFITRION", err.Error())
	case errors.Is(err, service.ErrFechaSesion):
		responderError(c, http.StatusBadRequest, "FECHA_INVALIDA", err.Error())
	case errors.Is(err, service.ErrTransicion):
		responderError(c, http.StatusConflict, "TRANSICION_INVALIDA", err.Error())
	default:
		errorInterno(c, err)
	}
}
