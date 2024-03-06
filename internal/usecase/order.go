package usecase

import (
	"context"

	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/entity/apperrors"
	"github.com/google/uuid"
)

var (
	ErrOrderExists             = apperrors.NewInvalid("order exists")
	ErrOrderSavedByAnotherUser = apperrors.NewInvalid("order saved by another user")
	ErrOrderNotFound           = apperrors.NewNotFound("order not found")
	ErrOrderStatusInvalid      = entity.ErrOrderStatusInvalid
	ErrOrderTypeInvalid        = entity.ErrOrderTypeInvalid
	ErrOrderNumberInvalid      = apperrors.NewInvalid("invalid order number")
)

type (
	OrderUseCase struct {
		orderRepo     OrderRepo
		balanceRepo   BalanceRepo
		validator     OrderValidator
		accrualAPI    OrderAccrualAPI
		backgroundJob BackgroundQueue
		tx            trm.Manager
	}
	OrderRepo interface {
		Get(ctx context.Context, id entity.OrderID) (entity.Order, error)
		GetAll(ctx context.Context, filter *OrderFilter) ([]entity.Order, error)
		Create(ctx context.Context, order entity.Order) error
		Update(ctx context.Context, order entity.Order) error
	}
	OrderFilter struct {
		Number *entity.OrderNumber
		Type   *entity.OrderType
		UserID *entity.UserID
	}
	OrderValidator interface {
		ValidateNumber(number entity.OrderNumber) bool
	}
	OrderAccrualAPI interface {
		Get(ctx context.Context, number entity.OrderNumber) (entity.OrderAccrual, error)
	}
	BackgroundQueue interface {
		Enqueue(ctx context.Context, entityID uuid.UUID, jobType entity.JobType) error
	}
)

func NewOrderUseCase(
	orderRepo OrderRepo,
	balanceRepo BalanceRepo,
	validator OrderValidator,
	accrualAPI OrderAccrualAPI,
	backgroundQueue BackgroundQueue,
	tx trm.Manager,
) *OrderUseCase {
	return &OrderUseCase{
		orderRepo:     orderRepo,
		balanceRepo:   balanceRepo,
		validator:     validator,
		accrualAPI:    accrualAPI,
		backgroundJob: backgroundQueue,
		tx:            tx,
	}
}

func (uc *OrderUseCase) Create(
	ctx context.Context,
	number entity.OrderNumber,
	userID entity.UserID,
) error {
	if !uc.validator.ValidateNumber(number) {
		return ErrOrderNumberInvalid
	}
	orders, err := uc.orderRepo.GetAll(ctx, &OrderFilter{Number: &number})
	switch {
	case err != nil:
		return err
	case len(orders) == 0:
	case orders[0].UserID == userID:
		return ErrOrderExists
	default:
		return ErrOrderSavedByAnotherUser
	}

	order, err := entity.NewIncomeOrder(number, userID)
	if err != nil {
		return err
	}

	err = uc.tx.Do(ctx, func(ctx context.Context) error {
		if err = uc.orderRepo.Create(ctx, *order); err != nil {
			return err
		}
		return uc.backgroundJob.Enqueue(ctx, uuid.UUID(order.ID), entity.JobTypePollAccrual)
	})
	if err != nil {
		return err
	}

	return nil
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
