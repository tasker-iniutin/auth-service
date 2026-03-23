package postgre

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	d "github.com/tasker-iniutin/auth-service/internal/domain"
)

func TestUserRepoCreateAndRead(t *testing.T) {
	db := openAuthTestDB(t)
	repo := NewPostgreRepo(db)

	user, err := repo.Create(context.Background(), d.UserCreateRequest{
		Email: "user@example.com",
		Login: "user",
	}, d.PasswordHash{
		Algo: "bcrypt",
		Hash: []byte("hash"),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	gotByEmail, err := repo.GetByEmail(context.Background(), "USER@example.com")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if gotByEmail.ID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, gotByEmail.ID)
	}

	gotByLogin, err := repo.GetByLogin(context.Background(), "USER")
	if err != nil {
		t.Fatalf("get by login: %v", err)
	}
	if gotByLogin.ID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, gotByLogin.ID)
	}

	cred, err := repo.GetCredentials(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get credentials: %v", err)
	}
	if cred.PasswordHash.Algo != "bcrypt" {
		t.Fatalf("expected bcrypt, got %q", cred.PasswordHash.Algo)
	}
	if string(cred.PasswordHash.Hash) != "hash" {
		t.Fatalf("unexpected hash: %q", string(cred.PasswordHash.Hash))
	}
}

func TestUserRepoCreateConflict(t *testing.T) {
	db := openAuthTestDB(t)
	repo := NewPostgreRepo(db)

	_, err := repo.Create(context.Background(), d.UserCreateRequest{
		Email: "user@example.com",
		Login: "user",
	}, d.PasswordHash{
		Algo: "bcrypt",
		Hash: []byte("hash"),
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	_, err = repo.Create(context.Background(), d.UserCreateRequest{
		Email: "USER@example.com",
		Login: "other",
	}, d.PasswordHash{
		Algo: "bcrypt",
		Hash: []byte("hash"),
	})
	if !errors.Is(err, d.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestUserRepoNotFound(t *testing.T) {
	db := openAuthTestDB(t)
	repo := NewPostgreRepo(db)

	_, err := repo.GetByEmail(context.Background(), "missing@example.com")
	if !errors.Is(err, d.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func openAuthTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("AUTH_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("AUTH_TEST_DATABASE_URL or DATABASE_URL is not set")
	}

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	setupAuthSchema(t, db)
	return db
}

func setupAuthSchema(t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	const schema = `
		CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			login VARCHAR(255) NOT NULL,
			CONSTRAINT users_email_not_empty CHECK (btrim(email) <> ''),
			CONSTRAINT users_login_not_empty CHECK (btrim(login) <> '')
		);
		CREATE UNIQUE INDEX IF NOT EXISTS ux_users_email_lower ON users (lower(email));
		CREATE UNIQUE INDEX IF NOT EXISTS ux_users_login_lower ON users (lower(login));
		CREATE TABLE IF NOT EXISTS credentials (
			user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			password_algo VARCHAR(128) NOT NULL,
			password_hash BYTEA NOT NULL,
			CONSTRAINT credentials_password_algo_not_empty CHECK (btrim(password_algo) <> '')
		);
		TRUNCATE TABLE credentials, users RESTART IDENTITY CASCADE;
	`

	if _, err := db.Exec(context.Background(), schema); err != nil {
		t.Fatalf("setup auth schema: %v", err)
	}
}
