package entity

import (
	"fmt"
	"time"

	"github.com/dlomanov/go-diploma-tpl/internal/entity/apperrors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	OrderTypeIncome  OrderType = "INCOME"
	OrderTypeOutcome OrderType = "OUTCOME"
)

const (
	// OrderStatusNew — заказ загружен в систему, но не попал в обработку.
	OrderStatusNew OrderStatus = "NEW"

	// OrderStatusProcessing — вознаграждение за заказ рассчитывается
	OrderStatusProcessing OrderStatus = "PROCESSING"

	// OrderStatusInvalid — система расчёта вознаграждений отказала в расчёте
	OrderStatusInvalid OrderStatus = "INVALID"

	// OrderStatusProcessed — данные по заказу проверены и информация о расчёте успешно получена.
	OrderStatusProcessed OrderStatus = "PROCESSED"
)

var (
	ErrOrderZeroOrNegativeAmount = apperrors.NewInvalid("order amount should be greater than zero")
	ErrOrderTypeInvalid          = apperrors.NewInvalid("invalid order type")
	ErrOrderStatusInvalid        = apperrors.NewInvalid("invalid order status")
	ErrOrderStatusFinal          = fmt.Errorf("%w: attempted update on order in final status", ErrOrderStatusInvalid)
)

type (
	OrderNumber  string
	OrderID      uuid.UUID
	OrderStatus  string
	OrderType    string
	OrderAccrual struct {
		Amount decimal.Decimal
		Status OrderStatus
	}
	Order struct {
		ID        OrderID
		Number    OrderNumber
		Type      OrderType
		Status    OrderStatus
		Amount    decimal.Decimal
		UserID    UserID
		CreatedAt time.Time
		UpdatedAt time.Time
	}
)

func NewIncomeOrder(
	number OrderNumber,
	userID UserID,
) (*Order, error) {
	orderID, err := NewOrderID()
	if err != nil {
		return nil, err
	}

	now := utcNow()
	return &Order{
		ID:        orderID,
		Number:    number,
		Type:      OrderTypeIncome,
		Status:    OrderStatusNew,
		Amount:    decimal.Zero,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func NewOutcomeOrder(
	number OrderNumber,
	amount decimal.Decimal,
	userID UserID,
) (*Order, error) {
	orderID, err := NewOrderID()
	if err != nil {
		return nil, err
	}

	if amount.IsNegative() || amount.IsZero() {
		return nil, ErrOrderZeroOrNegativeAmount
	}

	now := utcNow()
	return &Order{
		ID:        orderID,
		Number:    number,
		Type:      OrderTypeOutcome,
		Status:    OrderStatusProcessed,
		Amount:    amount,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (o *Order) Update(accrual OrderAccrual) (err error) {
	switch {
	case o.Status.Final():
		return ErrOrderStatusFinal
	case accrual.Status == OrderStatusProcessed:
		o.Amount = accrual.Amount
	}

	o.Status = accrual.Status
	o.UpdatedAt = utcNow()
	return nil
}

func NewOrderID() (OrderID, error) {
	id, err := uuid.NewUUID()
	return OrderID(id), err
}

func (t OrderType) Valid() bool {
	return t == OrderTypeIncome || t == OrderTypeOutcome
}

func (s OrderStatus) Final() bool {
	return s == OrderStatusInvalid || s == OrderStatusProcessed
}

func (s OrderStatus) Valid() bool {
	return s == OrderStatusNew ||
		s == OrderStatusProcessing ||
		s == OrderStatusInvalid ||
		s == OrderStatusProcessed
}

func (id OrderID) String() string {
	return uuid.UUID(id).String()
}
