package usecase

import (
	"context"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
)

type UserRepo interface {
	Exists(ctx context.Context, login entity.Login) (bool, error)
	Create(ctx context.Context, creds entity.HashCreds) (entity.UserID, error)
}

type PassHasher interface {
	Hash(password entity.Pass) (entity.PassHash, error)
	Compare(password entity.Pass, hash entity.PassHash) bool
}

type Tokener interface {
	Create(ctx context.Context, id entity.UserID) (entity.Token, error)
	GetUserID(ctx context.Context, token entity.Token) (entity.UserID, error)
}
