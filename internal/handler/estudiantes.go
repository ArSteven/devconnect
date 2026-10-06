package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArSteven/devconnect/internal/middleware"
	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/service"
	"github.com/gin-gonic/gin"
)

type EstudianteHandler struct {
	svc *service.EstudianteService
}

func NuevoEstudianteHandler(s *service.EstudianteService) *EstudianteHandler {
	return &EstudianteHandler{svc: s}
}

// Rutas espera un grupo que ya pasa por RequiereAuth.
func (h *EstudianteHandler) Rutas(g *gin.RouterGroup) {
	soloEstudiante := middleware.RequiereRol("estudiante")
	soloEmpresa := middleware.RequiereRol("empresa")

	g.GET("/estudiantes/:id/portafolio", h.portafolio)
	g.PUT("/estudiantes/yo/perfil", soloEstudiante, h.actualizarPerfil)

	g.GET("/talento", soloEmpresa, h.talento)
	g.GET("/suscripciones/actual", soloEmpresa, h.suscripcionActual)
	g.POST("/suscripciones", soloEmpresa, h.suscribirse)
}

func (h *EstudianteHandler) portafolio(c *gin.Context) {
	id, ok := idValido(c)
	if !ok {
		return
	}
	p, err := h.svc.Portafolio(c.Request.Context(),
		c.GetString(middleware.ClaveUsuarioID), c.GetString(middleware.ClaveRol), id)
	if errors.Is(err, service.ErrNoEncontrada) {
		responderError(c, http.StatusNotFound, "NO_ENCONTRADO", "no existe")
		return
	}
	if err != nil {
		errorInterno(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"portafolio": p})
}

func (h *EstudianteHandler) actualizarPerfil(c *gin.Context) {
	var in model.ActualizarPerfilInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "revisa los datos enviados")
		return
	}
	if err := h.svc.ActualizarPerfil(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), in); err != nil {
		if errors.Is(err, service.ErrPerfilInvalido) {
			responderError(c, http.StatusBadRequest, "PERFIL_INVALIDO", strings.TrimPrefix(err.Error(), "perfil inválido: "))
			return
		}
		errorInterno(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// talento: GET /talento?lenguaje=go,angular&ciudad=bucaramanga&con_mejoras=true
func (h *EstudianteHandler) talento(c *gin.Context) {
	var lenguajes []string
	if l := c.Query("lenguaje"); l != "" {
		lenguajes = strings.Split(l, ",")
	}
	ciudad := c.Query("ciudad")
	if len(ciudad) > 100 {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "ciudad demasiado larga")
		return
	}
	lista, err := h.svc.BuscarTalento(c.Request.Context(), model.FiltroTalento{
		Lenguajes:  lenguajes,
		Ciudad:     ciudad,
		ConMejoras: c.Query("con_mejoras") == "true",
	})
	if err != nil {
		errorInterno(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"talento": lista, "total": len(lista)})
}

func (h *EstudianteHandler) suscripcionActual(c *gin.Context) {
	s, err := h.svc.SuscripcionActual(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID))
	if err != nil {
		errorInterno(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"suscripcion": s}) // null = plan gratuito
}

func (h *EstudianteHandler) suscribirse(c *gin.Context) {
	var in model.SuscribirseInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "el periodo debe ser mensual o anual")
		return
	}
	s, err := h.svc.Suscribirse(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), in.Periodo)
	if errors.Is(err, service.ErrYaSuscrito) {
		responderError(c, http.StatusConflict, "YA_SUSCRITO", err.Error())
		return
	}
	if err != nil {
		errorInterno(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"suscripcion": s, "nota": "pago simulado en el prototipo"})
}
