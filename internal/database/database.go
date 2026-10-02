package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migraciones embed.FS

// Conectar abre un pool de conexiones y verifica que la base responda.
func Conectar(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL inválida: %w", err)
	}
	cfg.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear el pool: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctxPing); err != nil {
		pool.Close()
		return nil, fmt.Errorf("la base no responde: %w", err)
	}
	return pool, nil
}

// Migrar aplica en orden los archivos .sql que aún no se hayan ejecutado.
// Cada migración corre en una transacción: o se aplica completa o no se aplica.
func Migrar(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version     TEXT PRIMARY KEY,
		aplicada_en TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)
	if err != nil {
		return fmt.Errorf("no se pudo crear schema_migrations: %w", err)
	}

	archivos, err := fs.ReadDir(migraciones, "migrations")
	if err != nil {
		return err
	}
	nombres := make([]string, 0, len(archivos))
	for _, a := range archivos {
		nombres = append(nombres, a.Name())
	}
	sort.Strings(nombres)

	for _, nombre := range nombres {
		var aplicada bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, nombre,
		).Scan(&aplicada)
		if err != nil {
			return err
		}
		if aplicada {
			continue
		}

		sql, err := migraciones.ReadFile("migrations/" + nombre)
		if err != nil {
			return err
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("falló la migración %s: %w", nombre, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, nombre); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		log.Printf("migración aplicada: %s", nombre)
	}
	return nil
}
