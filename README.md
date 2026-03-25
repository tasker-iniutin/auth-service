# `auth-service`

Authentication microservice for the Todo API project.

It exposes a gRPC API, stores users in PostgreSQL, stores refresh sessions in Redis, issues RS256 access tokens, and rotates opaque refresh tokens.

## Responsibility

`auth-service` is responsible for:

- user registration;
- user login by `email` or `login`;
- issuing access and refresh tokens;
- rotating refresh tokens;
- revoking refresh sessions on logout.

It does not manage tasks or HTTP routing. HTTP is handled by `api-gateway`.

## Architecture

The service follows a layered structure:

- `cmd/auth-service`
  entry point;
- `internal/app`
  bootstrap and dependency wiring;
- `internal/domain`
  models, errors, repository contracts;
- `internal/usecase`
  business logic;
- `internal/store/postgre`
  PostgreSQL user repository;
- `internal/store/redis`
  Redis session repository;
- `internal/transport/grpc`
  gRPC handlers;
- `migrations`
  database schema.

Shared infrastructure lives in `common`:

- `common/configenv`
- `common/postgres`
- `common/runtime`
- `common/authsecurity`

### Why This Structure

The goal is explicit separation of concerns:

- transport maps protobuf to domain;
- use cases contain business logic;
- repositories contain infrastructure code;
- bootstrap stays in one place.

For Go, this keeps dependencies simple and makes the code easier to review.

## API

The protobuf contract is defined in `api-contracts/proto/auth/v1alpha/auth.proto`.

Exposed operations:

| Method | Purpose |
| --- | --- |
| `Register` | Create user and issue tokens |
| `Login` | Authenticate user and issue tokens |
| `Refresh` | Rotate refresh token and issue new access token |
| `Logout` | Revoke refresh session |

## Token Design

### Access Tokens

Access tokens are JWTs signed with `RS256`.

Why `RS256`:

- `auth-service` keeps the private key;
- other services verify tokens using only the public key;
- this fits microservice boundaries better than a shared symmetric secret.

### Refresh Tokens

Refresh tokens are opaque random strings, not JWTs.

Only their SHA-256 hash is stored in Redis.

Why:

- refresh tokens act as session handles, not self-describing documents;
- storing only the hash reduces exposure if Redis is leaked;
- rotation and revocation stay simple.

## Storage Design

### PostgreSQL

PostgreSQL stores durable identity data:

- `users`
- `credentials`

Why PostgreSQL:

- user data must survive restarts;
- uniqueness and integrity constraints matter;
- relational schema fits this model well.

### Redis

Redis stores refresh sessions.

Why Redis:

- sessions are short-lived;
- they need TTL support;
- they are looked up by token hash;
- they are frequently revoked and replaced.

This is a clean split:

- PostgreSQL for durable data;
- Redis for volatile session state.

## Database Schema

Migration: `auth-service/migrations/001_create_users.sql`

### `users`

- `id`
- `email`
- `login`
- non-empty checks;
- case-insensitive unique indexes on `email` and `login`.

### `credentials`

- `user_id`
- `password_algo`
- `password_hash`

`credentials.user_id` references `users(id)` with `ON DELETE CASCADE`.

### Why `users` and `credentials` Are Separate

This separates identity data from secret material.

It keeps the model cleaner and leaves room for future auth methods without overloading the `users` table.

### Why `password_hash` Uses `BYTEA`

The domain model treats the hash as bytes, so `BYTEA` is a more accurate type than `TEXT`.

## Main Flows

### Register

1. validate input;
2. hash password with `bcrypt`;
3. create user in PostgreSQL;
4. issue access and refresh tokens;
5. store refresh session in Redis.

### Login

1. load user by `email` or `login`;
2. load credentials;
3. compare password with `bcrypt`;
4. issue tokens;
5. create refresh session.

### Refresh

1. hash incoming refresh token;
2. load refresh session by hash;
3. check expiration;
4. issue new access token;
5. create new refresh session;
6. revoke old session.

### Logout

1. hash incoming refresh token;
2. revoke refresh session by hash.

### Why Rotation Is Used

Refresh token rotation limits damage from token reuse. After refresh, the previous session should no longer be valid.

## Repository Approach

Repositories are implemented with `pgx` and explicit SQL.

### Why Not an ORM

This project is educational, so explicit SQL is preferable:

- queries stay visible;
- constraints are easier to understand;
- transaction boundaries stay explicit;
- debugging is simpler.

For a service of this size, a small amount of repetition is acceptable.

## Configuration

Configuration is provided through environment variables.

Main variables:

- `AUTH_GRPC_ADDR`
- `JWT_PRIVATE_KEY_PEM`
- `JWT_ISSUER`
- `JWT_AUDIENCE`
- `JWT_ACCESS_TTL`
- `JWT_KEY_ID`
- `ENABLE_GRPC_REFLECTION`
- `DATABASE_URL`
- `REDIS_ADDR`
- `REDIS_PASSWORD`

Example values are in `auth-service/.env.example`.

## Local Run

Requirements:

- Go
- Docker / Docker Compose
- `goose`
- RSA private key in PEM format

Start infrastructure:

```bash
cd auth-service
make db-up
make migrate-up
```

Run service:

```bash
cd auth-service
export JWT_PRIVATE_KEY_PEM=/absolute/path/to/private.pem
go run ./cmd/auth-service
```

Defaults:

- PostgreSQL: `localhost:5433`
- Redis: `localhost:6379`

## Testing

Tests included:

- use case tests: `auth-service/internal/usecase/usecase_test.go`
- PostgreSQL repo tests: `auth-service/internal/store/postgre/user_repo_test.go`

Run:

```bash
go test ./...
```

Or via Docker (uses `GOCACHE=/tmp/go-build` inside the container):

```bash
docker compose --profile test up --abort-on-container-exit test
```

Repository tests use a real PostgreSQL instance and read:

- `AUTH_TEST_DATABASE_URL`, or
- `DATABASE_URL`

If no DSN is set, they skip.

Why real repo tests matter:

- they catch SQL mistakes;
- they verify constraints and indexes;
- they verify `BYTEA` handling and transaction behavior.

## Current Limitations

- no email verification;
- no password reset flow;
- no brute-force protection or rate limiting;
- refresh sessions do not yet store device or client metadata;
- no structured audit logging.

## Summary

Main design choices:

- separate authentication into its own service;
- use `RS256` for signed access tokens;
- use opaque refresh tokens with hash-based storage;
- keep users in PostgreSQL and sessions in Redis;
- use explicit SQL with `pgx`;
- keep secrets outside the repository and inject them through environment variables.

These choices favor explicitness, clear boundaries, and understandable behavior over convenience abstractions.
