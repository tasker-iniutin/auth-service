package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	d "github.com/tasker-iniutin/auth-service/internal/domain"
	"github.com/tasker-iniutin/auth-service/internal/store/mem"
	sec "github.com/tasker-iniutin/common/authsecurity"
	"golang.org/x/crypto/bcrypt"
)

type fakeIssuer struct {
	accessToken  string
	refreshToken string
	refreshHash  []byte
	exp          time.Time
	accessErr    error
	refreshErr   error
}

func (f fakeIssuer) NewAccess(userID uint64) (string, time.Time, error) {
	if f.accessErr != nil {
		return "", time.Time{}, f.accessErr
	}
	return f.accessToken, f.exp, nil
}

func (f fakeIssuer) NewRefresh() (string, []byte, error) {
	if f.refreshErr != nil {
		return "", nil, f.refreshErr
	}
	return f.refreshToken, append([]byte(nil), f.refreshHash...), nil
}

type fakeSessionRepo struct {
	sessionsByHash map[string]d.RefreshSession
	created        []d.RefreshSession
	revoked        [][]byte
	createErr      error
	getErr         error
	revokeErr      error
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{sessionsByHash: make(map[string]d.RefreshSession)}
}

func (r *fakeSessionRepo) CreateRefresh(ctx context.Context, s d.RefreshSession) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.created = append(r.created, s)
	r.sessionsByHash[string(s.TokenHash)] = s
	return nil
}

func (r *fakeSessionRepo) GetRefresh(ctx context.Context, tokenHash []byte) (d.RefreshSession, error) {
	if r.getErr != nil {
		return d.RefreshSession{}, r.getErr
	}
	s, ok := r.sessionsByHash[string(tokenHash)]
	if !ok {
		return d.RefreshSession{}, d.ErrNotFound
	}
	return s, nil
}

func (r *fakeSessionRepo) RevokeRefresh(ctx context.Context, tokenHash []byte) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}
	r.revoked = append(r.revoked, append([]byte(nil), tokenHash...))
	delete(r.sessionsByHash, string(tokenHash))
	return nil
}

func TestRegisterUserExecSuccess(t *testing.T) {
	userRepo := mem.NewUserRepo()
	sessionRepo := newFakeSessionRepo()
	issuer := fakeIssuer{
		accessToken:  "access-token",
		refreshToken: "refresh-token",
		refreshHash:  []byte("refresh-hash"),
		exp:          time.Now().Add(15 * time.Minute),
	}

	uc := NewRegisterUser(sessionRepo, userRepo, issuer)

	user, pair, err := uc.Exec(context.Background(), d.UserCreateRequest{
		Email: "user@example.com",
		Login: "user",
	}, "secret")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected created user id")
	}
	if pair.AccessToken != "access-token" || pair.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected token pair: %+v", pair)
	}
	if len(sessionRepo.created) != 1 {
		t.Fatalf("expected 1 refresh session, got %d", len(sessionRepo.created))
	}
}

