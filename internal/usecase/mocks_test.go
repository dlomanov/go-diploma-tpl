package usecase_test

import (
	"context"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/go-errors/errors"
	"github.com/google/uuid"
	"sync"
)

var _ usecase.UserRepo = (*MockUserRepo)(nil)

type MockUserRepo struct {
	mu      sync.RWMutex
	storage map[entity.Login]entity.User
}

func newMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		mu:      sync.RWMutex{},
		storage: make(map[entity.Login]entity.User),
	}
}

func (u *MockUserRepo) Exists(_ context.Context, login entity.Login) (bool, error) {
	_, ok := u.get(login)
	return ok, nil
}

func (u *MockUserRepo) Get(_ context.Context, login entity.Login) (entity.User, error) {
	user, ok := u.get(login)
	if !ok {
		return entity.User{}, errors.New(usecase.ErrAuthUserNotFound)
	}

	return user, nil
}

func (u *MockUserRepo) Create(_ context.Context, creds entity.HashCreds) (entity.UserID, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return entity.UserID{}, errors.New(err)
	}

	if _, ok := u.get(creds.Login); ok {
		return entity.UserID{}, errors.New(usecase.ErrAuthUserExists)
	}

	user := entity.User{
		ID:        entity.UserID(id),
		HashCreds: creds,
	}

	u.mu.Lock()
	defer u.mu.Unlock()
	if _, ok := u.storage[creds.Login]; ok {
		return entity.UserID{}, errors.New(usecase.ErrAuthUserExists)
	}
	u.storage[creds.Login] = user

	return user.ID, nil
}

func (u *MockUserRepo) get(login entity.Login) (entity.User, bool) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	user, ok := u.storage[login]
	return user, ok
}
