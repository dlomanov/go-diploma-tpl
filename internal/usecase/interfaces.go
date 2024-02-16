package usecase

import (
	"context"
	"errors"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
)

var (
	ErrTokenInvalid = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

type UserRepo interface {
	Exists(ctx context.Context, login entity.Login) (bool, error)
	Get(ctx context.Context, login entity.Login) (entity.User, error)
	Create(ctx context.Context, creds entity.HashCreds) (entity.UserID, error)
}

type PassHasher interface {
	Hash(password entity.Pass) (entity.PassHash, error)
	Compare(password entity.Pass, hash entity.PassHash) bool
}

type Tokener interface {
	Create(id entity.UserID) (entity.Token, error)
	GetUserID(token entity.Token) (entity.UserID, error)
}
