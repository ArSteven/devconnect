package repository

import (
	"context"
	"errors"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EstudianteRepo struct {
	db *pgxpool.Pool
}

func NuevoEstudianteRepo(db *pgxpool.Pool) *EstudianteRepo {
	return &EstudianteRepo{db: db}
}

// Perfil devuelve el perfil público y, aparte, el correo (que solo se muestra con permiso).
func (r *EstudianteRepo) Perfil(ctx context.Context, id string) (*model.PerfilEstudiante, string, error) {
	var p model.PerfilEstudiante
	var correo string
	err := r.db.QueryRow(ctx,
		`SELECT u.id::text, u.nombre, u.correo,
		        COALESCE(pe.programa, ''), COALESCE(pe.institucion, ''), COALESCE(pe.ciudad, ''),
		        pe.stack, COALESCE(pe.biografia, '')
		   FROM usuarios u
		   JOIN perfiles_estudiante pe ON pe.usuario_id = u.id
		  WHERE u.id = $1`, id,
	).Scan(&p.ID, &p.Nombre, &correo, &p.Programa, &p.Institucion, &p.Ciudad, &p.Stack, &p.Biografia)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNoEncontrado
	}
	if err != nil {
		return nil, "", err
	}
	return &p, correo, nil
}

func (r *EstudianteRepo) Totales(ctx context.Context, id string) (model.TotalesPortafolio, error) {
	var t model.TotalesPortafolio
	err := r.db.QueryRow(ctx,
		`SELECT
		   (SELECT count(*) FROM publicaciones WHERE autor_id = $1),
		   (SELECT count(*) FROM propuestas_mejora WHERE autor_id = $1),
		   (SELECT count(*) FROM propuestas_mejora WHERE autor_id = $1 AND estado = 'aceptada'),
		   (SELECT count(*) FROM propuestas_mejora pm JOIN publicaciones p ON p.id = pm.publicacion_id
		     WHERE p.autor_id = $1 AND pm.estado = 'aceptada'),
		   (SELECT count(DISTINCT x) FROM (
		      SELECT p.autor_id AS x FROM propuestas_mejora pm JOIN publicaciones p ON p.id = pm.publicacion_id
		       WHERE pm.autor_id = $1 AND pm.estado = 'aceptada'
		      UNION
		      SELECT pm.autor_id FROM propuestas_mejora pm JOIN publicaciones p ON p.id = pm.publicacion_id
		       WHERE p.autor_id = $1 AND pm.estado = 'aceptada') c),
		   (SELECT count(*) FROM sesiones_vivo WHERE anfitrion_id = $1 AND estado <> 'programada')`, id,
	).Scan(&t.Publicaciones, &t.PropuestasHechas, &t.MejorasAportadas, &t.MejorasRecibidas, &t.Colaboradores, &t.Sesiones)
	return t, err
}

// Habilidades suma, por lenguaje, lo que publicó y las mejoras que le aceptaron.
func (r *EstudianteRepo) Habilidades(ctx context.Context, id string) ([]model.Habilidad, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lenguaje, sum(pubs)::int, sum(aportes)::int FROM (
		  SELECT p.lenguaje, 1 AS pubs, 0 AS aportes FROM publicaciones p WHERE p.autor_id = $1
		  UNION ALL
		  SELECT p.lenguaje, 0, 1 FROM propuestas_mejora pm JOIN publicaciones p ON p.id = pm.publicacion_id
		   WHERE pm.autor_id = $1 AND pm.estado = 'aceptada'
		) t
		GROUP BY lenguaje
		ORDER BY sum(aportes) DESC, sum(pubs) DESC
		LIMIT 8`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []model.Habilidad{}
	for rows.Next() {
		var h model.Habilidad
		if err := rows.Scan(&h.Lenguaje, &h.Publicaciones, &h.Aportes); err != nil {
			return nil, err
		}
		lista = append(lista, h)
	}
	return lista, rows.Err()
}

// Destacados: sus mejoras aceptadas más recientes, con el problema que resolvieron.
func (r *EstudianteRepo) Destacados(ctx context.Context, id string) ([]model.Destacado, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pm.id::text, p.id::text, p.titulo, p.lenguaje, ua.nombre, pm.explicacion, pm.actualizado_en
		  FROM propuestas_mejora pm
		  JOIN publicaciones p ON p.id = pm.publicacion_id
		  JOIN usuarios ua ON ua.id = p.autor_id
		 WHERE pm.autor_id = $1 AND pm.estado = 'aceptada'
		 ORDER BY pm.actualizado_en DESC
		 LIMIT 4`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []model.Destacado{}
	for rows.Next() {
		var d model.Destacado
		if err := rows.Scan(&d.PropuestaID, &d.PublicacionID, &d.Titulo, &d.Lenguaje, &d.AutorOriginal, &d.Explicacion, &d.Fecha); err != nil {
			return nil, err
		}
		lista = append(lista, d)
	}
	return lista, rows.Err()
}

