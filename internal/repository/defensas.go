package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrDefensaVigente: la solución ya tiene una defensa invitada o realizada.
var ErrDefensaVigente = errors.New("la solución ya tiene una defensa vigente")

// Verificar deja constancia de que otra persona ejecutó la mejora y cambió el resultado. Si ya
// estaba verificada se conserva la primera vez. No toca actualizado_en: esa es la fecha de aceptación.
func (r *PublicacionRepo) Verificar(ctx context.Context, propuestaID, usuarioID string) (time.Time, error) {
	var en time.Time
	err := r.db.QueryRow(ctx,
		`UPDATE propuestas_mejora
		    SET verificada_en = COALESCE(verificada_en, now()),
		        verificada_por = CASE WHEN verificada_en IS NULL THEN $2::uuid ELSE verificada_por END
		  WHERE id = $1
		  RETURNING verificada_en`, propuestaID, usuarioID,
	).Scan(&en)
	if errors.Is(err, pgx.ErrNoRows) {
		return en, ErrNoEncontrado
	}
	return en, err
}

const selectDefensa = `
SELECT d.id::text, d.propuesta_id::text, p.id::text, p.titulo,
       d.empresa_id::text, COALESCE(e.razon_social, ue.nombre),
       d.estudiante_id::text, us.nombre, COALESCE(pe.github_url, ''),
       d.inicia_en, d.sala_jitsi, d.estado, d.aprobada
  FROM defensas d
  JOIN propuestas_mejora pm ON pm.id = d.propuesta_id
  JOIN publicaciones p ON p.id = pm.publicacion_id
  JOIN usuarios ue ON ue.id = d.empresa_id
  LEFT JOIN empresas e ON e.usuario_id = d.empresa_id
  JOIN usuarios us ON us.id = d.estudiante_id
  LEFT JOIN perfiles_estudiante pe ON pe.usuario_id = d.estudiante_id `

func escanearDefensa(row pgx.Row) (model.Defensa, error) {
	d := model.Defensa{DuracionMin: model.DuracionDefensaMin}
	err := row.Scan(&d.ID, &d.PropuestaID, &d.PublicacionID, &d.Titulo, &d.EmpresaID, &d.EmpresaNombre,
		&d.EstudianteID, &d.EstudianteNombre, &d.EstudianteGithub, &d.IniciaEn, &d.Sala, &d.Estado, &d.Aprobada)
	return d, err
}

func (r *PublicacionRepo) listarDefensas(ctx context.Context, sql string, args ...any) ([]model.Defensa, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []model.Defensa{}
	for rows.Next() {
		d, err := escanearDefensa(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, d)
	}
	return lista, rows.Err()
}

func (r *PublicacionRepo) CrearDefensa(ctx context.Context, d *model.Defensa) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO defensas (propuesta_id, empresa_id, estudiante_id, inicia_en, sala_jitsi)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id::text`,
		d.PropuestaID, d.EmpresaID, d.EstudianteID, d.IniciaEn, d.Sala,
	).Scan(&d.ID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_defensas_vigente" {
		return ErrDefensaVigente
	}
	return err
}

func (r *PublicacionRepo) ObtenerDefensa(ctx context.Context, id string) (*model.Defensa, error) {
	d, err := escanearDefensa(r.db.QueryRow(ctx, selectDefensa+`WHERE d.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// CambiarDefensa pasa la defensa de `desde` a `estado`. Si mientras tanto otra petición la cambió,
// no toca nada y devuelve ErrNoEncontrado.
func (r *PublicacionRepo) CambiarDefensa(ctx context.Context, id, desde, estado string, aprobada bool) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE defensas SET estado = $3, aprobada = $4, actualizado_en = now()
		  WHERE id = $1 AND estado = $2`, id, desde, estado, aprobada)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoEncontrado
	}
	return nil
}

// DefensasDe: las defensas de una empresa o de un estudiante, desde hace una semana en adelante.
func (r *PublicacionRepo) DefensasDe(ctx context.Context, usuarioID string) ([]model.Defensa, error) {
	return r.listarDefensas(ctx, selectDefensa+`
		WHERE (d.empresa_id = $1 OR d.estudiante_id = $1) AND d.inicia_en > now() - interval '7 days'
		ORDER BY d.inicia_en`, usuarioID)
}

// DefensasVigentes: las defensas invitadas o realizadas de las soluciones de un reto.
func (r *PublicacionRepo) DefensasVigentes(ctx context.Context, publicacionID string) ([]model.Defensa, error) {
	return r.listarDefensas(ctx, selectDefensa+`WHERE p.id = $1 AND d.estado <> 'cancelada'`, publicacionID)
}
