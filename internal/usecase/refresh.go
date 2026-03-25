package usecase

import (
	"context"
	"time"

	d "github.com/tasker-iniutin/auth-service/internal/domain"
	sec "github.com/tasker-iniutin/common/authsecurity"
	"github.com/google/uuid"
)

type RefreshUser struct {
	s d.SessionRepo
	i sec.Issuer
}

func NewRefreshUser(s d.SessionRepo, i sec.Issuer) *RefreshUser {
	return &RefreshUser{
		s: s,
		i: i,
	}
}

func (c *RefreshUser) Exec(ctx context.Context, refreshToken string) (d.TokenPair, error) {
	if err := ctx.Err(); err != nil {
		return d.TokenPair{}, err
	}
	if refreshToken == "" {
		return d.TokenPair{}, d.ErrValidation
	}

	// 1) hash incoming refresh token
	oldHash := sec.RefreshHash(refreshToken)

	// 2) load existing session
	sess, err := c.s.GetRefresh(ctx, oldHash)
	if err != nil {
		return d.TokenPair{}, err
	}

	// 3) check expiry
	now := time.Now()
	if !sess.ExpiresAt.After(now) {
		_ = c.s.RevokeRefresh(ctx, oldHash)
		return d.TokenPair{}, d.ErrSessionExpired
	}

	// 4) issue new access token
	accessToken, accessExp, err := c.i.NewAccess(uint64(sess.UserID))
	if err != nil {
		return d.TokenPair{}, err
	}

	// 5) issue new refresh token
	newRefreshToken, newRefreshHash, err := c.i.NewRefresh()
	if err != nil {
		return d.TokenPair{}, err
	}

	pair := d.TokenPair{
		AccessToken:   accessToken,
		RefreshToken:  newRefreshToken,
		AccessExpires: accessExp,
	}

	// 6) store new refresh session
	newSess := d.RefreshSession{
		ID:        d.SessionID(uuid.NewString()),
		UserID:    sess.UserID,
		TokenHash: append([]byte(nil), newRefreshHash...),
		CreatedAt: now,
		ExpiresAt: now.Add(14 * 24 * time.Hour),
	}

	if err := c.s.CreateRefresh(ctx, newSess); err != nil {
		return d.TokenPair{}, err
	}

	// 7) revoke old refresh session
	_ = c.s.RevokeRefresh(ctx, oldHash)

	return pair, nil
}
