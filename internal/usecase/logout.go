package usecase

import (
	"context"

	d "github.com/tasker-iniutin/auth-service/internal/domain"
	sec "github.com/tasker-iniutin/common/authsecurity"
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

	h := sec.RefreshHash(refreshToken)
	return c.s.RevokeRefresh(ctx, h)
}
