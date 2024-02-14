package usecase

import (
	"context"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/go-errors/errors"
)

var (
	ErrAuthUserExists = errors.Errorf("user already exists")
)

type AuthUseCase struct {
	repo    UserRepo
	pass    PassHasher
	tokener Tokener
}

func NewAuth(
	repo UserRepo,
	hasher PassHasher,
	tokener Tokener,
) *AuthUseCase {
	return &AuthUseCase{
		repo:    repo,
		pass:    hasher,
		tokener: tokener,
	}
}

func (uc *AuthUseCase) Register(
	ctx context.Context,
	creds entity.Creds,
) (token entity.Token, err error) {
	exists, err := uc.repo.Exists(ctx, creds.Login)
	if err != nil {
		return token, err
	}
	if exists {
		return token, errors.New(ErrAuthUserExists)
	}

	passHash, err := uc.pass.Hash(creds.Pass)
	if err != nil {
		return token, err
	}

	userID, err := uc.repo.Create(ctx, entity.HashCreds{
		Login:    creds.Login,
		PassHash: passHash,
	})
	if err != nil {
		return token, err
	}

	t, err := uc.tokener.Create(ctx, userID)
	if err != nil {
		return token, err
	}

	return t, nil
}
