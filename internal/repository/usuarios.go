package repository

import (
	"context"
	"errors"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoEncontrado    = errors.New("no encontrado")
	ErrCorreoDuplicado = errors.New("correo duplicado")
	ErrLimite          = errors.New("límite alcanzado")
)

type UsuarioRepo struct {
	db *pgxpool.Pool
}

func NuevoUsuarioRepo(db *pgxpool.Pool) *UsuarioRepo {
	return &UsuarioRepo{db: db}
}

// Crear inserta el usuario y, en la misma transacción, su perfil de
// estudiante o su registro de empresa. Si algo falla no queda nada a medias.
// La hora de aceptación de los términos la pone la base, no el cliente.
func (r *UsuarioRepo) Crear(ctx context.Context, u *model.Usuario, razonSocial string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no hace nada si ya se hizo Commit

	err = tx.QueryRow(ctx,
		`INSERT INTO usuarios (correo, nombre, hash_contrasena, rol, terminos_aceptados_en, version_terminos)
		 VALUES ($1, $2, $3, $4, now(), $5)
		 RETURNING id::text, creado_en`,
		u.Correo, u.Nombre, u.HashContrasena, u.Rol, u.VersionTerminos,
	).Scan(&u.ID, &u.CreadoEn)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // violación de UNIQUE
			return ErrCorreoDuplicado
		}
		return err
	}

	switch u.Rol {
	case "estudiante":
		_, err = tx.Exec(ctx, `INSERT INTO perfiles_estudiante (usuario_id) VALUES ($1)`, u.ID)
	case "empresa":
		_, err = tx.Exec(ctx, `INSERT INTO empresas (usuario_id, razon_social) VALUES ($1, $2)`, u.ID, razonSocial)
		u.RazonSocial = razonSocial
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *UsuarioRepo) BuscarPorCorreo(ctx context.Context, correo string) (*model.Usuario, error) {
	return r.buscar(ctx, `WHERE u.correo = $1`, correo)
}

func (r *UsuarioRepo) BuscarPorID(ctx context.Context, id string) (*model.Usuario, error) {
	return r.buscar(ctx, `WHERE u.id = $1`, id)
}

// buscar trae también el GitHub (estudiantes) o la razón social (empresas) para el encabezado.
func (r *UsuarioRepo) buscar(ctx context.Context, filtro string, valor string) (*model.Usuario, error) {
	var u model.Usuario
	err := r.db.QueryRow(ctx,
		`SELECT u.id::text, u.correo, u.nombre, u.hash_contrasena, u.rol, u.creado_en,
		        COALESCE(pe.github_url, ''), COALESCE(e.razon_social, '')
		   FROM usuarios u
		   LEFT JOIN perfiles_estudiante pe ON pe.usuario_id = u.id
		   LEFT JOIN empresas e ON e.usuario_id = u.id `+filtro,
		valor,
	).Scan(&u.ID, &u.Correo, &u.Nombre, &u.HashContrasena, &u.Rol, &u.CreadoEn, &u.GithubURL, &u.RazonSocial)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
