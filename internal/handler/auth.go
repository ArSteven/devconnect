package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ArSteven/devconnect/internal/middleware"
	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

const (
	cookieRefresh = "dc_refresh"
	rutaCookie    = "/api/v1/auth" // la cookie solo viaja a las rutas de auth
)

type AuthHandler struct {
	auth        *service.AuthService
	limiteLogin *middleware.Limitador
}

func NuevoAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{
		auth:        auth,
		limiteLogin: middleware.NuevoLimitador(5, time.Minute), // 5 intentos por minuto por correo
	}
}

// Rutas registra los endpoints bajo /api/v1/auth.
func (h *AuthHandler) Rutas(g *gin.RouterGroup, requiereAuth gin.HandlerFunc) {
	g.POST("/registro", h.registro)
	g.POST("/login", h.login)
	g.POST("/refresh", h.refresh)
	g.POST("/logout", h.logout)
	g.GET("/yo", requiereAuth, h.yo)
}

func (h *AuthHandler) registro(c *gin.Context) {
	var in model.RegistroInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "revisa los datos enviados")
		return
	}
	sesion, err := h.auth.Registrar(c.Request.Context(), in)
	if errors.Is(err, service.ErrTerminosNoAceptados) {
		responderError(c, http.StatusBadRequest, "TERMINOS_NO_ACEPTADOS", err.Error())
		return
	}
	if errors.Is(err, service.ErrCorreoEnUso) {
		responderError(c, http.StatusConflict, "CORREO_EN_USO", err.Error())
		return
	}
	if err != nil {
		errorInterno(c, err)
		return
	}
	h.responderSesion(c, http.StatusCreated, sesion)
}

func (h *AuthHandler) login(c *gin.Context) {
	var in model.LoginInput
	if err := leerJSON(c, &in); err != nil {
		responderError(c, http.StatusBadRequest, "DATOS_INVALIDOS", "revisa los datos enviados")
		return
	}
	if !h.limiteLogin.Permitir(strings.ToLower(strings.TrimSpace(in.Correo))) {
		responderError(c, http.StatusTooManyRequests, "DEMASIADOS_INTENTOS", "demasiados intentos, espera un minuto")
		return
	}
	sesion, err := h.auth.Login(c.Request.Context(), in)
	if errors.Is(err, service.ErrCredenciales) {
		responderError(c, http.StatusUnauthorized, "CREDENCIALES", err.Error())
		return
	}
	if err != nil {
		errorInterno(c, err)
		return
	}
	h.responderSesion(c, http.StatusOK, sesion)
}

func (h *AuthHandler) refresh(c *gin.Context) {
	refresh, _ := c.Cookie(cookieRefresh)
	sesion, err := h.auth.Refrescar(c.Request.Context(), refresh)
	if errors.Is(err, service.ErrSesionInvalida) {
		borrarCookie(c)
		responderError(c, http.StatusUnauthorized, "NO_AUTENTICADO", err.Error())
		return
	}
	if err != nil {
		errorInterno(c, err)
		return
	}
	h.responderSesion(c, http.StatusOK, sesion)
}

func (h *AuthHandler) logout(c *gin.Context) {
	refresh, _ := c.Cookie(cookieRefresh)
	h.auth.Logout(c.Request.Context(), refresh)
	borrarCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) yo(c *gin.Context) {
	u, err := h.auth.Yo(c.Request.Context(), c.GetString(middleware.ClaveUsuarioID))
	if err != nil {
		responderError(c, http.StatusUnauthorized, "NO_AUTENTICADO", "sesión inválida")
		return
	}
	c.JSON(http.StatusOK, gin.H{"usuario": u})
}

// responderSesion entrega el token de acceso en el cuerpo y el refresh en una
// cookie HttpOnly: JavaScript nunca puede leer el refresh token.
func (h *AuthHandler) responderSesion(c *gin.Context, estado int, s *service.Sesion) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieRefresh,
		Value:    s.RefreshToken,
		Path:     rutaCookie,
		MaxAge:   int(service.DuracionRefresh.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	c.JSON(estado, gin.H{
		"usuario":      s.Usuario,
		"token_acceso": s.TokenAcceso,
		"expira_en":    int(service.DuracionAcceso.Seconds()),
	})
}

func borrarCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: cookieRefresh, Value: "", Path: rutaCookie, MaxAge: -1,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode,
	})
}

// leerJSON decodifica rechazando campos que no existen en el struct
// y luego aplica las validaciones de las etiquetas binding.
func leerJSON(c *gin.Context, destino any) error {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		return err
	}
	return binding.Validator.ValidateStruct(destino)
}

func responderError(c *gin.Context, estado int, codigo, mensaje string) {
	c.JSON(estado, gin.H{"error": mensaje, "codigo": codigo})
}

// errorInterno registra el detalle en el log y al cliente solo le da un mensaje genérico.
func errorInterno(c *gin.Context, err error) {
	log.Printf("error interno en %s %s: %v", c.Request.Method, c.FullPath(), err)
	responderError(c, http.StatusInternalServerError, "ERROR_INTERNO", "algo salió mal, intenta de nuevo")
}
