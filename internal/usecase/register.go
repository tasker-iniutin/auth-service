package usecase

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	d "github.com/tasker-iniutin/auth-service/internal/domain"
	sec "github.com/tasker-iniutin/common/authsecurity"
)

type RegisterUser struct {
	s d.SessionRepo
	u d.UserRepo
	i sec.Issuer
}

func NewRegisterUser(s d.SessionRepo, u d.UserRepo, i sec.Issuer) *RegisterUser {
	return &RegisterUser{
		s: s,
		u: u,
		i: i,
	}
}

func (c *RegisterUser) Exec(ctx context.Context, r d.UserCreateRequest, password string) (d.User, d.TokenPair, error) {
	if err := ctx.Err(); err != nil {
		return d.User{}, d.TokenPair{}, err
	}
	if r.Email == "" || r.Login == "" || password == "" {
		return d.User{}, d.TokenPair{}, d.ErrValidation
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return d.User{}, d.TokenPair{}, err
	}

	user, err := c.u.Create(ctx, r, d.PasswordHash{
		Algo: "bcrypt",
		Hash: hash,
	})
	if err != nil {
		return d.User{}, d.TokenPair{}, err
	}

	accessToken, accessExp, err := c.i.NewAccess(uint64(user.ID))
	if err != nil {
		return d.User{}, d.TokenPair{}, err
	}

	refreshToken, refreshHash, err := c.i.NewRefresh()
	if err != nil {
		return d.User{}, d.TokenPair{}, err
	}

	pair := d.TokenPair{
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		AccessExpires: accessExp,
	}

	now := time.Now()
	sess := d.RefreshSession{
		ID:        d.SessionID(uint64(now.UnixNano())),
		UserID:    user.ID,
		TokenHash: append([]byte(nil), refreshHash...),
		CreatedAt: now,
		ExpiresAt: now.Add(14 * 24 * time.Hour),
	}

	if err := c.s.CreateRefresh(ctx, sess); err != nil {
		return d.User{}, d.TokenPair{}, err
	}

	return user, pair, nil
}
