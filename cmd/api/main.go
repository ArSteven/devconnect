package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/ArSteven/devconnect/internal/config"
	"github.com/ArSteven/devconnect/internal/database"
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

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		ctxPing, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctxPing); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"estado": "error", "bd": "sin conexión"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"estado": "ok", "bd": "ok"})
	})

	log.Printf("API escuchando en :%s (%s)", cfg.Puerto, cfg.Entorno)
	if err := r.Run(":" + cfg.Puerto); err != nil {
		log.Fatal(err)
	}
}
