package entity

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrBalanceNotEnoughFounds    = errors.New("not enough founds on the balance")
	ErrBalanceUserInvalid        = errors.New("invalid user")
	ErrBalanceOrderStatusInvalid = ErrOrderStatusInvalid
	ErrBalanceOrderAmountInvalid = ErrOrderNegativeAmount
)

type Balance struct {
	UserID    UserID
	Current   Amount
	Withdrawn Amount
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewBalance(userID UserID) Balance {
	now := utcNow()
	return Balance{
		UserID:    userID,
		Current:   ZeroAmount(),
		Withdrawn: ZeroAmount(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (b *Balance) Update(order Order) error {
	if b.UserID != order.UserID {
		return ErrBalanceUserInvalid
	}
	if order.Status != OrderStatusProcessed {
		return ErrBalanceOrderStatusInvalid
	}
	if order.Amount.IsNegative() {
		return ErrBalanceOrderAmountInvalid
	}

	switch order.Type {
	case OrderTypeIncome:
		return b.put(order.Amount)
	case OrderTypeOutcome:
		return b.withdraw(order.Amount)
	default:
		return fmt.Errorf("%w: invalid type %s", ErrOrderTypeInvalid, order.Type)
	}
}

func (b *Balance) put(amount Amount) error {
	b.Current = b.Current.Add(amount)
	return nil
}

func (b *Balance) withdraw(amount Amount) error {
	if b.Current.Sub(amount).IsNegative() {
		return ErrBalanceNotEnoughFounds
	}
	b.Current = b.Current.Sub(amount)
	b.Withdrawn = b.Withdrawn.Add(amount)
	return nil
}