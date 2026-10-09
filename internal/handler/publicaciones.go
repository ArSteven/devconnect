package handler

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/ArSteven/devconnect/internal/middleware"
	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/service"
	"github.com/gin-gonic/gin"
)

var patronUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type PublicacionHandler struct {
	svc *service.PublicacionService
}

func NuevoPublicacionHandler(s *service.PublicacionService) *PublicacionHandler {
	return &PublicacionHandler{svc: s}
}

// Rutas espera un grupo que ya pasa por RequiereAuth.
func (h *PublicacionHandler) Rutas(g *gin.RouterGroup) {
	soloEstudiante := middleware.RequiereRol("estudiante")
	soloEmpresa := middleware.RequiereRol("empresa")
	// La empresa decide sobre las soluciones de sus retos; el service verifica que sea la autora.
	autores := middleware.RequiereRol("estudiante", "empresa")

	g.GET("/publicaciones", h.listar)
	g.POST("/publicaciones", soloEstudiante, h.crear)
	g.GET("/publicaciones/:id", h.detalle)
	g.POST("/publicaciones/:id/propuestas", soloEstudiante, h.proponer)
	g.POST("/publicaciones/:id/comentarios", h.comentar)
	g.PATCH("/propuestas/:id", autores, h.decidir)
	g.POST("/retos", soloEmpresa, h.crearReto)
}

// listar: GET /publicaciones?q=goroutine&lenguaje=go&institucion=uts&estado=abierta&tipo=reto&pagina=2
func (h *PublicacionHandler) listar(c *gin.Context) {
	var f model.FiltroPublicaciones
	if err := c.ShouldBindQuery(&f); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "revisa los filtros de búsqueda")
		return
	}
	lista, err := h.svc.Listar(c.Request.Context(), f)
	if err != nil {
		errorInterno(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"publicaciones": lista, "pagina": max(f.Pagina, 1)})
}

func (h *PublicacionHandler) crearReto(c *gin.Context) {
	var in model.NuevoRetoInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "revisa el título, la descripción (mínimo 20 caracteres) y la fecha límite")
		return
	}
	p, err := h.svc.CrearReto(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), in)
	if err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"publicacion": p})
}

func (h *PublicacionHandler) crear(c *gin.Context) {
	var in model.NuevaPublicacionInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "revisa los datos enviados")
		return
	}
	p, err := h.svc.Crear(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), in)
	if err != nil {
		errorInterno(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"publicacion": p})
}

func (h *PublicacionHandler) detalle(c *gin.Context) {
	id, ok := idValido(c)
	if !ok {
		return
	}
	d, err := h.svc.Detalle(c.Request.Context(), id)
	if err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"publicacion": d})
}

func (h *PublicacionHandler) proponer(c *gin.Context) {
	id, ok := idValido(c)
	if !ok {
		return
	}
	var in model.NuevaPropuestaInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "revisa los datos enviados")
		return
	}
	p, err := h.svc.Proponer(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), id, in)
	if err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"propuesta": p})
}

func (h *PublicacionHandler) decidir(c *gin.Context) {
	id, ok := idValido(c)
	if !ok {
		return
	}
	var in model.DecisionInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "el estado debe ser aceptada o rechazada")
		return
	}
	if err := h.svc.Decidir(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), c.GetString(middleware.ClaveRol), id, in.Estado); err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "estado": in.Estado})
}

func (h *PublicacionHandler) comentar(c *gin.Context) {
	id, ok := idValido(c)
	if !ok {
		return
	}
	var in model.NuevoComentarioInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "revisa los datos enviados")
		return
	}
	com, err := h.svc.Comentar(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID), id, in)
	if err != nil {
		h.fallo(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"comentario": com})
}

// fallo traduce los errores de negocio a respuestas HTTP con código propio.
func (h *PublicacionHandler) fallo(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNoEncontrada):
		responderError(c, http.StatusNotFound, "NO_ENCONTRADO", "no existe")
	case errors.Is(err, service.ErrPropiaPublicacion):
		responderError(c, http.StatusForbidden, "PROPIA_PUBLICACION", err.Error())
	case errors.Is(err, service.ErrNoEsAutor):
		responderError(c, http.StatusForbidden, "NO_ES_AUTOR", err.Error())
	case errors.Is(err, service.ErrYaDecidida):
		responderError(c, http.StatusConflict, "YA_DECIDIDA", err.Error())
	case errors.Is(err, service.ErrRequiereSuscripcion):
		responderError(c, http.StatusForbidden, "SUSCRIPCION_REQUERIDA", err.Error())
	case errors.Is(err, service.ErrRetoCerrado):
		responderError(c, http.StatusConflict, "RETO_CERRADO", err.Error())
	case errors.Is(err, service.ErrFechaLimiteReto):
		responderError(c, http.StatusBadRequest, "FECHA_INVALIDA", err.Error())
	default:
		errorInterno(c, err)
	}
}

// idValido rechaza IDs mal formados antes de llegar a la base.
func idValido(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if !patronUUID.MatchString(id) {
		responderError(c, http.StatusNotFound, "NO_ENCONTRADO", "no existe")
		return "", false
	}
	return id, true
}
