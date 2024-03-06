package usecase

import (
	"context"
	"errors"
	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
)

var (
	ErrBalanceExists   = errors.New("balance already exists")
	ErrBalanceNotFound = errors.New("balance not found")
)

type (
	BalanceUseCase struct {
		orderRepo      OrderRepo
		orderValidator OrderValidator
		balanceRepo    BalanceRepo
		tx             trm.Manager
	}
	BalanceRepo interface {
		Get(ctx context.Context, userID entity.UserID) (entity.Balance, error)
		Save(ctx context.Context, balance entity.Balance) error
		Update(ctx context.Context, balance entity.Balance) error
	}
)

func NewBalanceUseCase(
	orderRepo OrderRepo,
	balanceRepo BalanceRepo,
	orderValidator OrderValidator,
	tx trm.Manager,
) *BalanceUseCase {
	return &BalanceUseCase{
		orderRepo:      orderRepo,
		balanceRepo:    balanceRepo,
		orderValidator: orderValidator,
		tx:             tx,
	}
}

func (uc *BalanceUseCase) GetBalance(
	ctx context.Context,
	userID entity.UserID,
) (entity.Balance, error) {
	return uc.balanceRepo.Get(ctx, userID)
}

func (uc *BalanceUseCase) Withdraw(
	ctx context.Context,
	number entity.OrderNumber,
	amount entity.Amount,
	userID entity.UserID,
) error {
	if !uc.orderValidator.ValidateNumber(number) {
		return ErrOrderNumberInvalid
	}

	outcomeOrder, err := entity.NewOutcomeOrder(number, amount, userID)
	if err != nil {
		return err
	}
	order := *outcomeOrder

	return uc.tx.Do(ctx, func(ctx context.Context) error {
		balance, err := uc.balanceRepo.Get(ctx, userID)
		if err != nil {
			return nil
		}
		if err = balance.Update(order); err != nil {
			return err
		}
		if err = uc.orderRepo.Create(ctx, order); err != nil {
			return err
		}
		return uc.balanceRepo.Update(ctx, balance)
	})
}

func (uc *BalanceUseCase) GetWithdrawals(
	ctx context.Context,
	userID entity.UserID,
) ([]entity.Order, error) {

	t := entity.OrderTypeOutcome
	return uc.orderRepo.GetAll(ctx, &OrderFilter{
		Type:   &t,
		UserID: &userID,
	})
}
