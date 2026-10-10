package repository

import (
	"context"
	"errors"
	"time"

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

// Perfil devuelve el perfil completo, la fecha de nacimiento y, aparte, el correo
// (que solo se muestra con permiso).
func (r *EstudianteRepo) Perfil(ctx context.Context, id string) (*model.PerfilEstudiante, *time.Time, string, error) {
	var p model.PerfilEstudiante
	var nacimiento *time.Time
	var correo string
	var semestre, anioInicio, anioFin *int16
	err := r.db.QueryRow(ctx,
		`SELECT u.id::text, u.nombre, u.correo,
		        COALESCE(pe.titular, ''), pe.fecha_nacimiento,
		        COALESCE(pe.programa, ''), COALESCE(pe.institucion, ''), pe.semestre,
		        COALESCE(pe.estado_academico, ''), pe.anio_inicio, pe.anio_fin,
		        COALESCE(pe.ciudad, ''), COALESCE(pe.disponibilidad, ''), COALESCE(pe.modalidad, ''),
		        COALESCE(pe.github_url, ''), COALESCE(pe.linkedin_url, ''), COALESCE(pe.sitio_url, ''),
		        pe.stack, pe.idiomas, pe.experiencia, COALESCE(pe.biografia, ''), pe.contacto_visible
		   FROM usuarios u
		   JOIN perfiles_estudiante pe ON pe.usuario_id = u.id
		  WHERE u.id = $1`, id,
	).Scan(&p.ID, &p.Nombre, &correo,
		&p.Titular, &nacimiento,
		&p.Programa, &p.Institucion, &semestre,
		&p.EstadoAcademico, &anioInicio, &anioFin,
		&p.Ciudad, &p.Disponibilidad, &p.Modalidad,
		&p.GithubURL, &p.LinkedinURL, &p.SitioURL,
		&p.Stack, &p.Idiomas, &p.Experiencia, &p.Biografia, &p.ContactoVisible)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, "", ErrNoEncontrado
	}
	if err != nil {
		return nil, nil, "", err
	}
	p.Semestre, p.AnioInicio, p.AnioFin = aInt(semestre), aInt(anioInicio), aInt(anioFin)
	if p.Experiencia == nil {
		p.Experiencia = []model.Experiencia{}
	}
	return &p, nacimiento, correo, nil
}

