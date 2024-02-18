package entity

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewBalance(t *testing.T) {
	userID := UserID(newUUID(t))
	balance := NewBalance(userID)
	assert.Equal(t, userID, balance.UserID)
	assert.Equal(t, ZeroAmount(), balance.Current)
	assert.Equal(t, ZeroAmount(), balance.Withdrawn)
	assert.NotEmpty(t, balance.CreatedAt)
	assert.NotEmpty(t, balance.UpdatedAt)
}

func TestBalance_Update(t *testing.T) {
	userID := UserID(newUUID(t))

	type args struct {
		order Order
	}
	type want struct {
		err       error
		current   Amount
		withdrawn Amount
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "1",
			args: args{order: newOrder(t, OrderTypeIncome, OrderStatusNew, NewAmount(10), userID)},
			want: want{
				err:       ErrBalanceOrderStatusInvalid,
				current:   ZeroAmount(),
				withdrawn: ZeroAmount(),
			},
		},
		{
			name: "2",
			args: args{order: newOrder(t, OrderTypeIncome, OrderStatusProcessed, NewAmount(10), UserID(newUUID(t)))},
			want: want{
				err:       ErrBalanceUserInvalid,
				current:   ZeroAmount(),
				withdrawn: ZeroAmount(),
			},
		},
		{
			name: "3",
			args: args{order: newOrder(t, OrderTypeIncome, OrderStatusProcessed, NewAmount(-10), userID)},
			want: want{
				err:       ErrBalanceOrderAmountInvalid,
				current:   ZeroAmount(),
				withdrawn: ZeroAmount(),
			},
		},
		{
			name: "4",
			args: args{order: newOrder(t, OrderTypeIncome, OrderStatusProcessed, NewAmount(10), userID)},
			want: want{
				err:       nil,
				current:   NewAmount(10),
				withdrawn: ZeroAmount(),
			},
		},
		{
			name: "5",
			args: args{order: newOrder(t, OrderTypeOutcome, "", NewAmount(10), userID)},
			want: want{
				err:       nil,
				current:   ZeroAmount(),
				withdrawn: NewAmount(10),
			},
		},
	}
	balance := NewBalance(userID)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := balance.Update(tt.args.order)
			require.ErrorIs(t, tt.want.err, err)
			require.True(t, tt.want.current.Equal(balance.Current))
			require.True(t, tt.want.withdrawn.Equal(balance.Withdrawn))
		})
	}
}

func newOrder(
	t *testing.T,
	typ OrderType,
	status OrderStatus,
	amount Amount,
	userID UserID,
) Order {
	switch typ {
	case OrderTypeIncome:
		order, err := NewIncomeOrder("123", userID)
		require.NoError(t, err)
		err = order.Update(status, amount)
		require.NoError(t, err)
		return *order

	case OrderTypeOutcome:
		order, err := NewOutcomeOrder("123", amount, userID)
		require.NoError(t, err)
		return *order
	default:
		panic(ErrOrderTypeInvalid)
	}
}
