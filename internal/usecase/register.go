package usecase

import (
	"context"
	d "todo/auth-service/internal/domain"
	"todo/auth-service/internal/security"
)

type RegisterUser struct {
	s d.SessionRepo
	u d.UserRepo
}

func NewRegisterUser(s d.SessionRepo, u d.UserRepo, i security.Issuer) *RegisterUser {
	return &RegisterUser{
		s: s,
		u: u,
	}
}

func (c *RegisterUser) Exec(ctx context.Context, r d.UserCreateRequest, password string) (d.User, d.TokenPair, error) {

}
