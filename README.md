# auth-service

gRPC service for user authentication. It issues RS256 JWT access tokens and opaque refresh tokens, stores refresh sessions in Redis, and keeps user records in an in-memory repository.

**Features**
- Register and login users with email/login + password.
- Issue access tokens (15 min TTL) and refresh tokens (14 days TTL).
- Refresh token rotation with revocation of old sessions.
- Logout by revoking the refresh session.

**Dependencies**
- Redis for refresh session storage.
- RSA keys for JWT signing in `auth-service/keys/private.pem`.

**Configuration**
- `REDIS_ADDR` (default `127.0.0.1:6379`)
- `REDIS_PASSWORD` (optional)
- gRPC listen address is hardcoded to `:50052` in `auth-service/cmd/auth-service/main.go`.

**Run**
```bash
cd /home/p0tniy/Documents/projects/todo/auth-service
REDIS_ADDR=127.0.0.1:6379 go run ./cmd/auth-service
```

**API (gRPC / HTTP via gateway)**
Auth service methods and HTTP annotations come from `api-contracts/proto/auth/v1alpha/auth.proto`.

| Method | Request | Response | HTTP route |
| --- | --- | --- | --- |
| Register | `RegisterRequest { email, login, password }` | `AuthResponse { user, tokens }` | `POST /v1alpha/auth/register` |
| Login | `LoginRequest { email|login, password }` | `AuthResponse { user, tokens }` | `POST /v1alpha/auth/login` |
| Refresh | `RefreshRequest { refresh_token }` | `TokenPair { access_token, refresh_token }` | `POST /v1alpha/auth/refresh` |
| Logout | `LogoutRequest { refresh_token }` | `google.protobuf.Empty` | `POST /v1alpha/auth/logout` |

**Storage model**
- Users are stored in-memory only (`auth-service/internal/store/mem/user_repo.go`). Data is lost on restart.
- Refresh sessions are stored in Redis, indexed by token hash and user ID (`auth-service/internal/store/redis/session_repo.go`).

**Keys**
- Private key: `auth-service/keys/private.pem` (used for signing access tokens).
- Public key: `auth-service/keys/public.pem` (used by other services to verify access tokens).

**Notes**
- Access tokens are issued with issuer `todo-auth` and audience `todo-api`.
- Refresh tokens are rotated on every refresh and the old session is revoked.
