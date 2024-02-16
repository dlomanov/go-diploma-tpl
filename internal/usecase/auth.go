package usecase

import (
	"context"
	"errors"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
)

var (
	ErrAuthUserExists       = errors.New("user already exists")
	ErrAuthUserNotFound     = errors.New("user not found")
	ErrAuthUserInvalidCreds = errors.New("user credentials are invalid")
)

type AuthUseCase struct {
	userRepo UserRepo
	pass     PassHasher
	tokener  Tokener
}

func NewAuth(
	repo UserRepo,
	hasher PassHasher,
	tokener Tokener,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo: repo,
		pass:     hasher,
		tokener:  tokener,
	}
}

func (uc *AuthUseCase) Register(
	ctx context.Context,
	creds entity.Creds,
) (token entity.Token, err error) {
	if creds.Login == "" || creds.Pass == "" {
		return token, ErrAuthUserInvalidCreds
	}

	exists, err := uc.userRepo.Exists(ctx, creds.Login)
	if err != nil {
		return token, err
	}
	if exists {
		return token, ErrAuthUserExists
	}

	passHash, err := uc.pass.Hash(creds.Pass)
	if err != nil {
		return token, err
	}

	userID, err := uc.userRepo.Create(ctx, entity.HashCreds{
		Login:    creds.Login,
		PassHash: passHash,
	})
	if err != nil {
		return token, err
	}

	t, err := uc.tokener.Create(userID)
	if err != nil {
		return token, err
	}

	return t, nil
}

func (uc *AuthUseCase) Login(
	ctx context.Context,
	creds entity.Creds,
) (token entity.Token, err error) {
	if creds.Login == "" || creds.Pass == "" {
		return token, ErrAuthUserInvalidCreds
	}

	user, err := uc.userRepo.Get(ctx, creds.Login)
	if err != nil {
		return token, err
	}

	if !uc.pass.Compare(creds.Pass, user.PassHash) {
		return token, ErrAuthUserInvalidCreds
	}

	t, err := uc.tokener.Create(user.ID)
	if err != nil {
		return token, err
	}

	return t, nil
}