func aInt(v *int16) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
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
		       WHERE pm.autor_id = $1 AND pm.estado = 'aceptada' AND p.tipo = 'pregunta'
		      UNION
		      SELECT pm.autor_id FROM propuestas_mejora pm JOIN publicaciones p ON p.id = pm.publicacion_id
		       WHERE p.autor_id = $1 AND pm.estado = 'aceptada') c),
		   (SELECT count(DISTINCT p.autor_id) FROM propuestas_mejora pm JOIN publicaciones p ON p.id = pm.publicacion_id
		     WHERE pm.autor_id = $1 AND pm.estado = 'aceptada' AND p.tipo = 'pregunta'),
		   (SELECT count(*) FROM sesiones_vivo WHERE anfitrion_id = $1 AND iniciada_en IS NOT NULL),
		   (SELECT count(DISTINCT pm.publicacion_id) FROM propuestas_mejora pm JOIN publicaciones p ON p.id = pm.publicacion_id
		     WHERE pm.autor_id = $1 AND pm.estado = 'aceptada' AND p.tipo = 'reto')`, id,
	).Scan(&t.Publicaciones, &t.PropuestasHechas, &t.MejorasAportadas, &t.MejorasRecibidas, &t.Colaboradores, &t.PersonasAyudadas, &t.Sesiones, &t.RetosResueltos)
	return t, err
}

// Habilidades suma, por lenguaje, lo que publicó y las mejoras que le aceptaron.
// "otro" no cuenta: no dice nada de una aptitud concreta.
func (r *EstudianteRepo) Habilidades(ctx context.Context, id string) ([]model.Habilidad, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lenguaje, sum(pubs)::int, sum(aportes)::int FROM (
		  SELECT p.lenguaje, 1 AS pubs, 0 AS aportes FROM publicaciones p WHERE p.autor_id = $1
		  UNION ALL
		  SELECT p.lenguaje, 0, 1 FROM propuestas_mejora pm JOIN publicaciones p ON p.id = pm.publicacion_id
		   WHERE pm.autor_id = $1 AND pm.estado = 'aceptada'
		) t
		WHERE lenguaje <> 'otro'
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
// Los retos de empresas van primero: es lo que más pesa para un reclutador.
func (r *EstudianteRepo) Destacados(ctx context.Context, id string) ([]model.Destacado, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pm.id::text, p.id::text, p.titulo, p.lenguaje, COALESCE(e.razon_social, ua.nombre),
		       p.tipo = 'reto', pm.explicacion, pm.actualizado_en
		  FROM propuestas_mejora pm
		  JOIN publicaciones p ON p.id = pm.publicacion_id
		  JOIN usuarios ua ON ua.id = p.autor_id
		  LEFT JOIN empresas e ON e.usuario_id = p.autor_id
		 WHERE pm.autor_id = $1 AND pm.estado = 'aceptada'
		 ORDER BY (p.tipo = 'reto') DESC, pm.actualizado_en DESC
		 LIMIT 4`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []model.Destacado{}
	for rows.Next() {
		var d model.Destacado
		if err := rows.Scan(&d.PropuestaID, &d.PublicacionID, &d.Titulo, &d.Lenguaje, &d.AutorOriginal, &d.EsReto, &d.Explicacion, &d.Fecha); err != nil {
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
		  UNION ALL SELECT iniciada_en::date FROM sesiones_vivo WHERE anfitrion_id = $1 AND iniciada_en IS NOT NULL
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
		SELECT CASE WHEN p.tipo = 'reto' THEN 'reto' ELSE 'aporte' END, pm.id::text, p.id::text, p.titulo, p.lenguaje,
		       COALESCE(e.razon_social, ua.nombre), pm.actualizado_en
		  FROM propuestas_mejora pm
		  JOIN publicaciones p ON p.id = pm.publicacion_id
		  JOIN usuarios ua ON ua.id = p.autor_id
		  LEFT JOIN empresas e ON e.usuario_id = p.autor_id
		 WHERE pm.autor_id = $1 AND pm.estado = 'aceptada'
		UNION ALL
		SELECT 'recibida', pm.id::text, p.id::text, p.titulo, p.lenguaje, up.nombre, pm.actualizado_en
		  FROM propuestas_mejora pm
		  JOIN publicaciones p ON p.id = pm.publicacion_id
		  JOIN usuarios up ON up.id = pm.autor_id
		 WHERE p.autor_id = $1 AND pm.estado = 'aceptada'
		UNION ALL
		SELECT 'sesion', s.id::text, '', s.titulo, '', '', s.iniciada_en
		  FROM sesiones_vivo s WHERE s.anfitrion_id = $1 AND s.iniciada_en IS NOT NULL
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
		    SET titular = NULLIF($2, ''), fecha_nacimiento = NULLIF($3, '')::date,
		        programa = NULLIF($4, ''), institucion = NULLIF($5, ''), semestre = $6,
		        estado_academico = NULLIF($7, ''), anio_inicio = $8, anio_fin = $9,
		        ciudad = NULLIF($10, ''), disponibilidad = NULLIF($11, ''), modalidad = NULLIF($12, ''),
		        github_url = NULLIF($13, ''), linkedin_url = NULLIF($14, ''), sitio_url = NULLIF($15, ''),
		        stack = $16, idiomas = $17, experiencia = $18, biografia = NULLIF($19, ''),
		        contacto_visible = COALESCE($20::boolean, contacto_visible),
		        actualizado_en = now()
		  WHERE usuario_id = $1`,
		id, in.Titular, in.FechaNacimiento,
		in.Programa, in.Institucion, in.Semestre,
		in.EstadoAcademico, in.AnioInicio, in.AnioFin,
		in.Ciudad, in.Disponibilidad, in.Modalidad,
		in.GithubURL, in.LinkedinURL, in.SitioURL,
		in.Stack, in.Idiomas, in.Experiencia, in.Biografia,
		in.ContactoVisible)
	return err
}

// selectTarjeta arma la tarjeta de talento con su evidencia: lenguajes verificados,
// propuestas, mejoras aceptadas y semanas activas de las últimas 8 (la constancia).
// $1 es siempre el ID de la empresa que consulta, para marcar a quienes ya guardó.
// extra agrega columnas fijas escritas en el código, nunca datos del usuario.
func selectTarjeta(extra string) string {
	return `
SELECT u.id::text, u.nombre, COALESCE(pe.titular, ''), COALESCE(pe.github_url, ''),
       COALESCE(pe.ciudad, ''), COALESCE(pe.institucion, ''), COALESCE(pe.programa, ''),
       pe.semestre, COALESCE(pe.estado_academico, ''), COALESCE(pe.disponibilidad, ''), COALESCE(pe.modalidad, ''),
       pe.stack, m.verificadas, m.pubs, m.propuestas, m.mejoras, m.semanas,
       EXISTS (SELECT 1 FROM candidatos_guardados cg WHERE cg.empresa_id = $1::uuid AND cg.estudiante_id = u.id)` + extra + `
  FROM usuarios u
  JOIN perfiles_estudiante pe ON pe.usuario_id = u.id
  CROSS JOIN LATERAL (
    SELECT
      (SELECT count(*) FROM publicaciones p WHERE p.autor_id = u.id)::int AS pubs,
      (SELECT count(*) FROM propuestas_mejora pm WHERE pm.autor_id = u.id)::int AS propuestas,
      (SELECT count(*) FROM propuestas_mejora pm WHERE pm.autor_id = u.id AND pm.estado = 'aceptada')::int AS mejoras,
      ARRAY(
        SELECT p.lenguaje FROM publicaciones p WHERE p.autor_id = u.id AND p.lenguaje <> 'otro'
        UNION
        SELECT p.lenguaje FROM propuestas_mejora pm JOIN publicaciones p ON p.id = pm.publicacion_id
         WHERE pm.autor_id = u.id AND pm.estado = 'aceptada' AND p.lenguaje <> 'otro'
      ) AS verificadas,
      (SELECT count(DISTINCT date_trunc('week', a.f)) FROM (
         SELECT creado_en AS f FROM publicaciones WHERE autor_id = u.id
         UNION ALL SELECT creado_en FROM propuestas_mejora WHERE autor_id = u.id
         UNION ALL SELECT creado_en FROM comentarios WHERE autor_id = u.id
       ) a WHERE a.f > now() - interval '8 weeks')::int AS semanas
  ) m `
}

func escanearTarjeta(row pgx.Row, extra ...any) (model.TarjetaTalento, error) {
	var t model.TarjetaTalento
	var semestre *int16
	destinos := []any{&t.ID, &t.Nombre, &t.Titular, &t.GithubURL,
		&t.Ciudad, &t.Institucion, &t.Programa,
		&semestre, &t.EstadoAcademico, &t.Disponibilidad, &t.Modalidad,
		&t.Stack, &t.Verificadas, &t.Publicaciones, &t.PropuestasHechas, &t.MejorasAportadas, &t.SemanasActivas,
		&t.Guardado}
	if err := row.Scan(append(destinos, extra...)...); err != nil {
		return t, err
	}
	t.Semestre = aInt(semestre)
	if t.Stack == nil {
		t.Stack = []string{}
	}
	if t.Verificadas == nil {
		t.Verificadas = []string{}
	}
	// Misma fórmula que el portafolio: cuánto le aceptan de lo que propone.
	if t.PropuestasHechas > 0 {
		tasa := (t.MejorasAportadas*100 + t.PropuestasHechas/2) / t.PropuestasHechas
		t.TasaAceptacion = &tasa
	}
	return t, nil
}

// BuscarTalento combina todos los filtros; uno vacío no filtra. Ciudades e Institucion
// llegan en minúsculas y sin tildes. Ordena primero a quienes más han aportado y con más constancia.
func (r *EstudianteRepo) BuscarTalento(ctx context.Context, f model.FiltroTalento) ([]model.TarjetaTalento, error) {
	if f.Lenguajes == nil {
		f.Lenguajes = []string{}
	}
	if f.Ciudades == nil {
		f.Ciudades = []string{}
	}
	rows, err := r.db.Query(ctx, selectTarjeta("")+`
		 WHERE u.rol = 'estudiante'
		   AND (cardinality($2::text[]) = 0 OR pe.stack && $2::text[] OR m.verificadas && $2::text[])
		   AND (cardinality($3::text[]) = 0 OR `+sinTildes("COALESCE(pe.ciudad, '')")+` = ANY($3::text[]))
		   AND ($4::text = '' OR `+sinTildes("COALESCE(pe.institucion, '')")+` = $4::text)
		   AND ($5::text = ''
		        OR ($5::text = 'egresado' AND pe.estado_academico = 'egresado')
		        OR (COALESCE(pe.estado_academico, '') <> 'egresado' AND (
		              ($5::text = 'inicial' AND pe.semestre BETWEEN 1 AND 3)
		           OR ($5::text = 'medio' AND pe.semestre BETWEEN 4 AND 6)
		           OR ($5::text = 'avanzado' AND pe.semestre >= 7))))
		   AND ($6::text = '' OR pe.disponibilidad = $6::text)
		   AND ($7::text = '' OR pe.modalidad = $7::text)
		   AND (NOT $8::boolean OR m.mejoras > 0)
		 ORDER BY m.mejoras DESC, m.semanas DESC, m.pubs DESC, u.nombre
		 LIMIT 50`,
		f.EmpresaID, f.Lenguajes, f.Ciudades, f.Institucion, f.Nivel, f.Disponibilidad, f.Modalidad, f.ConMejoras)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []model.TarjetaTalento{}
	for rows.Next() {
		t, err := escanearTarjeta(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}
	return lista, rows.Err()
}

// ListarCandidatos devuelve los estudiantes que guardó la empresa, con su correo.
// El service decide si el correo se entrega (suscripción activa y contacto visible).
func (r *EstudianteRepo) ListarCandidatos(ctx context.Context, empresaID string) ([]model.Candidato, error) {
	rows, err := r.db.Query(ctx, selectTarjeta(", u.correo, g.creado_en, pe.contacto_visible")+`
		  JOIN candidatos_guardados g ON g.estudiante_id = u.id AND g.empresa_id = $1::uuid
		 ORDER BY g.creado_en DESC`, empresaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []model.Candidato{}
	for rows.Next() {
		var c model.Candidato
		t, err := escanearTarjeta(rows, &c.Correo, &c.GuardadoEn, &c.ContactoVisible)
		if err != nil {
			return nil, err
		}
		c.TarjetaTalento = t
		lista = append(lista, c)
	}
	return lista, rows.Err()
}

// GuardarCandidato agrega un estudiante a la lista de la empresa. Si ya estaba, no pasa nada.
// Devuelve ErrNoEncontrado si el ID no es de un estudiante y ErrLimite si la lista está llena.
func (r *EstudianteRepo) GuardarCandidato(ctx context.Context, empresaID, estudianteID string, maximo int) error {
	var existe bool
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM perfiles_estudiante WHERE usuario_id = $2::uuid),
		        (SELECT count(*) FROM candidatos_guardados WHERE empresa_id = $1::uuid)`,
		empresaID, estudianteID,
	).Scan(&existe, &total)
	if err != nil {
		return err
	}
	if !existe {
		return ErrNoEncontrado
	}
	if total >= maximo {
		return ErrLimite
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO candidatos_guardados (empresa_id, estudiante_id) VALUES ($1::uuid, $2::uuid)
		 ON CONFLICT DO NOTHING`, empresaID, estudianteID)
	return err
}

func (r *EstudianteRepo) QuitarCandidato(ctx context.Context, empresaID, estudianteID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM candidatos_guardados WHERE empresa_id = $1::uuid AND estudiante_id = $2::uuid`,
		empresaID, estudianteID)
	return err
}

