package repository

import (
	"context"
	"errors"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SuscripcionRepo struct {
	db *pgxpool.Pool
}

func NuevoSuscripcionRepo(db *pgxpool.Pool) *SuscripcionRepo {
	return &SuscripcionRepo{db: db}
}

// Activa devuelve la suscripción vigente de la empresa. Se consulta en cada acción
// de pago: si venció, el acceso se corta en ese mismo momento.
func (r *SuscripcionRepo) Activa(ctx context.Context, empresaID string) (*model.Suscripcion, error) {
	var s model.Suscripcion
	err := r.db.QueryRow(ctx,
		`SELECT id::text, periodo, inicia_en, termina_en, estado
		   FROM suscripciones
		  WHERE empresa_id = $1 AND estado = 'activa' AND termina_en > now()
		  ORDER BY termina_en DESC
		  LIMIT 1`, empresaID,
	).Scan(&s.ID, &s.Periodo, &s.IniciaEn, &s.TerminaEn, &s.Estado)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Crear registra una suscripción con pago simulado: un mes o un año desde hoy.
func (r *SuscripcionRepo) Crear(ctx context.Context, empresaID, periodo string) (*model.Suscripcion, error) {
	var s model.Suscripcion
	err := r.db.QueryRow(ctx,
		`INSERT INTO suscripciones (empresa_id, periodo, termina_en)
		 VALUES ($1, $2, now() + CASE WHEN $2 = 'anual' THEN interval '1 year' ELSE interval '1 month' END)
		 RETURNING id::text, periodo, inicia_en, termina_en, estado`,
		empresaID, periodo,
	).Scan(&s.ID, &s.Periodo, &s.IniciaEn, &s.TerminaEn, &s.Estado)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
