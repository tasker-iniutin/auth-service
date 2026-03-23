package usecase

import (
	"context"
	"crypto/sha256"

	d "github.com/tasker-iniutin/auth-service/internal/domain"
)

type LogoutUser struct {
	s d.SessionRepo
}

func NewLogoutUser(s d.SessionRepo) *LogoutUser {
	return &LogoutUser{
		s: s,
	}
}

func (c *LogoutUser) Exec(ctx context.Context, refreshToken string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if refreshToken == "" {
		return d.ErrValidation
	}

	h := sha256.Sum256([]byte(refreshToken))
	return c.s.RevokeRefresh(ctx, h[:])
}