func TestRegisterUserExecValidation(t *testing.T) {
	uc := NewRegisterUser(newFakeSessionRepo(), mem.NewUserRepo(), fakeIssuer{})

	_, _, err := uc.Exec(context.Background(), d.UserCreateRequest{}, "")
	if !errors.Is(err, d.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestLoginUserExecSuccess(t *testing.T) {
	userRepo := mem.NewUserRepo()
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate hash: %v", err)
	}
	user, err := userRepo.Create(context.Background(), d.UserCreateRequest{
		Email: "user@example.com",
		Login: "user",
	}, d.PasswordHash{
		Algo: "bcrypt",
		Hash: hash,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	sessionRepo := newFakeSessionRepo()
	issuer := fakeIssuer{
		accessToken:  "access-token",
		refreshToken: "refresh-token",
		refreshHash:  []byte("refresh-hash"),
		exp:          time.Now().Add(15 * time.Minute),
	}

	uc := NewLoginUser(sessionRepo, userRepo, issuer)
	gotUser, pair, err := uc.Exec(context.Background(), &d.UserLoginRequest{
		Email:    "user@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	if gotUser.ID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, gotUser.ID)
	}
	if pair.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected refresh token: %q", pair.RefreshToken)
	}
	if len(sessionRepo.created) != 1 {
		t.Fatalf("expected session creation, got %d", len(sessionRepo.created))
	}
}

func TestLoginUserExecInvalidPassword(t *testing.T) {
	userRepo := mem.NewUserRepo()
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate hash: %v", err)
	}
	_, err = userRepo.Create(context.Background(), d.UserCreateRequest{
		Email: "user@example.com",
		Login: "user",
	}, d.PasswordHash{
		Algo: "bcrypt",
		Hash: hash,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	uc := NewLoginUser(newFakeSessionRepo(), userRepo, fakeIssuer{})
	_, _, err = uc.Exec(context.Background(), &d.UserLoginRequest{
		Login:    "user",
		Password: "wrong-secret",
	})
	if !errors.Is(err, d.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestRefreshUserExecSuccess(t *testing.T) {
	sessionRepo := newFakeSessionRepo()
	refreshToken := "old-refresh-token"
	oldHash := sec.RefreshHash(refreshToken)
	sessionRepo.sessionsByHash[string(oldHash)] = d.RefreshSession{
		ID:        d.SessionID("sess-1"),
		UserID:    42,
		TokenHash: append([]byte(nil), oldHash...),
		CreatedAt: time.Now().Add(-time.Hour),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	uc := NewRefreshUser(sessionRepo, fakeIssuer{
		accessToken:  "access-token",
		refreshToken: "new-refresh-token",
		refreshHash:  []byte("new-refresh-hash"),
		exp:          time.Now().Add(15 * time.Minute),
	})

	pair, err := uc.Exec(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("refresh user: %v", err)
	}
	if pair.RefreshToken != "new-refresh-token" {
		t.Fatalf("unexpected refresh token: %q", pair.RefreshToken)
	}
	if len(sessionRepo.created) != 1 {
		t.Fatalf("expected new session creation, got %d", len(sessionRepo.created))
	}
	if len(sessionRepo.revoked) != 1 {
		t.Fatalf("expected old session revoke, got %d", len(sessionRepo.revoked))
	}
}

func TestRefreshUserExecFailOpenOnRevokeError(t *testing.T) {
	sessionRepo := newFakeSessionRepo()
	refreshToken := "old-refresh-token"
	oldHash := sec.RefreshHash(refreshToken)
	sessionRepo.sessionsByHash[string(oldHash)] = d.RefreshSession{
		ID:        d.SessionID("sess-1"),
		UserID:    42,
		TokenHash: append([]byte(nil), oldHash...),
		CreatedAt: time.Now().Add(-time.Hour),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	sessionRepo.revokeErr = errors.New("revoke failed")

	uc := NewRefreshUser(sessionRepo, fakeIssuer{
		accessToken:  "access-token",
		refreshToken: "new-refresh-token",
		refreshHash:  []byte("new-refresh-hash"),
		exp:          time.Now().Add(15 * time.Minute),
	})

	pair, err := uc.Exec(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("refresh user: %v", err)
	}
	if pair.RefreshToken != "new-refresh-token" {
		t.Fatalf("unexpected refresh token: %q", pair.RefreshToken)
	}
	if len(sessionRepo.created) != 1 {
		t.Fatalf("expected new session creation, got %d", len(sessionRepo.created))
	}
	if len(sessionRepo.revoked) != 0 {
		t.Fatalf("expected no revoked sessions due to error, got %d", len(sessionRepo.revoked))
	}
}

func TestLogoutUserExecValidation(t *testing.T) {
	uc := NewLogoutUser(newFakeSessionRepo())

	err := uc.Exec(context.Background(), "")
	if !errors.Is(err, d.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestLogoutUserExecRevokesByRefreshHash(t *testing.T) {
	sessionRepo := newFakeSessionRepo()
	uc := NewLogoutUser(sessionRepo)

	refreshToken := "refresh-token"
	if err := uc.Exec(context.Background(), refreshToken); err != nil {
		t.Fatalf("logout user: %v", err)
	}

	if len(sessionRepo.revoked) != 1 {
		t.Fatalf("expected 1 revoked session, got %d", len(sessionRepo.revoked))
	}

	want := sec.RefreshHash(refreshToken)
	if string(sessionRepo.revoked[0]) != string(want) {
		t.Fatal("expected refresh token hash to be revoked")
	}
}
