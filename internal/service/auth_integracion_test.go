package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ArSteven/devconnect/internal/database"
	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Estas pruebas usan una base PostgreSQL real porque lo que verifican (refresh token de un
// solo uso, correos duplicados, revocación) depende de las sentencias SQL, no solo del Go.
//
// Solo corren si existe TEST_DATABASE_URL. Usa una base exclusiva para pruebas:
//
//	createdb devconnect_test
//	TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/devconnect_test?sslmode=disable go test ./...
//
// En GitHub Actions la crea el workflow de CI.

const contrasenaPrueba = "contrasena-de-prueba-123"

func baseDePrueba(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no está definida: se omiten las pruebas con base de datos")
	}
	ctx := context.Background()
	pool, err := database.Conectar(ctx, url)
	if err != nil {
		t.Fatalf("no se pudo conectar a la base de pruebas: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrar(ctx, pool); err != nil {
		t.Fatalf("no se pudieron aplicar las migraciones: %v", err)
	}
	return pool
}

func authDePrueba(t *testing.T, pool *pgxpool.Pool) *AuthService {
	t.Helper()
	// Eventos en nil: las métricas se guardan en segundo plano y no son parte de lo que se prueba.
	s, err := NuevoAuthService(repository.NuevoUsuarioRepo(pool), repository.NuevoTokenRepo(pool), nil, secretoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// registrarEstudiante crea una cuenta con un correo único y la borra al terminar el test.
func registrarEstudiante(t *testing.T, s *AuthService, pool *pgxpool.Pool) (correo string, sesion *Sesion) {
	t.Helper()
	correo = fmt.Sprintf("prueba-%d@devconnect.test", time.Now().UnixNano())
	sesion, err := s.Registrar(context.Background(), model.RegistroInput{
		Nombre: "Estudiante de prueba", Correo: correo, Contrasena: contrasenaPrueba,
		Rol: "estudiante", AceptaTerminos: true,
	})
	if err != nil {
		t.Fatalf("no se pudo registrar la cuenta de prueba: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM usuarios WHERE correo = $1`, correo)
	})
	return correo, sesion
}

func TestRegistroYLogin(t *testing.T) {
	pool := baseDePrueba(t)
	s := authDePrueba(t, pool)
	ctx := context.Background()
	correo, registro := registrarEstudiante(t, s, pool)

	if registro.Usuario.Rol != "estudiante" || registro.TokenAcceso == "" || registro.RefreshToken == "" {
		t.Fatalf("el registro no devolvió una sesión completa: %+v", registro)
	}

	// El correo se normaliza: mayúsculas y espacios no impiden iniciar sesión.
	sesion, err := s.Login(ctx, model.LoginInput{Correo: "  " + strings.ToUpper(correo) + " ", Contrasena: contrasenaPrueba})
	if err != nil {
		t.Fatalf("login con credenciales correctas falló: %v", err)
	}
	claims, err := s.ValidarToken(sesion.TokenAcceso)
	if err != nil {
		t.Fatalf("el token de acceso emitido no es válido: %v", err)
	}
	if claims.Subject != registro.Usuario.ID || claims.Rol != "estudiante" {
		t.Errorf("el token no corresponde al usuario: subject=%q rol=%q", claims.Subject, claims.Rol)
	}
}

// Contraseña incorrecta y correo inexistente devuelven el mismo error: la API no revela
// qué correos están registrados.
func TestLoginNoRevelaSiElCorreoExiste(t *testing.T) {
	pool := baseDePrueba(t)
	s := authDePrueba(t, pool)
	ctx := context.Background()
	correo, _ := registrarEstudiante(t, s, pool)

	_, errClave := s.Login(ctx, model.LoginInput{Correo: correo, Contrasena: "contrasena-equivocada"})
	_, errCorreo := s.Login(ctx, model.LoginInput{Correo: "nadie-" + correo, Contrasena: contrasenaPrueba})

	if !errors.Is(errClave, ErrCredenciales) || !errors.Is(errCorreo, ErrCredenciales) {
		t.Fatalf("se esperaba ErrCredenciales en ambos casos: clave=%v correo=%v", errClave, errCorreo)
	}
}

func TestRegistroRechazaCorreoDuplicado(t *testing.T) {
	pool := baseDePrueba(t)
	s := authDePrueba(t, pool)
	correo, _ := registrarEstudiante(t, s, pool)

	_, err := s.Registrar(context.Background(), model.RegistroInput{
		Nombre: "Otra persona", Correo: strings.ToUpper(correo), Contrasena: contrasenaPrueba,
		Rol: "estudiante", AceptaTerminos: true,
	})
	if !errors.Is(err, ErrCorreoEnUso) {
		t.Fatalf("se esperaba ErrCorreoEnUso, se obtuvo %v", err)
	}
}

// Un refresh token sirve una sola vez: si alguien lo roba y lo usa, el original deja de funcionar.
func TestRefreshTokenEsDeUnSoloUso(t *testing.T) {
	pool := baseDePrueba(t)
	s := authDePrueba(t, pool)
	ctx := context.Background()
	_, sesion := registrarEstudiante(t, s, pool)

	nueva, err := s.Refrescar(ctx, sesion.RefreshToken)
	if err != nil {
		t.Fatalf("el primer uso del refresh token falló: %v", err)
	}
	if nueva.RefreshToken == sesion.RefreshToken {
		t.Fatal("la renovación devolvió el mismo refresh token")
	}
	if _, err := s.Refrescar(ctx, sesion.RefreshToken); !errors.Is(err, ErrSesionInvalida) {
		t.Fatalf("un refresh token ya usado volvió a funcionar: %v", err)
	}
	if _, err := s.Refrescar(ctx, nueva.RefreshToken); err != nil {
		t.Fatalf("el refresh token nuevo no funcionó: %v", err)
	}
}

// Dos peticiones simultáneas con el mismo refresh token: solo una puede ganar.
func TestRefreshTokenConcurrenteSoloFuncionaUnaVez(t *testing.T) {
	pool := baseDePrueba(t)
	s := authDePrueba(t, pool)
	_, sesion := registrarEstudiante(t, s, pool)

	const intentos = 5
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		exitos int
		inicio = make(chan struct{})
	)
	for range intentos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-inicio
			if _, err := s.Refrescar(context.Background(), sesion.RefreshToken); err == nil {
				mu.Lock()
				exitos++
				mu.Unlock()
			}
		}()
	}
	close(inicio)
	wg.Wait()

	if exitos != 1 {
		t.Fatalf("el mismo refresh token se usó %d veces; se esperaba exactamente 1", exitos)
	}
}

