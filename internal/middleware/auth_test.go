package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ArSteven/devconnect/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const secretoPrueba = "secreto-de-prueba-con-mas-de-32-caracteres"

func init() {
	gin.SetMode(gin.TestMode)
}

func tokenPara(t *testing.T, usuarioID, rol string) string {
	t.Helper()
	claims := service.Claims{
		Rol: rol,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   usuarioID,
			Issuer:    "devconnect",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(service.DuracionAcceso)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secretoPrueba))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// enrutador reproduce cómo main.go protege las rutas: autenticación primero, rol después.
func enrutador(t *testing.T) *gin.Engine {
	t.Helper()
	auth, err := service.NuevoAuthService(nil, nil, nil, secretoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(CabecerasSeguridad())
	privado := r.Group("", RequiereAuth(auth))
	privado.GET("/yo", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.GetString(ClaveUsuarioID), "rol": c.GetString(ClaveRol)})
	})
	privado.GET("/talento", RequiereRol("empresa"), func(c *gin.Context) { c.Status(http.StatusOK) })
	privado.GET("/admin/metricas", RequiereRol("admin"), func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func pedir(r *gin.Engine, ruta, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, ruta, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func codigoDeError(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var cuerpo struct {
		Codigo string `json:"codigo"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("la respuesta de error no es JSON: %q", w.Body.String())
	}
	return cuerpo.Codigo
}

func TestRequiereAuthRechazaPeticionesSinTokenValido(t *testing.T) {
	r := enrutador(t)
	casos := map[string]string{
		"sin cabecera":                "",
		"esquema distinto":            "Basic dXN1YXJpbzpjbGF2ZQ==",
		"Bearer sin token":            "Bearer ",
		"token que no es un JWT":      "Bearer abc.def.ghi",
		"token sin la palabra Bearer": tokenPara(t, "u1", "estudiante"),
	}
	for nombre, authorization := range casos {
		t.Run(nombre, func(t *testing.T) {
			w := pedir(r, "/yo", authorization)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("estado %d; se esperaba 401", w.Code)
			}
			if codigo := codigoDeError(t, w); codigo != "NO_AUTENTICADO" {
				t.Errorf("código %q; se esperaba NO_AUTENTICADO", codigo)
			}
		})
	}
}

func TestRequiereAuthDejaElUsuarioEnElContexto(t *testing.T) {
	w := pedir(enrutador(t), "/yo", "Bearer "+tokenPara(t, "u-42", "estudiante"))
	if w.Code != http.StatusOK {
		t.Fatalf("estado %d; se esperaba 200. Cuerpo: %s", w.Code, w.Body.String())
	}
	var cuerpo struct{ ID, Rol string }
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	if cuerpo.ID != "u-42" || cuerpo.Rol != "estudiante" {
		t.Errorf("contexto inesperado: %+v", cuerpo)
	}
}

func TestRequiereRol(t *testing.T) {
	r := enrutador(t)
	casos := []struct {
		nombre, ruta, rol string
		estado            int
	}{
		{"empresa entra a talento", "/talento", "empresa", http.StatusOK},
		{"estudiante no entra a talento", "/talento", "estudiante", http.StatusForbidden},
		{"admin entra a métricas", "/admin/metricas", "admin", http.StatusOK},
		{"empresa no entra a métricas", "/admin/metricas", "empresa", http.StatusForbidden},
		{"estudiante no entra a métricas", "/admin/metricas", "estudiante", http.StatusForbidden},
		{"rol inventado no entra", "/talento", "superusuario", http.StatusForbidden},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			w := pedir(r, c.ruta, "Bearer "+tokenPara(t, "u1", c.rol))
			if w.Code != c.estado {
				t.Fatalf("estado %d; se esperaba %d", w.Code, c.estado)
			}
			if c.estado == http.StatusForbidden {
				if codigo := codigoDeError(t, w); codigo != "ROL_NO_PERMITIDO" {
					t.Errorf("código %q; se esperaba ROL_NO_PERMITIDO", codigo)
				}
			}
		})
	}
}

func TestCabecerasSeguridad(t *testing.T) {
	// Incluso una respuesta de error lleva las cabeceras.
	w := pedir(enrutador(t), "/yo", "")
	esperadas := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
		"Cache-Control":          "no-store",
	}
	for cabecera, valor := range esperadas {
		if obtenido := w.Header().Get(cabecera); obtenido != valor {
			t.Errorf("%s = %q; se esperaba %q", cabecera, obtenido, valor)
		}
	}
}

func TestLimitadorFrenaFuerzaBruta(t *testing.T) {
	l := NuevoLimitador(3, time.Minute)
	for i := 1; i <= 3; i++ {
		if !l.Permitir("ana@ejemplo.com") {
			t.Fatalf("el intento %d fue bloqueado y estaba dentro del límite", i)
		}
	}
	if l.Permitir("ana@ejemplo.com") {
		t.Fatal("el cuarto intento en menos de un minuto no fue bloqueado")
	}
	// Bloquear un correo no afecta a los demás.
	if !l.Permitir("otro@ejemplo.com") {
		t.Fatal("el límite de un correo bloqueó a otro")
	}
}

func TestLimiteCuerpo(t *testing.T) {
	r := gin.New()
	r.Use(LimiteCuerpo(10))
	r.POST("/eco", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		var demasiado *http.MaxBytesError
		if errors.As(err, &demasiado) {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})

	enviar := func(cuerpo string) int {
		req := httptest.NewRequest(http.MethodPost, "/eco", strings.NewReader(cuerpo))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	if estado := enviar("corto"); estado != http.StatusOK {
		t.Errorf("un cuerpo dentro del límite fue rechazado (estado %d)", estado)
	}
	if estado := enviar("mucho más de diez bytes"); estado != http.StatusRequestEntityTooLarge {
		t.Errorf("un cuerpo mayor al límite fue aceptado (estado %d)", estado)
	}
}
