package token

import (
	"fmt"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
)

var _ usecase.Tokener = (*JWTTokener)(nil)

type JWTTokener struct {
	secret  []byte
	expires time.Duration
}

func NewJWT(secret []byte, expires time.Duration) JWTTokener {
	return JWTTokener{
		secret:  secret,
		expires: expires,
	}
}

var method = jwt.SigningMethodHS256

type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
}

func (t JWTTokener) Create(id entity.UserID) (entity.Token, error) {
	token := jwt.NewWithClaims(method, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.expires)),
		},
		UserID: uuid.UUID(id).String(),
	})

	tokenString, err := token.SignedString(t.secret)
	if err != nil {
		return "", err
	}

	return entity.Token(tokenString), nil
}

func (t JWTTokener) GetUserID(token entity.Token) (entity.UserID, error) {
	c := new(Claims)

	value, err := jwt.ParseWithClaims(string(token), c, func(token *jwt.Token) (any, error) {
		if m, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || m.Name != method.Name {
			return nil, fmt.Errorf("%w unexpected signing method: %v", usecase.ErrTokenInvalid, token.Header["alg"])
		}
		return t.secret, nil
	})
	if err != nil {
		return entity.UserID{}, err
	}
	if !value.Valid {
		return entity.UserID{}, usecase.ErrTokenInvalid
	}

	expires := c.ExpiresAt.UTC()
	now := time.Now().UTC()
	if expires.Compare(now) == -1 {
		return entity.UserID{}, usecase.ErrTokenExpired
	}

	id, err := uuid.Parse(c.UserID)
	if err != nil {
		return entity.UserID{}, err
	}

	return entity.UserID(id), nil
}
