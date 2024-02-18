package usecase_test

import (
	"context"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/algo/validator"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrderUseCase_SuccessCase(t *testing.T) {
	orderRepo := mocks.NewMockOrderRepo()
	balanceRepo := mocks.NewMockBalanceRepo()
	uc := usecase.NewOrderUseCase(
		orderRepo,
		balanceRepo,
		validator.NewOrderValidator(),
		mocks.NewMockPublisher(),
		mocks.NewMockTrm(),
	)

	ctx := context.Background()
	amount := entity.NewAmount(10)
	number := entity.OrderNumber("6200000000000005")
	id, err := uuid.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	userID := entity.UserID(id)

	order, err := uc.Save(ctx, number, userID)
	require.NoErrorf(t, err, "")
	require.Equalf(t, order.Type, entity.OrderTypeIncome, "")
	require.Equalf(t, order.Status, entity.OrderStatusNew, "")

	err = uc.Update(ctx, order.ID, entity.OrderStatusProcessing, amount)
	require.NoErrorf(t, err, "")
	order, err = orderRepo.Get(ctx, order.ID)
	require.NoErrorf(t, err, "")
	require.Equalf(t, order.Amount, entity.ZeroAmount(), "")
	require.Equalf(t, order.Status, entity.OrderStatusProcessing, "")
	require.Equalf(t, order.Type, entity.OrderTypeIncome, "")

	err = uc.Update(ctx, order.ID, entity.OrderStatusProcessed, amount)
	require.NoErrorf(t, err, "")
	order, err = orderRepo.Get(ctx, order.ID)
	require.NoErrorf(t, err, "")
	require.Equalf(t, order.Amount, amount, "")
	require.Equalf(t, order.Status, entity.OrderStatusProcessed, "")
	require.Equalf(t, order.Type, entity.OrderTypeIncome, "")

	err = uc.Update(ctx, order.ID, entity.OrderStatusInvalid, amount)
	require.Error(t, err, "")
	order, err = orderRepo.Get(ctx, order.ID)
	require.NoErrorf(t, err, "")
	require.Equalf(t, order.Amount, amount, "")
	require.Equalf(t, order.Type, entity.OrderTypeIncome, "")
	require.Equalf(t, order.Status, entity.OrderStatusProcessed, "")
}
