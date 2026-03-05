package domain

import "time"

type UserID uint64
type SessionID uint64

type UserCreateRequest struct {
	Email string
	Login string
}

type UserLoginRequest struct {
	Email    string
	Login    string
	Password string
}

type User struct {
	ID    UserID
	Email string
	Login string
}

type PasswordHash struct {
	Algo string
	Hash []byte
}

type Credentials struct {
	UserID       UserID
	PasswordHash PasswordHash
}

type TokenPair struct {
	AccessToken   string
	RefreshToken  string
	AccessExpires time.Time
}

type RefreshSession struct {
	ID        SessionID
	UserID    UserID
	TokenHash []byte
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}
