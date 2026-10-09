package model

import "time"

// Usuario es la fila de la tabla usuarios.
type Usuario struct {
	ID              string
	Correo          string
	Nombre          string
	HashContrasena  string
	Rol             string
	CreadoEn        time.Time
	GithubURL       string // solo estudiantes: de aquí sale la foto del avatar
	RazonSocial     string // solo empresas
	VersionTerminos string // versión de los términos y la política que aceptó al registrarse
}

// UsuarioPublico es lo único que la API devuelve sobre un usuario autenticado.
// Nunca incluye el hash de la contraseña.
type UsuarioPublico struct {
	ID          string `json:"id"`
	Nombre      string `json:"nombre"`
	Correo      string `json:"correo"`
	Rol         string `json:"rol"`
	GithubURL   string `json:"github_url"`
	RazonSocial string `json:"razon_social,omitempty"`
}

func (u Usuario) Publico() UsuarioPublico {
	return UsuarioPublico{ID: u.ID, Nombre: u.Nombre, Correo: u.Correo, Rol: u.Rol, GithubURL: u.GithubURL, RazonSocial: u.RazonSocial}
}

// RegistroInput valida el cuerpo de POST /auth/registro.
// max=72 en la contraseña: bcrypt ignora lo que pase de 72 bytes.
type RegistroInput struct {
	Nombre      string `json:"nombre" binding:"required,min=2,max=100"`
	Correo      string `json:"correo" binding:"required,email,max=254"`
	Contrasena  string `json:"contrasena" binding:"required,min=8,max=72"`
	Rol         string `json:"rol" binding:"required,oneof=estudiante empresa"`
	RazonSocial string `json:"razon_social" binding:"required_if=Rol empresa,max=200"`
	// Sin binding: "required" rechazaría false con un error genérico. El service lo valida.
	AceptaTerminos bool `json:"acepta_terminos"`
}

// LoginInput valida el cuerpo de POST /auth/login.
type LoginInput struct {
	Correo     string `json:"correo" binding:"required,email,max=254"`
	Contrasena string `json:"contrasena" binding:"required,max=72"`
}
