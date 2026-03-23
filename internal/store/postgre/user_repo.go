package postgre

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	d "github.com/tasker-iniutin/auth-service/internal/domain"
)

type userRepoImpl struct {
	db *pgxpool.Pool
}

func NewPostgreRepo(db *pgxpool.Pool) *userRepoImpl {
	return &userRepoImpl{db: db}
}

func (r *userRepoImpl) Create(ctx context.Context, u d.UserCreateRequest, password d.PasswordHash) (d.User, error) {
	const q1 = `
		INSERT INTO users (email, login)
		VALUES ($1, $2)
		RETURNING id, email, login
	`
	const q2 = `
		INSERT INTO credentials (user_id, password_algo, password_hash)
		VALUES ($1, $2, $3)
	`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return d.User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var nU d.User
	err = tx.QueryRow(
		ctx,
		q1,
		u.Email,
		u.Login,
	).Scan(
		&nU.ID,
		&nU.Email,
		&nU.Login,
	)
	if err != nil {
		return d.User{}, mapPGError(err)
	}

	_, err = tx.Exec(
		ctx,
		q2,
		nU.ID,
		password.Algo,
		password.Hash,
	)
	if err != nil {
		return d.User{}, mapPGError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return d.User{}, fmt.Errorf("commit tx: %w", err)
	}

	return nU, nil
}

func (r *userRepoImpl) GetByEmail(ctx context.Context, email string) (d.User, error) {
	const q = `
		SELECT id, email, login
		FROM users
		WHERE lower(email) = lower($1)
	`

	var u d.User
	err := r.db.QueryRow(ctx, q, email).Scan(
		&u.ID,
		&u.Email,
		&u.Login,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return d.User{}, d.ErrNotFound
		}
		return d.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return u, nil
}

func (r *userRepoImpl) GetByLogin(ctx context.Context, login string) (d.User, error) {
	const q = `
		SELECT id, email, login
		FROM users
		WHERE lower(login) = lower($1)
	`

	var u d.User
	err := r.db.QueryRow(ctx, q, login).Scan(
		&u.ID,
		&u.Email,
		&u.Login,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return d.User{}, d.ErrNotFound
		}
		return d.User{}, fmt.Errorf("get user by login: %w", err)
	}

	return u, nil
}

func (r *userRepoImpl) GetCredentials(ctx context.Context, id d.UserID) (d.Credentials, error) {
	const q = `
		SELECT user_id, password_algo, password_hash
		FROM credentials
		WHERE user_id = $1
	`

	var c d.Credentials
	err := r.db.QueryRow(ctx, q, id).Scan(
		&c.UserID,
		&c.PasswordHash.Algo,
		&c.PasswordHash.Hash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return d.Credentials{}, d.ErrNotFound
		}
		return d.Credentials{}, fmt.Errorf("get credentials: %w", err)
	}

	return c, nil
}

func mapPGError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return d.ErrConflict
		case "23514", "23502":
			return d.ErrValidation
		}
	}

	return err
}
