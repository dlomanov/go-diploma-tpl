package entity

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
)

const (
	OrderEventUpdated   Event = "order_event_status_updated"
	OrderEventCompleted Event = "order_event_accrual_updated"
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
	ErrOrderNegativeAmount = errors.New("order amount should be positive")
	ErrOrderTypeInvalid    = errors.New("invalid order type")
	ErrOrderStatusInvalid  = errors.New("invalid order status")
	ErrOrderStatusFinal    = fmt.Errorf("%w: attempted update on order in final status", ErrOrderStatusInvalid)
)

type (
	OrderNumber string
	OrderID     uuid.UUID
	OrderStatus string
	OrderType   string
	Order       struct {
		ID        OrderID
		Number    OrderNumber
		Type      OrderType
		Status    OrderStatus
		Amount    Amount
		UserID    UserID
		CreatedAt time.Time
		UpdatedAt time.Time
		Events    Events
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
		Amount:    ZeroAmount(),
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
		Events:    NewEvents(),
	}, nil
}

func NewOutcomeOrder(
	number OrderNumber,
	amount Amount,
	userID UserID,
) (*Order, error) {
	orderID, err := NewOrderID()
	if err != nil {
		return nil, err
	}

	if amount.IsNegative() {
		return nil, ErrOrderNegativeAmount
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
		Events:    NewEvents(),
	}, nil
}

func (o *Order) Update(
	status OrderStatus,
	accrual Amount,
) (err error) {
	if o.Status.Final() {
		return ErrOrderStatusFinal
	}

	if o.Status == status {
		return nil
	}

	switch status {
	case o.Status:
		return nil
	case OrderStatusProcessed:
		o.Amount = accrual
		o.Events = append(o.Events, OrderEventCompleted)
	default:
		o.Events = append(o.Events, OrderEventUpdated)
	}
	o.Status = status
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