func TestLogoutRevocaElRefreshToken(t *testing.T) {
	pool := baseDePrueba(t)
	s := authDePrueba(t, pool)
	ctx := context.Background()
	_, sesion := registrarEstudiante(t, s, pool)

	s.Logout(ctx, sesion.RefreshToken)

	if _, err := s.Refrescar(ctx, sesion.RefreshToken); !errors.Is(err, ErrSesionInvalida) {
		t.Fatalf("el refresh token siguió funcionando después del logout: %v", err)
	}
}

// En la base nunca queda el refresh token en claro, solo su hash.
func TestRefreshTokenSeGuardaHasheado(t *testing.T) {
	pool := baseDePrueba(t)
	s := authDePrueba(t, pool)
	_, sesion := registrarEstudiante(t, s, pool)

	var enClaro, hasheado int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FILTER (WHERE token_hash = $1), count(*) FILTER (WHERE token_hash = $2)
		   FROM refresh_tokens`,
		sesion.RefreshToken, hashToken(sesion.RefreshToken),
	).Scan(&enClaro, &hasheado)
	if err != nil {
		t.Fatal(err)
	}
	if enClaro != 0 || hasheado != 1 {
		t.Fatalf("se esperaba solo el hash en la base: en claro=%d hasheado=%d", enClaro, hasheado)
	}
}
