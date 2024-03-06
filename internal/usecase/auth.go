package usecase

import (
	"context"
	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/entity/apperrors"
	"go.uber.org/zap"
)

var (
	ErrAuthUserExists       = apperrors.NewInvalid("user already exists")
	ErrAuthUserInvalidCreds = apperrors.NewInvalid("user credentials are invalid")
	ErrAuthTokenInvalid     = apperrors.NewInvalid("invalid token")
	ErrAuthTokenExpired     = apperrors.NewInvalid("token expired")
	ErrAuthUserNotFound     = apperrors.NewNotFound("user not found")
)

type (
	AuthUseCase struct {
		logger      *zap.Logger
		userRepo    UserRepo
		balanceRepo BalanceRepo
		pass        PassHasher
		tokener     Tokener
		tx          trm.Manager
	}
	UserRepo interface {
		Get(ctx context.Context, login entity.Login) (entity.User, error)
		Exists(ctx context.Context, login entity.Login) (bool, error)
		Create(ctx context.Context, user entity.User) error
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
	logger *zap.Logger,
	userRepo UserRepo,
	balanceRepo BalanceRepo,
	hasher PassHasher,
	tokener Tokener,
	tx trm.Manager,
) *AuthUseCase {
	return &AuthUseCase{
		logger:      logger,
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
	if !creds.Valid() {
		return token, ErrAuthUserInvalidCreds
	}
	exists, err := uc.userRepo.Exists(ctx, creds.Login)
	switch {
	case err != nil:
		return token, err
	case exists:
		return token, ErrAuthUserExists
	}

	passHash, err := uc.pass.Hash(creds.Pass)
	if err != nil {
		uc.logger.Debug("failed to calculate pass hash", zap.Error(err))
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
		err = uc.userRepo.Create(ctx, user)
		if err != nil {
			return err
		}
		return uc.balanceRepo.Save(ctx, balance)
	})
	if err != nil {
		return token, err
	}
	t, err := uc.tokener.Create(user.ID)
	if err != nil {
		uc.logger.Debug("failed to create token", zap.Error(err))
		return token, err
	}

	return t, nil
}

func (uc *AuthUseCase) Login(
	ctx context.Context,
	creds entity.Creds,
) (token entity.Token, err error) {
	if !creds.Valid() {
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

func (uc *AuthUseCase) GetUserID(token entity.Token) (entity.UserID, error) {
	if !token.Valid() {
		return entity.UserID{}, ErrAuthTokenInvalid
	}

	return uc.tokener.GetUserID(token)
}
