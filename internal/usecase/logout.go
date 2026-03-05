package usecase

import (
	"context"
	d "todo/auth-service/internal/domain"
	"todo/auth-service/internal/security"
)

type LogoutUser struct {
	s d.SessionRepo
	u d.UserRepo
}

func NewLogoutUser(s d.SessionRepo, i security.Verifier) *LogoutUser {
	return &LogoutUser{
		s: s,
		u: u,
	}
}

func (c *LogoutUser) Exec(ctx context.Context, refreshToken string) error {

}
