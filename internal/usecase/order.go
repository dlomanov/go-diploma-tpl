package usecase

import (
	"context"
	"errors"
	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/google/uuid"
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
		orderRepo     OrderRepo
		balanceRepo   BalanceRepo
		validator     OrderValidator
		accrualAPI    OrderAccrualAPI
		backgroundJob BackgroundJob
		tx            trm.Manager
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
	OrderAccrualAPI interface {
		Get(ctx context.Context, number entity.OrderNumber) (entity.OrderAccrual, error)
	}
	BackgroundJob interface {
		Enqueue(entityID uuid.UUID, jobType entity.JobType) error
	}
)

func NewOrderUseCase(
	orderRepo OrderRepo,
	balanceRepo BalanceRepo,
	validator OrderValidator,
	tx trm.Manager,
) *OrderUseCase {
	return &OrderUseCase{
		orderRepo:   orderRepo,
		balanceRepo: balanceRepo,
		validator:   validator,
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
		return uc.backgroundJob.Enqueue(uuid.UUID(order.ID), entity.JobTypePollAccrual)
	})
	if err != nil {
		return entity.Order{}, err
	}

	return *order, nil
}

func (uc *OrderUseCase) UpdateAccrual(
	ctx context.Context,
	orderID entity.OrderID,
) (entity.OrderEvent, error) {
	order, err := uc.orderRepo.Get(ctx, orderID)
	if err != nil {
		return "", err
	}
	accrual, err := uc.accrualAPI.Get(ctx, order.Number)
	if err != nil {
		return "", err
	}

	if err = order.Update(accrual); err != nil {
		return "", err
	}

	es := order.Events
	switch {
	case es.Contains(entity.Event(entity.OrderEventUpdated)):
		if err = uc.orderRepo.Update(ctx, order); err != nil {
			return "", err
		}
		return entity.OrderEventUpdated, nil
	case es.Contains(entity.Event(entity.OrderEventCompleted)):
		err = uc.completeOrder(ctx, order)
		if err != nil {
			return "", err
		}
		return entity.OrderEventCompleted, nil
	default:
		return "", entity.ErrOrderEventInvalid
	}
}

func (uc *OrderUseCase) GetAll(
	ctx context.Context,
	userID entity.UserID,
) ([]entity.Order, error) {
	return uc.orderRepo.GetAll(ctx, &OrderFilter{UserID: &userID})
}

func (uc *OrderUseCase) completeOrder(
	ctx context.Context,
	order entity.Order,
) error {
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
		if err = uc.balanceRepo.Update(ctx, balance); err != nil {
			return err
		}
		return nil
	})

}