// EsCandidatoGuardado indica si la empresa ya tiene al estudiante en su lista (para el perfil).
func (r *EstudianteRepo) EsCandidatoGuardado(ctx context.Context, empresaID, estudianteID string) (bool, error) {
	var guardado bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM candidatos_guardados WHERE empresa_id = $1::uuid AND estudiante_id = $2::uuid)`,
		empresaID, estudianteID).Scan(&guardado)
	return guardado, err
}

// DestacadosRecientes: quienes más mejoras lograron que les aceptaran en los últimos `dias`.
func (r *EstudianteRepo) DestacadosRecientes(ctx context.Context, dias, limite int) ([]model.EstudianteDestacado, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id::text, u.nombre, COALESCE(pe.institucion, ''), COALESCE(pe.github_url, ''), count(*)::int
		  FROM propuestas_mejora pm
		  JOIN usuarios u ON u.id = pm.autor_id
		  JOIN perfiles_estudiante pe ON pe.usuario_id = u.id
		 WHERE pm.estado = 'aceptada' AND pm.actualizado_en > now() - make_interval(days => $1::int)
		 GROUP BY u.id, u.nombre, pe.institucion, pe.github_url
		 ORDER BY count(*) DESC, max(pm.actualizado_en) DESC
		 LIMIT $2`, dias, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []model.EstudianteDestacado{}
	for rows.Next() {
		var d model.EstudianteDestacado
		if err := rows.Scan(&d.ID, &d.Nombre, &d.Institucion, &d.GithubURL, &d.Mejoras); err != nil {
			return nil, err
		}
		lista = append(lista, d)
	}
	return lista, rows.Err()
}

