package pass

import (
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"golang.org/x/crypto/bcrypt"
)

var _ usecase.PassHasher = (*Hasher)(nil)

type Hasher struct {
	cost int
}

func NewHasher(cost int) Hasher {
	return Hasher{cost: cost}
}

func (h Hasher) Hash(password entity.Pass) (entity.PassHash, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return entity.PassHash(hash), nil
}

func (h Hasher) Compare(password entity.Pass, hash entity.PassHash) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