// Actividad cuenta contribuciones por día en los últimos 6 meses (para el mapa de calor).
func (r *EstudianteRepo) Actividad(ctx context.Context, id string) ([]model.DiaActividad, error) {
	rows, err := r.db.Query(ctx, `
		SELECT to_char(dia, 'YYYY-MM-DD'), count(*)::int FROM (
		  SELECT creado_en::date AS dia FROM publicaciones WHERE autor_id = $1
		  UNION ALL SELECT creado_en::date FROM propuestas_mejora WHERE autor_id = $1
		  UNION ALL SELECT creado_en::date FROM comentarios WHERE autor_id = $1
		  UNION ALL SELECT inicia_en::date FROM sesiones_vivo WHERE anfitrion_id = $1 AND estado <> 'programada'
		) t
		WHERE dia > current_date - 182
		GROUP BY dia
		ORDER BY dia`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []model.DiaActividad{}
	for rows.Next() {
		var d model.DiaActividad
		if err := rows.Scan(&d.Fecha, &d.Total); err != nil {
			return nil, err
		}
		lista = append(lista, d)
	}
	return lista, rows.Err()
}

// Historial arma la línea de tiempo del portafolio a partir de la actividad real.
// No hay tabla de portafolio: así nunca se desincroniza.
func (r *EstudianteRepo) Historial(ctx context.Context, id string) ([]model.EventoHistorial, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 'publicacion', p.id::text, p.id::text, p.titulo, p.lenguaje, '', p.creado_en
		  FROM publicaciones p WHERE p.autor_id = $1
		UNION ALL
		SELECT 'aporte', pm.id::text, p.id::text, p.titulo, p.lenguaje, ua.nombre, pm.actualizado_en
		  FROM propuestas_mejora pm
		  JOIN publicaciones p ON p.id = pm.publicacion_id
		  JOIN usuarios ua ON ua.id = p.autor_id
		 WHERE pm.autor_id = $1 AND pm.estado = 'aceptada'
		UNION ALL
		SELECT 'recibida', pm.id::text, p.id::text, p.titulo, p.lenguaje, up.nombre, pm.actualizado_en
		  FROM propuestas_mejora pm
		  JOIN publicaciones p ON p.id = pm.publicacion_id
		  JOIN usuarios up ON up.id = pm.autor_id
		 WHERE p.autor_id = $1 AND pm.estado = 'aceptada'
		UNION ALL
		SELECT 'sesion', s.id::text, '', s.titulo, '', '', s.inicia_en
		  FROM sesiones_vivo s WHERE s.anfitrion_id = $1
		ORDER BY 7 DESC
		LIMIT 50`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []model.EventoHistorial{}
	for rows.Next() {
		var e model.EventoHistorial
		if err := rows.Scan(&e.Tipo, &e.ID, &e.PublicacionID, &e.Titulo, &e.Lenguaje, &e.ConQuien, &e.Fecha); err != nil {
			return nil, err
		}
		lista = append(lista, e)
	}
	return lista, rows.Err()
}

func (r *EstudianteRepo) ActualizarPerfil(ctx context.Context, id string, in model.ActualizarPerfilInput) error {
	_, err := r.db.Exec(ctx,
		`UPDATE perfiles_estudiante
		    SET programa = NULLIF($2, ''), institucion = NULLIF($3, ''), ciudad = NULLIF($4, ''),
		        stack = $5, biografia = NULLIF($6, ''), actualizado_en = now()
		  WHERE usuario_id = $1`,
		id, in.Programa, in.Institucion, in.Ciudad, in.Stack, in.Biografia)
	return err
}

// BuscarTalento filtra estudiantes por lenguaje (en su stack o en lo que han publicado),
// ciudad y si tienen mejoras aceptadas. Ordena primero a quienes más han aportado.
func (r *EstudianteRepo) BuscarTalento(ctx context.Context, f model.FiltroTalento) ([]model.TarjetaTalento, error) {
	if f.Lenguajes == nil {
		f.Lenguajes = []string{}
	}
	rows, err := r.db.Query(ctx, `
		SELECT u.id::text, u.nombre, COALESCE(pe.ciudad, ''), COALESCE(pe.institucion, ''), pe.stack,
		       (SELECT count(*) FROM publicaciones p WHERE p.autor_id = u.id) AS pubs,
		       (SELECT count(*) FROM propuestas_mejora pm WHERE pm.autor_id = u.id AND pm.estado = 'aceptada') AS mejoras
		  FROM usuarios u
		  JOIN perfiles_estudiante pe ON pe.usuario_id = u.id
		 WHERE u.rol = 'estudiante'
		   AND (cardinality($1::text[]) = 0
		        OR pe.stack && $1::text[]
		        OR EXISTS (SELECT 1 FROM publicaciones p WHERE p.autor_id = u.id AND p.lenguaje = ANY($1::text[]))
		        OR EXISTS (SELECT 1 FROM propuestas_mejora pm JOIN publicaciones p ON p.id = pm.publicacion_id
		                    WHERE pm.autor_id = u.id AND pm.estado = 'aceptada' AND p.lenguaje = ANY($1::text[])))
		   AND ($2::text = '' OR lower(pe.ciudad) = lower($2::text))
		   AND (NOT $3::boolean OR EXISTS (SELECT 1 FROM propuestas_mejora pm WHERE pm.autor_id = u.id AND pm.estado = 'aceptada'))
		 ORDER BY mejoras DESC, pubs DESC, u.nombre
		 LIMIT 50`,
		f.Lenguajes, f.Ciudad, f.ConMejoras)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []model.TarjetaTalento{}
	for rows.Next() {
		var t model.TarjetaTalento
		if err := rows.Scan(&t.ID, &t.Nombre, &t.Ciudad, &t.Institucion, &t.Stack, &t.Publicaciones, &t.MejorasAportadas); err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}
	return lista, rows.Err()
}
