package usecase

import (
	"context"
	"errors"
	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
)

var (
	ErrAuthUserExists       = errors.New("user already exists")
	ErrAuthUserNotFound     = errors.New("user not found")
	ErrAuthUserInvalidCreds = errors.New("user credentials are invalid")
	ErrAuthTokenInvalid     = errors.New("invalid token")
	ErrAuthTokenExpired     = errors.New("token expired")
)

type (
	AuthUseCase struct {
		userRepo    UserRepo
		balanceRepo BalanceRepo
		pass        PassHasher
		tokener     Tokener
		tx          trm.Manager
	}
	UserRepo interface {
		Get(ctx context.Context, login entity.Login) (entity.User, error)
		Exists(ctx context.Context, login entity.Login) (bool, error)
		Save(ctx context.Context, user entity.User) error
	}
	PassHasher interface {
		Hash(password entity.Pass) (entity.PassHash, error)
		Compare(password entity.Pass, hash entity.PassHash) bool
	}
	Tokener interface {
		Create(id entity.UserID) (entity.Token, error)
		GetUserID(token entity.Token) (entity.UserID, error)
	}
)

func NewAuthUseCase(
	userRepo UserRepo,
	balanceRepo BalanceRepo,
	hasher PassHasher,
	tokener Tokener,
	tx trm.Manager,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:    userRepo,
		balanceRepo: balanceRepo,
		pass:        hasher,
		tokener:     tokener,
		tx:          tx,
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

	user, err := entity.NewUser(entity.HashCreds{
		Login:    creds.Login,
		PassHash: passHash,
	})
	if err != nil {
		return token, err
	}

	balance := entity.NewBalance(user.ID)

	err = uc.tx.Do(ctx, func(ctx context.Context) error {
		if err = uc.userRepo.Save(ctx, user); err != nil {
			return err
		}
		return uc.balanceRepo.Save(ctx, balance)
	})
	if err != nil {
		return token, err
	}

	t, err := uc.tokener.Create(user.ID)
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
