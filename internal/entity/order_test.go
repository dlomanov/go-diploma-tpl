package entity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestNewIncomeOrder(t *testing.T) {
	userID := uuid.New()
	orderNumber := OrderNumber("test")

	order, err := NewIncomeOrder(orderNumber, UserID(userID))
	require.NoError(t, err)
	require.Equal(t, OrderTypeIncome, order.Type, "Order type mismatch, want %s, got %s", OrderTypeIncome, order.Type)
	require.Equal(t, OrderStatusNew, order.Status, "Order status mismatch, want %s, got %s", OrderStatusNew, order.Status)
	require.Equal(t, decimal.Zero, order.Amount, "Order amount mismatch, want %s, got %s", decimal.Zero, order.Amount)
	require.Equal(t, userID, uuid.UUID(order.UserID), "User ID mismatch, want %s, got %s", userID, uuid.UUID(order.UserID))
}

func TestNewOutcomeOrder(t *testing.T) {
	userID := uuid.New()
	orderNumber := OrderNumber("test")
	amount := decimal.NewFromInt(100)

	order, err := NewOutcomeOrder(orderNumber, amount, UserID(userID))
	require.NoError(t, err, "Failed to create new outcome order")
	require.Equal(t, OrderTypeOutcome, order.Type, "Order type mismatch, want %s, got %s", OrderTypeOutcome, order.Type)
	require.Equal(t, OrderStatusProcessed, order.Status, "Order status mismatch, want %s, got %s", OrderStatusProcessed, order.Status)
	require.Equal(t, amount, order.Amount, "Order amount mismatch, want %s, got %s", amount, order.Amount)
	require.Equal(t, userID, uuid.UUID(order.UserID), "User ID mismatch, want %s, got %s", userID, uuid.UUID(order.UserID))
}

func TestOrderUpdate(t *testing.T) {
	order, err := NewIncomeOrder(OrderNumber("test"), newUserID())
	require.NoError(t, err)

	type (
		args struct {
			accrual OrderAccrual
			reset   bool
		}
		want struct {
			err    error
			event  OrderEvent
			amount decimal.Decimal
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
				accrual: OrderAccrual{
					Amount: decimal.NewFromInt(100),
					Status: OrderStatusProcessing,
				},
			},
			want: want{
				event: OrderEventUpdated,
			},
		},
		{
			name: "update inavlid - ok",
			args: args{
				accrual: OrderAccrual{
					Amount: decimal.NewFromInt(100),
					Status: OrderStatusInvalid,
				},
			},
			want: want{
				event: OrderEventCompleted,
			},
		},
		{
			name: "update processed - fail",
			args: args{
				accrual: OrderAccrual{
					Amount: decimal.NewFromInt(100),
					Status: OrderStatusProcessed,
				},
			},
			want: want{
				err: ErrOrderStatusFinal,
			},
		},
		{
			name: "update processing after reset - ok",
			args: args{
				reset: true,
				accrual: OrderAccrual{
					Amount: decimal.NewFromInt(100),
					Status: OrderStatusProcessing,
				},
			},
			want: want{
				event: OrderEventUpdated,
			},
		},
		{
			name: "update processed - ok",
			args: args{
				reset: true,
				accrual: OrderAccrual{
					Amount: decimal.NewFromInt(10),
					Status: OrderStatusProcessed,
				},
			},
			want: want{
				event:  OrderEventCompleted,
				amount: decimal.NewFromInt(10),
			},
		},
		{
			name: "update invalid - fail",
			args: args{
				accrual: OrderAccrual{
					Amount: decimal.NewFromInt(100),
					Status: OrderStatusInvalid,
				},
			},
			want: want{
				err: ErrOrderStatusFinal,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.reset {
				order, err = NewIncomeOrder(OrderNumber("test"), newUserID())
				require.NoError(t, err)
			}

			err := order.Update(tt.args.accrual)
			require.ErrorIs(t, err, tt.want.err, "Error mismatch, want %v, got %v", tt.want.err, err)
			if err != nil {
				return
			}
			require.Equal(t, tt.args.accrual.Status, order.Status, "Status mismatch, want %v, got %v", tt.args.accrual.Status, order.Status)
			require.True(t, order.Events.Contains(Event(tt.want.event)), "Event not found, want %v", tt.want.event)
			require.True(t, order.Amount.Equal(tt.want.amount), "Amount mismatch, want %v, got %v", tt.want.amount, order.Amount)
		})
	}
}

func TestOrderStatusFinal(t *testing.T) {
	require.True(t, OrderStatusInvalid.Final(), "Invalid order status is not final")
	require.True(t, OrderStatusProcessed.Final(), "Processed order status is not final")
	require.False(t, OrderStatusNew.Final(), "New order status is final")
	require.False(t, OrderStatusProcessing.Final(), "Processing order status is final")
}
