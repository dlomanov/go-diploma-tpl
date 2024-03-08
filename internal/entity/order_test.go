package entity_test

import (
	"testing"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestNewIncomeOrder(t *testing.T) {
	userID := uuid.New()
	orderNumber := entity.OrderNumber("test")

	order, err := entity.NewIncomeOrder(orderNumber, entity.UserID(userID))
	require.NoError(t, err)
	require.Equal(t, entity.OrderTypeIncome, order.Type, "Order type mismatch, want %s, got %s", entity.OrderTypeIncome, order.Type)
	require.Equal(t, entity.OrderStatusNew, order.Status, "Order status mismatch, want %s, got %s", entity.OrderStatusNew, order.Status)
	require.Equal(t, decimal.Zero, order.Amount, "Order amount mismatch, want %s, got %s", decimal.Zero, order.Amount)
	require.Equal(t, userID, uuid.UUID(order.UserID), "User ID mismatch, want %s, got %s", userID, uuid.UUID(order.UserID))
}

func TestNewOutcomeOrder(t *testing.T) {
	userID := uuid.New()
	orderNumber := entity.OrderNumber("test")
	amount := decimal.NewFromInt(100)

	order, err := entity.NewOutcomeOrder(orderNumber, amount, entity.UserID(userID))
	require.NoError(t, err, "Failed to create new outcome order")
	require.Equal(t, entity.OrderTypeOutcome, order.Type, "Order type mismatch, want %s, got %s", entity.OrderTypeOutcome, order.Type)
	require.Equal(t, entity.OrderStatusProcessed, order.Status, "Order status mismatch, want %s, got %s", entity.OrderStatusProcessed, order.Status)
	require.Equal(t, amount, order.Amount, "Order amount mismatch, want %s, got %s", amount, order.Amount)
	require.Equal(t, userID, uuid.UUID(order.UserID), "User ID mismatch, want %s, got %s", userID, uuid.UUID(order.UserID))
}

func TestOrderUpdate(t *testing.T) {
	order, err := entity.NewIncomeOrder(entity.OrderNumber("test"), newUserID())
	require.NoError(t, err)

	type (
		args struct {
			accrual entity.OrderAccrual
			reset   bool
		}
		want struct {
			err    error
			amount decimal.Decimal
			status entity.OrderStatus
		}
	)
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "update processing - ok",
			args: args{
				accrual: entity.OrderAccrual{
					Amount: decimal.NewFromInt(100),
					Status: entity.OrderStatusProcessing,
				},
			},
			want: want{
				status: entity.OrderStatusProcessing,
			},
		},
		{
			name: "update inavlid - ok",
			args: args{
				accrual: entity.OrderAccrual{
					Amount: decimal.NewFromInt(100),
					Status: entity.OrderStatusInvalid,
				},
			},
			want: want{
				status: entity.OrderStatusInvalid,
			},
		},
		{
			name: "update processed - fail",
			args: args{
				accrual: entity.OrderAccrual{
					Amount: decimal.NewFromInt(100),
					Status: entity.OrderStatusProcessed,
				},
			},
			want: want{
				err:    entity.ErrOrderStatusFinal,
				status: entity.OrderStatusInvalid,
			},
		},
		{
			name: "update processing after reset - ok",
			args: args{
				reset: true,
				accrual: entity.OrderAccrual{
					Amount: decimal.NewFromInt(100),
					Status: entity.OrderStatusProcessing,
				},
			},
			want: want{
				status: entity.OrderStatusProcessing,
			},
		},
		{
			name: "update processed - ok",
			args: args{
				reset: true,
				accrual: entity.OrderAccrual{
					Amount: decimal.NewFromInt(10),
					Status: entity.OrderStatusProcessed,
				},
			},
			want: want{
				status: entity.OrderStatusProcessed,
				amount: decimal.NewFromInt(10),
			},
		},
		{
			name: "update invalid - fail",
			args: args{
				accrual: entity.OrderAccrual{
					Amount: decimal.NewFromInt(100),
					Status: entity.OrderStatusInvalid,
				},
			},
			want: want{
				err:    entity.ErrOrderStatusFinal,
				status: entity.OrderStatusProcessed,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.reset {
				order, err = entity.NewIncomeOrder(entity.OrderNumber("test"), newUserID())
				require.NoError(t, err)
			}

			err := order.Update(tt.args.accrual)
			require.Equal(t, tt.want.status, order.Status, "Status mismatch, want %v, got %v", tt.want.status, order.Status)
			require.ErrorIs(t, err, tt.want.err, "Error mismatch, want %v, got %v", tt.want.err, err)
			if err != nil {
				return
			}
			require.Equal(t, tt.args.accrual.Status, order.Status, "Status mismatch, want %v, got %v", tt.args.accrual.Status, order.Status)
			require.True(t, order.Amount.Equal(tt.want.amount), "Amount mismatch, want %v, got %v", tt.want.amount, order.Amount)
		})
	}
}

func TestOrderStatusFinal(t *testing.T) {
	require.True(t, entity.OrderStatusInvalid.Final(), "Invalid order status is not final")
	require.True(t, entity.OrderStatusProcessed.Final(), "Processed order status is not final")
	require.False(t, entity.OrderStatusNew.Final(), "New order status is final")
	require.False(t, entity.OrderStatusProcessing.Final(), "Processing order status is final")
}
