package entity

import (
	"fmt"
	"time"

	"github.com/dlomanov/go-diploma-tpl/internal/entity/apperrors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	OrderEventUpdated   OrderEvent = "order_event_status_updated"
	OrderEventCompleted OrderEvent = "order_event_accrual_updated"
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
	ErrOrderNegativeAmount = apperrors.NewInvalid("order amount should be positive")
	ErrOrderTypeInvalid    = apperrors.NewInvalid("invalid order type")
	ErrOrderStatusInvalid  = apperrors.NewInvalid("invalid order status")
	ErrOrderEventInvalid   = apperrors.NewInvalid("invalid order event")
	ErrOrderStatusFinal    = fmt.Errorf("%w: attempted update on order in final status", ErrOrderStatusInvalid)
)

type (
	OrderNumber  string
	OrderID      uuid.UUID
	OrderStatus  string
	OrderType    string
	OrderEvent   Event
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
		Amount:    decimal.Zero,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
		Events:    NewEvents(),
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

func (o *Order) Update(accrual OrderAccrual) (err error) {
	if o.Status.Final() {
		return ErrOrderStatusFinal
	}

	switch accrual.Status {
	case OrderStatusProcessed:
		o.Amount = accrual.Amount
		o.Events = append(o.Events, Event(OrderEventCompleted))
	case OrderStatusInvalid:
		o.Events = append(o.Events, Event(OrderEventCompleted))
	default:
		o.Events = append(o.Events, Event(OrderEventUpdated))
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
