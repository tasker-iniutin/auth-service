package usecase

import (
	"context"
	d "todo/auth-service/internal/domain"
	"todo/auth-service/internal/security"
)

type RefreshUser struct {
	s d.SessionRepo
	u d.UserRepo
}

func NewRefreshUser(u d.UserRepo, s d.SessionRepo, i security.Issuer, v security.Verifier) *RefreshUser {
	return &RefreshUser{
		s: s,
		u: u,
	}
}

func (c *RefreshUser) Exec(ctx context.Context, refreshToken string) (d.TokenPair, error) {

}
