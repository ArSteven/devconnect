package repository

import (
	"context"
	"errors"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PublicacionRepo struct {
	db *pgxpool.Pool
}

func NuevoPublicacionRepo(db *pgxpool.Pool) *PublicacionRepo {
	return &PublicacionRepo{db: db}
}

// En el detalle viaja el código completo; en el feed, solo las primeras 12 líneas.
const (
	codigoCompleto  = `p.codigo`
	codigoRecortado = `array_to_string((string_to_array(p.codigo, E'\n'))[1:12], E'\n')`
)

// selectPublicacion arma el SELECT común. En un reto el autor es una empresa:
// se muestra su razón social. codigo es siempre una de las constantes de arriba.
func selectPublicacion(codigo string) string {
	return `
SELECT p.id::text, p.autor_id::text, COALESCE(e.razon_social, u.nombre),
       COALESCE(pe.institucion, ''), COALESCE(pe.github_url, ''),
       p.titulo, COALESCE(p.descripcion, ''), p.lenguaje, ` + codigo + `,
       COALESCE(array_length(string_to_array(p.codigo, E'\n'), 1), 0),
       p.estado, p.tipo, p.fecha_limite, p.creado_en,
       (SELECT count(*) FROM propuestas_mejora pm WHERE pm.publicacion_id = p.id),
       (SELECT count(*) FROM comentarios c WHERE c.publicacion_id = p.id)
  FROM publicaciones p
  JOIN usuarios u ON u.id = p.autor_id
  LEFT JOIN perfiles_estudiante pe ON pe.usuario_id = p.autor_id
  LEFT JOIN empresas e ON e.usuario_id = p.autor_id `
}

func escanearPublicacion(row pgx.Row) (model.Publicacion, error) {
	var p model.Publicacion
	err := row.Scan(&p.ID, &p.AutorID, &p.AutorNombre, &p.AutorInstitucion, &p.AutorGithub,
		&p.Titulo, &p.Descripcion, &p.Lenguaje, &p.Codigo, &p.TotalLineas,
		&p.Estado, &p.Tipo, &p.FechaLimite, &p.CreadoEn, &p.Propuestas, &p.Comentarios)
	return p, err
}

// Listar devuelve el feed más reciente primero. Todos los filtros se combinan y
// uno vacío no filtra. Q e Institucion llegan ya en minúsculas y sin tildes.
func (r *PublicacionRepo) Listar(ctx context.Context, f model.FiltroPublicaciones, limite, offset int) ([]model.Publicacion, error) {
	rows, err := r.db.Query(ctx, selectPublicacion(codigoRecortado)+`
		WHERE ($1 = '' OR p.lenguaje = $1)
		  AND ($2 = '' OR p.estado = $2)
		  AND ($3 = '' OR p.tipo = $3)
		  AND ($4 = '' OR `+sinTildes("p.titulo")+` LIKE '%' || $4 || '%'
		               OR `+sinTildes("COALESCE(p.descripcion, '')")+` LIKE '%' || $4 || '%')
		  AND ($5 = '' OR `+sinTildes("COALESCE(pe.institucion, '')")+` = $5)
		ORDER BY p.creado_en DESC
		LIMIT $6 OFFSET $7`,
		f.Lenguaje, f.Estado, f.Tipo, escaparLike(f.Q), f.Institucion, limite, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []model.Publicacion{}
	for rows.Next() {
		p, err := escanearPublicacion(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, p)
	}
	return lista, rows.Err()
}

func (r *PublicacionRepo) Obtener(ctx context.Context, id string) (*model.Publicacion, error) {
	p, err := escanearPublicacion(r.db.QueryRow(ctx, selectPublicacion(codigoCompleto)+`WHERE p.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Crear guarda una pregunta o un reto; p.Tipo y p.FechaLimite los fija el service.
func (r *PublicacionRepo) Crear(ctx context.Context, p *model.Publicacion) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO publicaciones (autor_id, titulo, descripcion, lenguaje, codigo, tipo, fecha_limite)
		 VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7)
		 RETURNING id::text, estado, creado_en`,
		p.AutorID, p.Titulo, p.Descripcion, p.Lenguaje, p.Codigo, p.Tipo, p.FechaLimite,
	).Scan(&p.ID, &p.Estado, &p.CreadoEn)
}

func (r *PublicacionRepo) ListarPropuestas(ctx context.Context, publicacionID string) ([]model.Propuesta, error) {
	rows, err := r.db.Query(ctx,
		`SELECT pm.id::text, pm.publicacion_id::text, pm.autor_id::text, u.nombre, COALESCE(pe.github_url, ''),
		        pm.codigo, pm.explicacion, pm.estado, pm.creado_en
		   FROM propuestas_mejora pm
		   JOIN usuarios u ON u.id = pm.autor_id
		   LEFT JOIN perfiles_estudiante pe ON pe.usuario_id = pm.autor_id
		  WHERE pm.publicacion_id = $1
		  ORDER BY pm.creado_en`, publicacionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []model.Propuesta{}
	for rows.Next() {
		var p model.Propuesta
		if err := rows.Scan(&p.ID, &p.PublicacionID, &p.AutorID, &p.AutorNombre, &p.AutorGithub,
			&p.Codigo, &p.Explicacion, &p.Estado, &p.CreadoEn); err != nil {
			return nil, err
		}
		lista = append(lista, p)
	}
	return lista, rows.Err()
}

func (r *PublicacionRepo) ListarComentarios(ctx context.Context, publicacionID string) ([]model.Comentario, error) {
	rows, err := r.db.Query(ctx,
		`SELECT c.id::text, c.autor_id::text, COALESCE(e.razon_social, u.nombre), COALESCE(pe.github_url, ''),
		        u.rol, c.texto, c.creado_en
		   FROM comentarios c
		   JOIN usuarios u ON u.id = c.autor_id
		   LEFT JOIN perfiles_estudiante pe ON pe.usuario_id = c.autor_id
		   LEFT JOIN empresas e ON e.usuario_id = c.autor_id
		  WHERE c.publicacion_id = $1
		  ORDER BY c.creado_en`, publicacionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []model.Comentario{}
	for rows.Next() {
		var c model.Comentario
		if err := rows.Scan(&c.ID, &c.AutorID, &c.AutorNombre, &c.AutorGithub, &c.AutorRol, &c.Texto, &c.CreadoEn); err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, rows.Err()
}

func (r *PublicacionRepo) CrearPropuesta(ctx context.Context, p *model.Propuesta) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO propuestas_mejora (publicacion_id, autor_id, codigo, explicacion)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text, estado, creado_en`,
		p.PublicacionID, p.AutorID, p.Codigo, p.Explicacion,
	).Scan(&p.ID, &p.Estado, &p.CreadoEn)
}

func (r *PublicacionRepo) InfoPropuesta(ctx context.Context, propuestaID string) (*model.PropuestaInfo, error) {
	var i model.PropuestaInfo
	err := r.db.QueryRow(ctx,
		`SELECT pm.estado, pm.publicacion_id::text, p.autor_id::text, p.tipo
		   FROM propuestas_mejora pm
		   JOIN publicaciones p ON p.id = pm.publicacion_id
		  WHERE pm.id = $1`, propuestaID,
	).Scan(&i.Estado, &i.PublicacionID, &i.AutorPublicacion, &i.TipoPublicacion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

// Decidir cambia una propuesta pendiente a aceptada o rechazada. Si se acepta,
// la publicación queda resuelta. Todo en una transacción. Devuelve ErrNoEncontrado
// si la propuesta ya no estaba pendiente (por ejemplo, otra petición la decidió antes).
func (r *PublicacionRepo) Decidir(ctx context.Context, propuestaID, estado string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var publicacionID string
	err = tx.QueryRow(ctx,
		`UPDATE propuestas_mejora SET estado = $2, actualizado_en = now()
		  WHERE id = $1 AND estado = 'pendiente'
		  RETURNING publicacion_id::text`,
		propuestaID, estado,
	).Scan(&publicacionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoEncontrado
	}
	if err != nil {
		return err
	}

	if estado == "aceptada" {
		if _, err := tx.Exec(ctx,
			`UPDATE publicaciones SET estado = 'resuelta', actualizado_en = now() WHERE id = $1`,
			publicacionID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PublicacionRepo) CrearComentario(ctx context.Context, publicacionID string, c *model.Comentario) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO comentarios (publicacion_id, autor_id, texto)
		 VALUES ($1, $2, $3)
		 RETURNING id::text, creado_en`,
		publicacionID, c.AutorID, c.Texto,
	).Scan(&c.ID, &c.CreadoEn)
}
