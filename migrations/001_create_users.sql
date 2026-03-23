-- +goose Up
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    login VARCHAR(255) NOT NULL,
    CONSTRAINT users_email_not_empty CHECK (btrim(email) <> ''),
    CONSTRAINT users_login_not_empty CHECK (btrim(login) <> '')
);

CREATE UNIQUE INDEX ux_users_email_lower ON users (lower(email));
CREATE UNIQUE INDEX ux_users_login_lower ON users (lower(login));

CREATE TABLE credentials (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_algo VARCHAR(128) NOT NULL,
    password_hash BYTEA NOT NULL,
    CONSTRAINT credentials_password_algo_not_empty CHECK (btrim(password_algo) <> '')
);

-- +goose Down
DROP TABLE IF EXISTS credentials;
DROP INDEX IF EXISTS ux_users_login_lower;
DROP INDEX IF EXISTS ux_users_email_lower;
DROP TABLE IF EXISTS users;
