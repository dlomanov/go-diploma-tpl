package entity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestNewBalance(t *testing.T) {
	userID := newUserID()
	balance := NewBalance(UserID(userID))

	require.Equal(t, uuid.UUID(userID), uuid.UUID(balance.UserID), "UserID mismatch")
	require.Equal(t, decimal.Zero, balance.Current, "Current balance is not zero")
	require.Equal(t, decimal.Zero, balance.Withdrawn, "Withdrawn balance is not zero")
	require.False(t, balance.CreatedAt.IsZero(), "CreatedAt is zero")
	require.False(t, balance.UpdatedAt.IsZero(), "UpdatedAt is zero")
}

func TestBalanceUpdate(t *testing.T) {
	balance := NewBalance(newUserID())
	tempNumber := OrderNumber("temp")

	type args struct {
		order *Order
	}
	type want struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "invalid user",
			args: args{
				order: must(t, func() (any, error) { return NewIncomeOrder(tempNumber, newUserID()) }).(*Order),
			},
			want: want{
				err: ErrBalanceUserInvalid,
			},
		},
		{
			name: "invalid status",
			args: args{
				order: must(t, func() (any, error) { return NewIncomeOrder(tempNumber, balance.UserID) }).(*Order),
			},
			want: want{
				err: ErrBalanceOrderStatusInvalid,
			},
		},
		{
			name: "invalid amount",
			args: args{
				order: must(t, func() (any, error) {
					o, err := NewOutcomeOrder(tempNumber, decimal.NewFromInt(0), balance.UserID)
					if err != nil {
						return nil, err
					}
					o.Amount = decimal.NewFromInt(-1)
					return o, nil
				}).(*Order),
			},
			want: want{
				err: ErrBalanceOrderAmountInvalid,
			},
		},
	}

	for _, tt := range tests {
		err := balance.Update(*tt.args.order)
		require.ErrorIs(t, err, tt.want.err)
	}
}

func must(t *testing.T, f func() (any, error)) any {
	v, err := f()
	if err != nil {
		require.NoError(t, err)
	}
	return v
}

func newUserID() UserID {
	return UserID(uuid.New())
}
