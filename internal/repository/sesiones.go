package repository

import (
	"context"
	"errors"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SesionRepo struct {
	db *pgxpool.Pool
}

func NuevoSesionRepo(db *pgxpool.Pool) *SesionRepo {
	return &SesionRepo{db: db}
}

const selectSesion = `
SELECT s.id::text, s.anfitrion_id::text, u.nombre, COALESCE(pe.github_url, ''), s.titulo, COALESCE(s.descripcion, ''),
       s.inicia_en, s.iniciada_en, s.sala_jitsi, s.estado
  FROM sesiones_vivo s
  JOIN usuarios u ON u.id = s.anfitrion_id
  LEFT JOIN perfiles_estudiante pe ON pe.usuario_id = s.anfitrion_id `

func escanearSesion(row pgx.Row) (model.SesionVivo, error) {
	var s model.SesionVivo
	err := row.Scan(&s.ID, &s.AnfitrionID, &s.AnfitrionNombre, &s.AnfitrionGithub, &s.Titulo, &s.Descripcion,
		&s.IniciaEn, &s.IniciadaEn, &s.Sala, &s.Estado)
	return s, err
}

// Listar devuelve primero las que están en vivo y luego las próximas. Una sala que
// lleva más de 6 horas "en vivo" se da por olvidada y deja de mostrarse.
func (r *SesionRepo) Listar(ctx context.Context) ([]model.SesionVivo, error) {
	rows, err := r.db.Query(ctx, selectSesion+`
		WHERE (s.estado = 'en_vivo' AND COALESCE(s.iniciada_en, s.inicia_en) > now() - interval '6 hours')
		   OR (s.estado = 'programada' AND s.inicia_en > now() - interval '3 hours')
		ORDER BY (s.estado = 'en_vivo') DESC, s.inicia_en
		LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []model.SesionVivo{}
	for rows.Next() {
		s, err := escanearSesion(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, s)
	}
	return lista, rows.Err()
}

func (r *SesionRepo) Obtener(ctx context.Context, id string) (*model.SesionVivo, error) {
	s, err := escanearSesion(r.db.QueryRow(ctx, selectSesion+`WHERE s.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SesionRepo) Crear(ctx context.Context, s *model.SesionVivo) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO sesiones_vivo (anfitrion_id, titulo, descripcion, inicia_en, sala_jitsi)
		 VALUES ($1, $2, NULLIF($3, ''), $4, $5)
		 RETURNING id::text, estado`,
		s.AnfitrionID, s.Titulo, s.Descripcion, s.IniciaEn, s.Sala,
	).Scan(&s.ID, &s.Estado)
}

// CambiarEstado guarda además cuándo empezó de verdad: así una sesión cancelada
// (programada -> finalizada) no cuenta como dictada en el portafolio.
func (r *SesionRepo) CambiarEstado(ctx context.Context, id, estado string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE sesiones_vivo
		    SET estado = $2,
		        iniciada_en = CASE WHEN $2 = 'en_vivo' THEN now() ELSE iniciada_en END,
		        actualizado_en = now()
		  WHERE id = $1`, id, estado)
	return err
}
