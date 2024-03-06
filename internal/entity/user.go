package entity

import (
	"github.com/google/uuid"
	"time"
)

type (
	Creds struct {
		Login Login
		Pass  Pass
	}
	HashCreds struct {
		Login    Login
		PassHash PassHash
	}
	UserToken struct {
		ID    UserID
		Token Token
	}
	Login    string
	Pass     string
	PassHash string
	Token    string
	UserID   uuid.UUID
	User     struct {
		ID UserID
		HashCreds
		CreatedAt time.Time
		UpdatedAt time.Time
	}
)

func NewUser(creds HashCreds) (User, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return User{}, err
	}

	now := utcNow()
	return User{
		ID:        UserID(id),
		HashCreds: creds,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (c Creds) Valid() bool {
	return c.Login != "" && c.Pass != ""
}

func (t Token) Valid() bool {
	return t != ""
}
