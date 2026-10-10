package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	DuracionAcceso  = 15 * time.Minute
	DuracionRefresh = 7 * 24 * time.Hour
	costoBcrypt     = 12
	emisor          = "devconnect"
	// VersionTerminos es la versión vigente de los términos y la política de datos.
	// La define el servidor: el cliente solo dice si acepta, no qué versión.
	VersionTerminos = "1.0"
)

var (
	ErrCredenciales        = errors.New("correo o contraseña incorrectos")
	ErrCorreoEnUso         = errors.New("ese correo ya está registrado")
	ErrSesionInvalida      = errors.New("sesión inválida o vencida")
	ErrTerminosNoAceptados = errors.New("debes aceptar los términos y condiciones y la política de tratamiento de datos")
)

// Claims es lo que viaja dentro del token de acceso: solo el ID (Subject) y el rol.
type Claims struct {
	Rol string `json:"rol"`
	jwt.RegisteredClaims
}

type Sesion struct {
	Usuario      model.UsuarioPublico
	TokenAcceso  string
	RefreshToken string
}

type AuthService struct {
	usuarios  *repository.UsuarioRepo
	tokens    *repository.TokenRepo
	eventos   *Eventos
	secreto   []byte
	hashFalso []byte
}

func NuevoAuthService(u *repository.UsuarioRepo, t *repository.TokenRepo, e *Eventos, secreto string) (*AuthService, error) {
	// Hash de relleno: si el correo no existe igual comparamos contra algo,
	// para que la respuesta tarde lo mismo y no delate qué correos están registrados.
	hashFalso, err := bcrypt.GenerateFromPassword([]byte("relleno-que-nunca-coincide"), costoBcrypt)
	if err != nil {
		return nil, err
	}
	return &AuthService{usuarios: u, tokens: t, eventos: e, secreto: []byte(secreto), hashFalso: hashFalso}, nil
}

func normalizarCorreo(c string) string {
	return strings.ToLower(strings.TrimSpace(c))
}

// Registrar exige la autorización previa y expresa de la Ley 1581 de 2012: sin ella
// no se crea la cuenta. Se valida antes de bcrypt para no gastar CPU en una petición que se rechaza.
func (s *AuthService) Registrar(ctx context.Context, in model.RegistroInput) (*Sesion, error) {
	if !in.AceptaTerminos {
		return nil, ErrTerminosNoAceptados
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Contrasena), costoBcrypt)
	if err != nil {
		return nil, err
	}
	u := &model.Usuario{
		Correo:          normalizarCorreo(in.Correo),
		Nombre:          strings.TrimSpace(in.Nombre),
		HashContrasena:  string(hash),
		Rol:             in.Rol,
		VersionTerminos: VersionTerminos,
	}
	if err := s.usuarios.Crear(ctx, u, strings.TrimSpace(in.RazonSocial)); err != nil {
		if errors.Is(err, repository.ErrCorreoDuplicado) {
			return nil, ErrCorreoEnUso
		}
		return nil, err
	}
	return s.emitirSesion(ctx, u)
}

func (s *AuthService) Login(ctx context.Context, in model.LoginInput) (*Sesion, error) {
	u, err := s.usuarios.BuscarPorCorreo(ctx, normalizarCorreo(in.Correo))
	if errors.Is(err, repository.ErrNoEncontrado) {
		bcrypt.CompareHashAndPassword(s.hashFalso, []byte(in.Contrasena))
		return nil, ErrCredenciales
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.HashContrasena), []byte(in.Contrasena)) != nil {
		return nil, ErrCredenciales
	}
	s.eventos.InicioSesion(ctx, u.ID)
	return s.emitirSesion(ctx, u)
}

// Refrescar consume el refresh token (queda revocado) y emite una sesión nueva.
func (s *AuthService) Refrescar(ctx context.Context, refresh string) (*Sesion, error) {
	if refresh == "" {
		return nil, ErrSesionInvalida
	}
	usuarioID, err := s.tokens.Consumir(ctx, hashToken(refresh))
	if errors.Is(err, repository.ErrNoEncontrado) {
		return nil, ErrSesionInvalida
	}
	if err != nil {
		return nil, err
	}
	u, err := s.usuarios.BuscarPorID(ctx, usuarioID)
	if err != nil {
		return nil, ErrSesionInvalida
	}
	// Renovar el token también es volver a la plataforma: cuenta para las métricas de actividad.
	s.eventos.InicioSesion(ctx, u.ID)
	return s.emitirSesion(ctx, u)
}

// Logout revoca el refresh token. Si ya no era válido no importa: el resultado es el mismo.
func (s *AuthService) Logout(ctx context.Context, refresh string) {
	if refresh != "" {
		s.tokens.Consumir(ctx, hashToken(refresh))
	}
}

func (s *AuthService) Yo(ctx context.Context, usuarioID string) (model.UsuarioPublico, error) {
	u, err := s.usuarios.BuscarPorID(ctx, usuarioID)
	if err != nil {
		return model.UsuarioPublico{}, err
	}
	return u.Publico(), nil
}

// ValidarToken verifica firma, algoritmo, emisor y vencimiento del token de acceso.
func (s *AuthService) ValidarToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (any, error) { return s.secreto, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(emisor),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, ErrSesionInvalida
	}
	return claims, nil
}

func (s *AuthService) emitirSesion(ctx context.Context, u *model.Usuario) (*Sesion, error) {
	ahora := time.Now()
	claims := Claims{
		Rol: u.Rol,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			Issuer:    emisor,
			IssuedAt:  jwt.NewNumericDate(ahora),
			ExpiresAt: jwt.NewNumericDate(ahora.Add(DuracionAcceso)),
		},
	}
	acceso, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secreto)
	if err != nil {
		return nil, err
	}

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	refresh := base64.RawURLEncoding.EncodeToString(bytes)
	if err := s.tokens.Guardar(ctx, u.ID, hashToken(refresh), ahora.Add(DuracionRefresh)); err != nil {
		return nil, err
	}

	return &Sesion{Usuario: u.Publico(), TokenAcceso: acceso, RefreshToken: refresh}, nil
}

func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}
