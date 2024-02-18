package entity

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrder_NewIncomeOrder(t *testing.T) {
	number := OrderNumber("123")
	userID := UserID(newUUID(t))
	order, err := NewIncomeOrder(number, userID)
	assert.NoError(t, err)
	assert.Equal(t, OrderStatusNew, order.Status)
	assert.Equal(t, OrderTypeIncome, order.Type)
	assert.Equal(t, ZeroAmount(), order.Amount)
	assert.Equal(t, number, order.Number)
	assert.Equal(t, userID, order.UserID)
	assert.NotEmpty(t, order.CreatedAt)
	assert.NotEmpty(t, order.UpdatedAt)
}

func TestOrder_NewOutcomeOrder(t *testing.T) {
	number := OrderNumber("123")
	amount := NewAmount(10)
	userID := UserID(newUUID(t))
	order, err := NewOutcomeOrder(number, amount, userID)
	assert.NoError(t, err)
	assert.Equal(t, OrderStatusProcessed, order.Status)
	assert.Equal(t, OrderTypeOutcome, order.Type)
	assert.Equal(t, amount, order.Amount)
	assert.Equal(t, number, order.Number)
	assert.Equal(t, userID, order.UserID)
	assert.NotEmpty(t, order.CreatedAt)
	assert.NotEmpty(t, order.UpdatedAt)
}

func TestOrder_Update(t *testing.T) {
	type args struct {
		newOrder bool
		amount   Amount
		status   OrderStatus
	}
	type want struct {
		err       error
		amount    Amount
		status    OrderStatus
		event     Event
		wantEvent bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "1",
			args: args{
				status: OrderStatusProcessing,
				amount: NewAmount(10),
			},
			want: want{
				err:       nil,
				amount:    ZeroAmount(),
				status:    OrderStatusProcessing,
				event:     OrderEventUpdated,
				wantEvent: true,
			},
		},
		{
			name: "2",
			args: args{
				status: OrderStatusNew,
				amount: NewAmount(10),
			},
			want: want{
				err:       nil,
				amount:    ZeroAmount(),
				status:    OrderStatusNew,
				event:     OrderEventUpdated,
				wantEvent: true,
			},
		},
		{
			name: "3",
			args: args{
				status: OrderStatusNew,
				amount: NewAmount(10),
			},
			want: want{
				err:       nil,
				amount:    ZeroAmount(),
				status:    OrderStatusNew,
				wantEvent: false,
			},
		},
		{
			name: "4",
			args: args{
				status: OrderStatusProcessed,
				amount: NewAmount(10),
			},
			want: want{
				err:       nil,
				amount:    NewAmount(10),
				status:    OrderStatusProcessed,
				event:     OrderEventCompleted,
				wantEvent: true,
			},
		},
		{
			name: "5",
			args: args{
				status: OrderStatusNew,
				amount: ZeroAmount(),
			},
			want: want{
				err:    ErrOrderStatusFinal,
				amount: NewAmount(10),
				status: OrderStatusProcessed,
			},
		},
		{
			name: "6",
			args: args{
				newOrder: true,
				status:   OrderStatusInvalid,
				amount:   NewAmount(10),
			},
			want: want{
				err:       nil,
				amount:    ZeroAmount(),
				status:    OrderStatusInvalid,
				event:     OrderEventUpdated,
				wantEvent: true,
			},
		},
		{
			name: "7",
			args: args{
				status: OrderStatusProcessed,
				amount: NewAmount(10),
			},
			want: want{
				err:    ErrOrderStatusFinal,
				amount: ZeroAmount(),
				status: OrderStatusInvalid,
			},
		},
	}

	number := OrderNumber("123")
	userID := UserID(newUUID(t))
	var (
		order *Order
		err   error
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if order == nil || tt.args.newOrder {
				order, err = NewIncomeOrder(number, userID)
				require.NoError(t, err)
			}
			err = order.Update(tt.args.status, tt.args.amount)
			require.ErrorIs(t, tt.want.err, err)
			require.Equal(t, tt.want.status, order.Status)
			require.Equal(t, tt.want.amount, order.Amount)
			if tt.want.wantEvent {
				events := order.Events
				require.False(t, events.Empty())
				require.Len(t, events, 1)
				require.True(t, events.Contains(tt.want.event))
			} else {
				events := order.Events
				require.True(t, events.Empty())
			}
			order.Events = NewEvents()
		})
	}
}

func newUUID(t *testing.T) uuid.UUID {
	id, err := uuid.NewUUID()
	require.NoError(t, err)
	return id
}
