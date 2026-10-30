package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

const secretoPrueba = "secreto-de-prueba-con-mas-de-32-caracteres"

// firmar arma un token de acceso como lo haría un cliente (legítimo o no).
func firmar(t *testing.T, metodo jwt.SigningMethod, clave any, claims Claims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(metodo, claims).SignedString(clave)
	if err != nil {
		t.Fatalf("no se pudo firmar el token de prueba: %v", err)
	}
	return token
}

func claimsValidos(rol string) Claims {
	ahora := time.Now()
	return Claims{
		Rol: rol,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "11111111-2222-3333-4444-555555555555",
			Issuer:    emisor,
			IssuedAt:  jwt.NewNumericDate(ahora),
			ExpiresAt: jwt.NewNumericDate(ahora.Add(DuracionAcceso)),
		},
	}
}

func TestValidarTokenAceptaUnTokenBienFormado(t *testing.T) {
	s := &AuthService{secreto: []byte(secretoPrueba)}
	token := firmar(t, jwt.SigningMethodHS256, []byte(secretoPrueba), claimsValidos("empresa"))

	claims, err := s.ValidarToken(token)
	if err != nil {
		t.Fatalf("un token válido fue rechazado: %v", err)
	}
	if claims.Subject != "11111111-2222-3333-4444-555555555555" || claims.Rol != "empresa" {
		t.Errorf("claims inesperados: subject=%q rol=%q", claims.Subject, claims.Rol)
	}
}

// Cada caso es una forma conocida de falsificar o reutilizar un token de acceso.
func TestValidarTokenRechazaTokensManipulados(t *testing.T) {
	s := &AuthService{secreto: []byte(secretoPrueba)}

	vencido := claimsValidos("estudiante")
	vencido.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))

	sinVencimiento := claimsValidos("estudiante")
	sinVencimiento.ExpiresAt = nil

	otroEmisor := claimsValidos("estudiante")
	otroEmisor.Issuer = "otra-app"

	casos := []struct {
		nombre string
		token  string
	}{
		{"firmado con otro secreto", firmar(t, jwt.SigningMethodHS256, []byte("otro-secreto-de-mas-de-32-caracteres!!"), claimsValidos("admin"))},
		{"algoritmo none (sin firma)", firmar(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, claimsValidos("admin"))},
		{"algoritmo distinto a HS256", firmar(t, jwt.SigningMethodHS512, []byte(secretoPrueba), claimsValidos("admin"))},
		{"vencido", firmar(t, jwt.SigningMethodHS256, []byte(secretoPrueba), vencido)},
		{"sin fecha de vencimiento", firmar(t, jwt.SigningMethodHS256, []byte(secretoPrueba), sinVencimiento)},
		{"emitido por otra aplicación", firmar(t, jwt.SigningMethodHS256, []byte(secretoPrueba), otroEmisor)},
		{"texto que no es un JWT", "esto-no-es-un-token"},
		{"vacío", ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			claims, err := s.ValidarToken(c.token)
			if !errors.Is(err, ErrSesionInvalida) {
				t.Fatalf("se esperaba ErrSesionInvalida, se obtuvo err=%v claims=%+v", err, claims)
			}
		})
	}
}

// El rol viaja dentro del token: cambiarlo sin volver a firmar debe invalidarlo.
func TestValidarTokenRechazaRolModificado(t *testing.T) {
	s := &AuthService{secreto: []byte(secretoPrueba)}
	original := firmar(t, jwt.SigningMethodHS256, []byte(secretoPrueba), claimsValidos("estudiante"))
	partes := strings.Split(original, ".")

	falso := firmar(t, jwt.SigningMethodHS256, []byte("cualquier-otro-secreto-de-32-caracteres"), claimsValidos("admin"))
	cuerpoFalso := strings.Split(falso, ".")[1]

	manipulado := partes[0] + "." + cuerpoFalso + "." + partes[2]
	if _, err := s.ValidarToken(manipulado); !errors.Is(err, ErrSesionInvalida) {
		t.Fatalf("un token con el cuerpo cambiado fue aceptado: %v", err)
	}
}

// Sin la autorización de datos (Ley 1581) no se crea la cuenta, y se rechaza antes de tocar la base.
func TestRegistrarExigeAceptarTerminos(t *testing.T) {
	s := &AuthService{} // sin repositorios: si llegara a la base, el test entraría en pánico
	_, err := s.Registrar(context.Background(), model.RegistroInput{
		Nombre: "Ana", Correo: "ana@ejemplo.com", Contrasena: "contrasena-segura", Rol: "estudiante",
		AceptaTerminos: false,
	})
	if !errors.Is(err, ErrTerminosNoAceptados) {
		t.Fatalf("se esperaba ErrTerminosNoAceptados, se obtuvo %v", err)
	}
}

func TestRefrescarSinTokenEsSesionInvalida(t *testing.T) {
	s := &AuthService{}
	if _, err := s.Refrescar(context.Background(), ""); !errors.Is(err, ErrSesionInvalida) {
		t.Fatalf("se esperaba ErrSesionInvalida, se obtuvo %v", err)
	}
}

// En la base solo se guarda el hash del refresh token: debe ser estable, de largo fijo
// y no contener el token original.
func TestHashToken(t *testing.T) {
	a := hashToken("token-a")
	if a != hashToken("token-a") {
		t.Error("el hash del mismo token cambió entre llamadas")
	}
	if a == hashToken("token-b") {
		t.Error("dos tokens distintos produjeron el mismo hash")
	}
	if len(a) != 64 {
		t.Errorf("se esperaban 64 caracteres hexadecimales (SHA-256), hay %d", len(a))
	}
	if strings.Contains(a, "token-a") {
		t.Error("el hash contiene el token en claro")
	}
}

func TestNormalizarCorreo(t *testing.T) {
	casos := map[string]string{
		"  Ana@Ejemplo.COM ": "ana@ejemplo.com",
		"ana@ejemplo.com":    "ana@ejemplo.com",
		"\tANA@UTS.EDU.CO\n": "ana@uts.edu.co",
	}
	for entrada, esperado := range casos {
		if obtenido := normalizarCorreo(entrada); obtenido != esperado {
			t.Errorf("normalizarCorreo(%q) = %q; se esperaba %q", entrada, obtenido, esperado)
		}
	}
}
