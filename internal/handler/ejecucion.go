package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/ArSteven/devconnect/internal/middleware"
	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/service"
	"github.com/gin-gonic/gin"
)

// EjecucionHandler atiende «Pruébalo» para Go. JavaScript, TypeScript, Python y SQL corren en el
// navegador de quien prueba; Go lo ejecuta el Go Playground oficial, nunca este servidor.
type EjecucionHandler struct {
	svc    *service.EjecucionService
	limite *middleware.Limitador
}

func NuevoEjecucionHandler(s *service.EjecucionService) *EjecucionHandler {
	return &EjecucionHandler{svc: s, limite: middleware.NuevoLimitador(10, time.Minute)} // 10 por minuto por persona
}

// Rutas espera un grupo que ya pasa por RequiereAuth.
func (h *EjecucionHandler) Rutas(g *gin.RouterGroup) {
	g.POST("/ejecutar/go", h.ejecutarGo)
}

func (h *EjecucionHandler) ejecutarGo(c *gin.Context) {
	var in model.EjecutarInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "falta el código a ejecutar")
		return
	}
	if !h.limite.Permitir(c.GetString(middleware.ClaveUsuarioID)) {
		responderError(c, http.StatusTooManyRequests, "DEMASIADAS_EJECUCIONES", "llegaste al límite de 10 ejecuciones por minuto, espera un momento")
		return
	}
	res, err := h.svc.Go(c.Request.Context(), in.Codigo)
	switch {
	case errors.Is(err, service.ErrCodigoGrande):
		responderError(c, http.StatusRequestEntityTooLarge, "CODIGO_MUY_GRANDE", err.Error())
	case errors.Is(err, service.ErrTiempoAgotado):
		responderError(c, http.StatusGatewayTimeout, "TIEMPO_AGOTADO", err.Error())
	case errors.Is(err, service.ErrPlayground):
		responderError(c, http.StatusBadGateway, "PLAYGROUND_NO_DISPONIBLE", service.ErrPlayground.Error())
	case err != nil:
		errorInterno(c, err)
	default:
		c.JSON(http.StatusOK, gin.H{"resultado": res})
	}
}
