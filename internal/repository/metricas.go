package repository

import (
	"context"
	"time"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricaRepo struct {
	db *pgxpool.Pool
}

func NuevoMetricaRepo(db *pgxpool.Pool) *MetricaRepo {
	return &MetricaRepo{db: db}
}

// Calcular cuenta lo ocurrido entre desde (incluido) y hasta (excluido). Las tasas las calcula el service.
func (r *MetricaRepo) Calcular(ctx context.Context, desde, hasta time.Time) (*model.Metricas, error) {
	m := &model.Metricas{}
	m.Usuarios.RegistradosPorRol = map[string]int{"estudiante": 0, "empresa": 0, "admin": 0}

	rows, err := r.db.Query(ctx,
		`SELECT rol, count(*)::int FROM usuarios WHERE creado_en >= $1 AND creado_en < $2 GROUP BY rol`, desde, hasta)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var rol string
		var total int
		if err := rows.Scan(&rol, &total); err != nil {
			rows.Close()
			return nil, err
		}
		m.Usuarios.RegistradosPorRol[rol] = total
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Activos: cualquier evento en los 7 y 30 días que terminan en "hasta".
	err = r.db.QueryRow(ctx, `
		SELECT count(DISTINCT usuario_id) FILTER (WHERE creado_en >= $1::timestamptz - interval '7 days')::int,
		       count(DISTINCT usuario_id)::int
		  FROM eventos
		 WHERE creado_en >= $1::timestamptz - interval '30 days' AND creado_en < $1::timestamptz`, hasta,
	).Scan(&m.Usuarios.Activos7Dias, &m.Usuarios.Activos30Dias)
	if err != nil {
		return nil, err
	}

	// Retención: de la cohorte (registrados en el rango), quienes tuvieron actividad justo el día 1 y
	// el día 7 después de registrarse. Solo son elegibles quienes ya llegaron a ese día.
	err = r.db.QueryRow(ctx, `
		WITH cohorte AS (
		  SELECT u.id, (u.creado_en AT TIME ZONE `+zonaMetricas+`)::date AS dia
		    FROM usuarios u
		   WHERE u.rol <> 'admin' AND u.creado_en >= $1 AND u.creado_en < $2
		), regreso AS (
		  SELECT c.dia,
		         EXISTS (SELECT 1 FROM eventos e WHERE e.usuario_id = c.id
		                    AND (e.creado_en AT TIME ZONE `+zonaMetricas+`)::date = c.dia + 1) AS dia1,
		         EXISTS (SELECT 1 FROM eventos e WHERE e.usuario_id = c.id
		                    AND (e.creado_en AT TIME ZONE `+zonaMetricas+`)::date = c.dia + 7) AS dia7
		    FROM cohorte c
		)
		SELECT count(*)::int,
		       count(*) FILTER (WHERE dia + 1 <= hoy)::int, count(*) FILTER (WHERE dia + 1 <= hoy AND dia1)::int,
		       count(*) FILTER (WHERE dia + 7 <= hoy)::int, count(*) FILTER (WHERE dia + 7 <= hoy AND dia7)::int
		  FROM regreso, (SELECT (now() AT TIME ZONE `+zonaMetricas+`)::date AS hoy) h`, desde, hasta,
	).Scan(&m.Retencion.Cohorte,
		&m.Retencion.Dia1.Elegibles, &m.Retencion.Dia1.Volvieron,
		&m.Retencion.Dia7.Elegibles, &m.Retencion.Dia7.Volvieron)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM publicaciones WHERE creado_en >= $1 AND creado_en < $2)::int,
		       (SELECT count(*) FROM propuestas_mejora WHERE creado_en >= $1 AND creado_en < $2)::int,
		       (SELECT count(*) FROM propuestas_mejora
		         WHERE estado = 'aceptada' AND actualizado_en >= $1 AND actualizado_en < $2)::int,
		       (SELECT count(DISTINCT autor_id) FROM propuestas_mejora WHERE creado_en >= $1 AND creado_en < $2)::int`,
		desde, hasta,
	).Scan(&m.Colaboracion.Publicaciones, &m.Colaboracion.Propuestas,
		&m.Colaboracion.PropuestasAceptadas, &m.Colaboracion.ColaboradoresDistintos)
	if err != nil {
		return nil, err
	}

	// Conversión: de las empresas registradas en el rango, las que han tenido un plan alguna vez.
	err = r.db.QueryRow(ctx, `
		SELECT count(*)::int,
		       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM suscripciones s WHERE s.empresa_id = u.id))::int,
		       (SELECT count(DISTINCT empresa_id) FROM suscripciones WHERE estado = 'activa' AND termina_en > now())::int
		  FROM usuarios u
		 WHERE u.rol = 'empresa' AND u.creado_en >= $1 AND u.creado_en < $2`, desde, hasta,
	).Scan(&m.Empresas.Registradas, &m.Empresas.Suscritas, &m.Empresas.ConPlanActivo)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT count(*)::int, count(DISTINCT (usuario_id, objetivo_id))::int
		  FROM eventos
		 WHERE tipo = 'contacto_visto' AND creado_en >= $1 AND creado_en < $2`, desde, hasta,
	).Scan(&m.ContactosVistos.Total, &m.ContactosVistos.Unicos)
	if err != nil {
		return nil, err
	}
	return m, nil
}
