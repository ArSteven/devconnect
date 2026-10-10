package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/ArSteven/devconnect/internal/config"
	"github.com/ArSteven/devconnect/internal/database"
	"github.com/ArSteven/devconnect/internal/handler"
	"github.com/ArSteven/devconnect/internal/middleware"
	"github.com/ArSteven/devconnect/internal/repository"
	"github.com/ArSteven/devconnect/internal/service"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Cargar()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := database.Conectar(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := database.Migrar(ctx, pool); err != nil {
		log.Fatal(err)
	}

	// Capas: repository -> service -> handler
	eventos := service.NuevoEventos(repository.NuevoEventoRepo(pool))
	usuarios := repository.NuevoUsuarioRepo(pool)
	tokens := repository.NuevoTokenRepo(pool)
	authService, err := service.NuevoAuthService(usuarios, tokens, eventos, cfg.JWTSecret)
	if err != nil {
		log.Fatal(err)
	}
	authHandler := handler.NuevoAuthHandler(authService)
	requiereAuth := middleware.RequiereAuth(authService)

	suscripciones := repository.NuevoSuscripcionRepo(pool)

	publicaciones := repository.NuevoPublicacionRepo(pool)
	pubService := service.NuevoPublicacionService(publicaciones, suscripciones)
	pubHandler := handler.NuevoPublicacionHandler(pubService)

	estudiantes := repository.NuevoEstudianteRepo(pool)
	estService := service.NuevoEstudianteService(estudiantes, suscripciones, eventos)
	estHandler := handler.NuevoEstudianteHandler(estService)

	sesiones := repository.NuevoSesionRepo(pool)
	sesHandler := handler.NuevoSesionHandler(service.NuevoSesionService(sesiones))

	metHandler := handler.NuevoMetricasHandler(service.NuevoMetricasService(repository.NuevoMetricaRepo(pool)))
	ejeHandler := handler.NuevoEjecucionHandler(service.NuevoEjecucionService(service.URLGoPlayground))

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.CabecerasSeguridad())
	r.Use(middleware.LimiteCuerpo(1 << 20)) // 1 MB
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
	}

	r.GET("/health", func(c *gin.Context) {
		ctxPing, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		// La última migración aplicada confirma qué versión de la base quedó desplegada.
		var migracion string
		if err := pool.QueryRow(ctxPing, `SELECT COALESCE(max(version), '') FROM schema_migrations`).Scan(&migracion); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"estado": "error", "bd": "sin conexión"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"estado": "ok", "bd": "ok", "migracion": migracion})
	})

	v1 := r.Group("/api/v1")
	authHandler.Rutas(v1.Group("/auth"), requiereAuth)

	// Todo lo de aquí abajo exige sesión iniciada
	privado := v1.Group("", requiereAuth)
	pubHandler.Rutas(privado)
	estHandler.Rutas(privado)
	sesHandler.Rutas(privado)
	metHandler.Rutas(privado)
	ejeHandler.Rutas(privado)

	log.Printf("API escuchando en :%s (%s)", cfg.Puerto, cfg.Entorno)
	if err := r.Run(":" + cfg.Puerto); err != nil {
		log.Fatal(err)
	}
}
