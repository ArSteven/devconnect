package handler

import (
	"errors"
	"net/http"

	"github.com/ArSteven/devconnect/internal/middleware"
	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/service"
	"github.com/gin-gonic/gin"
)

type MetricasHandler struct {
	svc *service.MetricasService
}

func NuevoMetricasHandler(s *service.MetricasService) *MetricasHandler {
	return &MetricasHandler{svc: s}
}

// Rutas espera un grupo que ya pasa por RequiereAuth. Las métricas son solo para el administrador.
func (h *MetricasHandler) Rutas(g *gin.RouterGroup) {
	g.GET("/admin/metricas", middleware.RequiereRol("admin"), h.metricas)
}

// metricas: GET /admin/metricas?desde=2026-10-01&hasta=2026-10-31 (las dos fechas son opcionales).
func (h *MetricasHandler) metricas(c *gin.Context) {
	var f model.FiltroMetricas
	if err := c.ShouldBindQuery(&f); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "las fechas van en formato AAAA-MM-DD")
		return
	}
	m, err := h.svc.Calcular(c.Request.Context(), f)
	if errors.Is(err, service.ErrRangoInvalido) {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", err.Error())
		return
	}
	if err != nil {
		errorInterno(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"metricas": m})
}
