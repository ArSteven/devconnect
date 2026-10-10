package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// zonaMetricas: los días de las métricas son los de Colombia. Texto fijo del código.
const zonaMetricas = `'America/Bogota'`

// inicioDeHoy es la medianoche de hoy en Colombia, como instante.
const inicioDeHoy = `(date_trunc('day', now() AT TIME ZONE ` + zonaMetricas + `) AT TIME ZONE ` + zonaMetricas + `)`

type EventoRepo struct {
	db *pgxpool.Pool
}

func NuevoEventoRepo(db *pgxpool.Pool) *EventoRepo {
	return &EventoRepo{db: db}
}

// Registrar guarda un evento. objetivoID vacío = sin objetivo.
func (r *EventoRepo) Registrar(ctx context.Context, usuarioID, tipo, objetivoID string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO eventos (usuario_id, tipo, objetivo_id) VALUES ($1, $2, NULLIF($3, '')::uuid)`,
		usuarioID, tipo, objetivoID)
	return err
}

// RegistrarUnoPorDia guarda un evento por cada objetivo (o uno sin objetivo si la lista está
// vacía), salvo los que ese usuario ya tenga hoy: así un inicio de sesión o un contacto visto
// cuenta como mucho una vez al día.
func (r *EventoRepo) RegistrarUnoPorDia(ctx context.Context, usuarioID, tipo string, objetivos []string) error {
	if objetivos == nil {
		objetivos = []string{}
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO eventos (usuario_id, tipo, objetivo_id)
		SELECT $1::uuid, $2::text, o.id
		  FROM unnest(CASE WHEN cardinality($3::text[]) = 0 THEN ARRAY[NULL]::uuid[] ELSE $3::text[]::uuid[] END) AS o(id)
		 WHERE NOT EXISTS (
		   SELECT 1 FROM eventos e
		    WHERE e.tipo = $2::text AND e.usuario_id = $1::uuid AND e.objetivo_id IS NOT DISTINCT FROM o.id
		      AND e.creado_en >= `+inicioDeHoy+`)`,
		usuarioID, tipo, objetivos)
	return err
}
