package usecase

import (
	"context"
	"crypto/sha256"

	d "github.com/tasker-iniutin/auth-service/internal/domain"
	sec "github.com/tasker-iniutin/common/authsecurity"
)

type LogoutUser struct {
	s d.SessionRepo
	v sec.Verifier
}

func NewLogoutUser(s d.SessionRepo, v sec.Verifier) *LogoutUser {
	return &LogoutUser{
		s: s,
		v: v,
	}
}

func (c *LogoutUser) Exec(ctx context.Context, refreshToken string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if refreshToken == "" {
		return d.ErrValidation
	}
	if _, err := c.v.VerifyAccess(refreshToken); err != nil {
		return d.ErrUnauthorized
	}

	h := sha256.Sum256([]byte(refreshToken))
	return c.s.RevokeRefresh(ctx, h[:])
}
