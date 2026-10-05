package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepo struct {
	db *pgxpool.Pool
}

func NuevoTokenRepo(db *pgxpool.Pool) *TokenRepo {
	return &TokenRepo{db: db}
}

// Guardar registra el hash de un refresh token nuevo. El token en sí nunca se guarda.
func (r *TokenRepo) Guardar(ctx context.Context, usuarioID, hash string, expira time.Time) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO refresh_tokens (usuario_id, token_hash, expira_en) VALUES ($1, $2, $3)`,
		usuarioID, hash, expira,
	)
	return err
}

// Consumir revoca un token vigente y devuelve a quién pertenecía, en una sola
// sentencia atómica: dos peticiones simultáneas con el mismo token no pueden
// usarlo las dos. Si el token no existe, ya se usó o venció, devuelve ErrNoEncontrado.
func (r *TokenRepo) Consumir(ctx context.Context, hash string) (string, error) {
	var usuarioID string
	err := r.db.QueryRow(ctx,
		`UPDATE refresh_tokens
		    SET revocado_en = now()
		  WHERE token_hash = $1 AND revocado_en IS NULL AND expira_en > now()
		  RETURNING usuario_id::text`,
		hash,
	).Scan(&usuarioID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNoEncontrado
	}
	return usuarioID, err
}