// Catalogos: instituciones y ciudades que de verdad aparecen en los perfiles, las más comunes primero.
// "UTS" escrita con y sin tildes o mayúsculas cuenta como una sola, con la escritura más usada.
func (r *EstudianteRepo) Catalogos(ctx context.Context) (*model.Catalogos, error) {
	consultar := func(columna string) ([]model.OpcionCatalogo, error) {
		rows, err := r.db.Query(ctx, `
			SELECT mode() WITHIN GROUP (ORDER BY valor), count(*)::int FROM (
			  SELECT trim(`+columna+`) AS valor FROM perfiles_estudiante WHERE COALESCE(trim(`+columna+`), '') <> ''
			) t
			GROUP BY `+sinTildes("valor")+` ORDER BY 2 DESC, 1
			LIMIT 60`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		lista := []model.OpcionCatalogo{}
		for rows.Next() {
			var o model.OpcionCatalogo
			if err := rows.Scan(&o.Valor, &o.Total); err != nil {
				return nil, err
			}
			lista = append(lista, o)
		}
		return lista, rows.Err()
	}

	instituciones, err := consultar("institucion")
	if err != nil {
		return nil, err
	}
	ciudades, err := consultar("ciudad")
	if err != nil {
		return nil, err
	}
	return &model.Catalogos{Instituciones: instituciones, Ciudades: ciudades}, nil
}
