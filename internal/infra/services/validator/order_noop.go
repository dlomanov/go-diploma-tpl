package validator

import "github.com/dlomanov/go-diploma-tpl/internal/entity"

type (
	OrderNoopValidator struct{}
)

func NewOrderNoopValidator() OrderNoopValidator {
	return OrderNoopValidator{}
}

func (OrderNoopValidator) ValidateNumber(entity.OrderNumber) bool {
	return true
}
