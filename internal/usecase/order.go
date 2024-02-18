package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
)

const (
	OrderEventCreated OrderEvent = "order_event_created"
)

var (
	ErrOrderExists        = errors.New("order exists")
	ErrOrderNotFound      = errors.New("order not found")
	ErrOrderStatusInvalid = entity.ErrOrderStatusInvalid
	ErrOrderTypeInvalid   = entity.ErrOrderTypeInvalid
	ErrOrderNumberInvalid = errors.New("invalid order number")
	// ErrBalanceExists
)

type (
	OrderUseCase struct {
		orderRepo   OrderRepo
		balanceRepo BalanceRepo
		validator   OrderValidator
		tx          trm.Manager
		publisher   OrderPublisher
	}
	OrderRepo interface {
		Get(ctx context.Context, id entity.OrderID) (entity.Order, error)
		GetAll(ctx context.Context, filter *OrderFilter) ([]entity.Order, error)
		Save(ctx context.Context, order entity.Order) error
		Update(ctx context.Context, order entity.Order) error
	}
	OrderFilter struct {
		Type   *entity.OrderType
		UserID *entity.UserID
	}
	OrderValidator interface {
		ValidateNumber(number entity.OrderNumber) bool
	}
	OrderPublisher interface {
		Publish(ctx context.Context, event OrderEvent, order entity.Order) error
	}
	OrderEvent entity.Event
)

func NewOrderUseCase(
	orderRepo OrderRepo,
	balanceRepo BalanceRepo,
	validator OrderValidator,
	publisher OrderPublisher,
	tx trm.Manager,
) *OrderUseCase {
	return &OrderUseCase{
		orderRepo:   orderRepo,
		balanceRepo: balanceRepo,
		validator:   validator,
		publisher:   publisher,
		tx:          tx,
	}
}

func (uc *OrderUseCase) Save(
	ctx context.Context,
	number entity.OrderNumber,
	userID entity.UserID,
) (entity.Order, error) {
	if !uc.validator.ValidateNumber(number) {
		return entity.Order{}, ErrOrderNumberInvalid
	}

	order, err := entity.NewIncomeOrder(number, userID)
	if err != nil {
		return entity.Order{}, err
	}

	err = uc.tx.Do(ctx, func(ctx context.Context) error {
		if err = uc.orderRepo.Save(ctx, *order); err != nil {
			return err
		}
		return uc.publisher.Publish(ctx, OrderEventCreated, *order)
	})
	if err != nil {
		return entity.Order{}, err
	}

	return *order, nil
}

func (uc *OrderUseCase) Update(
	ctx context.Context,
	orderID entity.OrderID,
	status entity.OrderStatus,
	accrual entity.Amount,
) error {
	if !status.Valid() {
		return fmt.Errorf("%w unkown status %s", ErrOrderStatusInvalid, status)
	}

	order, err := uc.orderRepo.Get(ctx, orderID)
	if err != nil {
		return err
	}

	if err = order.Update(status, accrual); err != nil {
		return err
	}

	es := order.Events
	if es.Empty() {
		return nil
	}

	switch {
	case es.Contains(entity.OrderEventUpdated):
		return uc.orderRepo.Update(ctx, order)
	case es.Contains(entity.OrderEventCompleted):
		return uc.tx.Do(ctx, func(ctx context.Context) error {
			balance, err := uc.balanceRepo.Get(ctx, order.UserID)
			if err != nil {
				return err
			}
			if err = balance.Update(order); err != nil {
				return err
			}
			if err = uc.orderRepo.Update(ctx, order); err != nil {
				return err
			}
			return uc.balanceRepo.Update(ctx, balance)
		})
	default:
		return fmt.Errorf("unexpected order event %v", es)
	}
}

func (uc *OrderUseCase) GetAll(
	ctx context.Context,
	userID entity.UserID,
) ([]entity.Order, error) {
	return uc.orderRepo.GetAll(ctx, &OrderFilter{UserID: &userID})
}
