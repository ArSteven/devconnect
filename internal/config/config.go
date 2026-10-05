package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Puerto      string
	DatabaseURL string
	JWTSecret   string
	Entorno     string // "desarrollo" o "produccion"
}

// Cargar lee la configuración. En local toma los valores de .env;
// en Railway vienen de las variables del servicio.
func Cargar() (Config, error) {
	_ = godotenv.Load() // si no hay .env (Railway), no pasa nada

	cfg := Config{
		Puerto:      os.Getenv("PORT"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Entorno:     os.Getenv("ENV"),
	}
	if cfg.Puerto == "" {
		cfg.Puerto = "8080"
	}
	if cfg.Entorno == "" {
		cfg.Entorno = "desarrollo"
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("falta la variable DATABASE_URL")
	}
	if len(cfg.JWTSecret) < 32 {
		return cfg, errors.New("JWT_SECRET falta o tiene menos de 32 caracteres")
	}
	return cfg, nil
}
