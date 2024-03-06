package validator

import (
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/services/checksum"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
)

var _ usecase.OrderValidator = (*OrderValidator)(nil)

type (
	OrderValidator struct{}
)

func NewOrderValidator() OrderValidator {
	return OrderValidator{}
}

func (o OrderValidator) ValidateNumber(number entity.OrderNumber) bool {
	return checksum.ValidateLuhn([]byte(number))
}
