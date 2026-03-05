package usecase

import (
	"context"
	d "todo/auth-service/internal/domain"
	"todo/auth-service/internal/security"
)

type LoginUser struct {
	s d.SessionRepo
	u d.UserRepo
}

func NewLoginUser(s d.SessionRepo, u d.UserRepo, i security.Issuer) *LoginUser {
	return &LoginUser{
		s: s,
		u: u,
	}
}

func (c *LoginUser) Exec(ctx context.Context, l *d.UserLoginRequest) (d.User, d.TokenPair, error) {

}
