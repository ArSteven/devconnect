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
	ErrNoEncontrado     = errors.New("no encontrado")
	ErrCorreoDuplicado  = errors.New("correo duplicado")
)

type UsuarioRepo struct {
	db *pgxpool.Pool
}

func NuevoUsuarioRepo(db *pgxpool.Pool) *UsuarioRepo {
	return &UsuarioRepo{db: db}
}

// Crear inserta el usuario y, en la misma transacción, su perfil de
// estudiante o su registro de empresa. Si algo falla no queda nada a medias.
func (r *UsuarioRepo) Crear(ctx context.Context, u *model.Usuario, razonSocial string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no hace nada si ya se hizo Commit

	err = tx.QueryRow(ctx,
		`INSERT INTO usuarios (correo, nombre, hash_contrasena, rol)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text, creado_en`,
		u.Correo, u.Nombre, u.HashContrasena, u.Rol,
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
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *UsuarioRepo) BuscarPorCorreo(ctx context.Context, correo string) (*model.Usuario, error) {
	return r.buscar(ctx, `WHERE correo = $1`, correo)
}

func (r *UsuarioRepo) BuscarPorID(ctx context.Context, id string) (*model.Usuario, error) {
	return r.buscar(ctx, `WHERE id = $1`, id)
}

func (r *UsuarioRepo) buscar(ctx context.Context, filtro string, valor string) (*model.Usuario, error) {
	var u model.Usuario
	err := r.db.QueryRow(ctx,
		`SELECT id::text, correo, nombre, hash_contrasena, rol, creado_en FROM usuarios `+filtro,
		valor,
	).Scan(&u.ID, &u.Correo, &u.Nombre, &u.HashContrasena, &u.Rol, &u.CreadoEn)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
