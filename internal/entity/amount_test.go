package entity

import (
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAmount_Add(t *testing.T) {
	tests := []struct {
		name string
		a    Amount
		b    Amount
		want Amount
	}{
		{
			name: "1",
			a:    Amount(decimal.NewFromFloat(1.333333)),
			b:    Amount(decimal.NewFromFloat(1.333333)),
			want: Amount(decimal.NewFromFloat(2.67)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.True(t, tt.a.Add(tt.b).Equal(tt.want))
		})
	}
}

func TestAmount_Sub(t *testing.T) {
	tests := []struct {
		name string
		a    Amount
		b    Amount
		want Amount
	}{
		{
			name: "1",
			a:    Amount(decimal.NewFromFloat(1.333333)),
			b:    Amount(decimal.NewFromFloat(1.333333)),
			want: Amount(decimal.NewFromFloat(0)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.True(t, tt.a.Sub(tt.b).Equal(ZeroAmount()))
		})
	}
}
