package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	d "github.com/tasker-iniutin/auth-service/internal/domain"
	sec "github.com/tasker-iniutin/common/authsecurity"
)

type LoginUser struct {
	s d.SessionRepo
	u d.UserRepo
	i sec.Issuer
}

func NewLoginUser(s d.SessionRepo, u d.UserRepo, i sec.Issuer) *LoginUser {
	return &LoginUser{
		s: s,
		u: u,
		i: i,
	}
}

func (c *LoginUser) Exec(ctx context.Context, l *d.UserLoginRequest) (d.User, d.TokenPair, error) {
	if err := ctx.Err(); err != nil {
		return d.User{}, d.TokenPair{}, err
	}
	if l == nil || l.Password == "" || (l.Email == "" && l.Login == "") {
		return d.User{}, d.TokenPair{}, d.ErrValidation
	}

	var (
		u   d.User
		err error
	)

	switch {
	case l.Email != "":
		u, err = c.u.GetByEmail(ctx, l.Email)
	case l.Login != "":
		u, err = c.u.GetByLogin(ctx, l.Login)
	default:
		return d.User{}, d.TokenPair{}, d.ErrValidation
	}
	if err != nil {
		return d.User{}, d.TokenPair{}, err
	}

	cred, err := c.u.GetCredentials(ctx, u.ID)
	if err != nil {
		return d.User{}, d.TokenPair{}, err
	}

	if cred.PasswordHash.Algo != "bcrypt" || len(cred.PasswordHash.Hash) == 0 {
		return d.User{}, d.TokenPair{}, d.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword(cred.PasswordHash.Hash, []byte(l.Password)); err != nil {
		return d.User{}, d.TokenPair{}, d.ErrInvalidCredentials
	}

	accessToken, accessExp, err := c.i.NewAccess(uint64(u.ID))
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
		ID:        d.SessionID(uuid.NewString()),
		UserID:    u.ID,
		TokenHash: append([]byte(nil), refreshHash...),
		CreatedAt: now,
		ExpiresAt: now.Add(14 * 24 * time.Hour),
	}

	if err := c.s.CreateRefresh(ctx, sess); err != nil {
		return d.User{}, d.TokenPair{}, err
	}

	return u, pair, nil
}
