package mem

import (
	"context"
	"sync"

	d "todo/auth-service/internal/domain"
)

type userRepoImpl struct {
	mu sync.RWMutex

	userByID map[d.UserID]d.User
	credById map[d.UserID]d.Credentials

	idByEmail map[string]d.UserID
	idByLogin map[string]d.UserID

	counter uint64
}

func NewUserRepo() *userRepoImpl {
	return &userRepoImpl{
		userByID:  make(map[d.UserID]d.User),
		credById:  make(map[d.UserID]d.Credentials),
		idByEmail: make(map[string]d.UserID),
		idByLogin: make(map[string]d.UserID),
		counter:   1,
	}
}

func (r *userRepoImpl) Create(ctx context.Context, u d.UserCreateRequest, password d.PasswordHash) (d.User, error) {
	if err := ctx.Err(); err != nil {
		return d.User{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if u.Email == "" || u.Login == "" {
		return d.User{}, d.ErrValidation
	}

	if _, ok := r.idByEmail[u.Email]; ok {
		return d.User{}, d.ErrConflict
	}
	if _, ok := r.idByLogin[u.Login]; ok {
		return d.User{}, d.ErrConflict
	}

	if r.counter == 0 {
		return d.User{}, d.ErrRepoIsFull
	}

	id := d.UserID(r.counter)
	r.counter++

	nU := d.User{
		ID:    id,
		Email: u.Email,
		Login: u.Login,
	}
	nCred := d.Credentials{
		UserID:       id,
		PasswordHash: password,
	}

	r.userByID[id] = nU
	r.credById[id] = nCred
	r.idByEmail[u.Email] = id
	r.idByLogin[u.Login] = id

	return nU, nil
}

func (r *userRepoImpl) GetByEmail(ctx context.Context, email string) (d.User, error) {
	if err := ctx.Err(); err != nil {
		return d.User{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.idByEmail[email]
	if !ok {
		return d.User{}, d.ErrNotFound
	}
	u, ok := r.userByID[id]
	if !ok {
		return d.User{}, d.ErrNotFound
	}
	return u, nil
}

func (r *userRepoImpl) GetByLogin(ctx context.Context, login string) (d.User, error) {
	if err := ctx.Err(); err != nil {
		return d.User{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.idByLogin[login]
	if !ok {
		return d.User{}, d.ErrNotFound
	}
	u, ok := r.userByID[id]
	if !ok {
		return d.User{}, d.ErrNotFound
	}
	return u, nil
}
