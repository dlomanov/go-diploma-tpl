package entity

import "github.com/shopspring/decimal"

const places = 2

type Amount decimal.Decimal

func NewAmount(value float64) Amount {
	return Amount(decimal.NewFromFloat(value))
}

func (a Amount) Add(b Amount) Amount {
	ad := decimal.Decimal(a)
	bd := decimal.Decimal(b)
	return Amount(ad.Add(bd).Round(places))
}

func (a Amount) Sub(b Amount) Amount {
	ad := decimal.Decimal(a)
	bd := decimal.Decimal(b)
	return Amount(ad.Sub(bd).Round(places))
}

func (a Amount) IsNegative() bool {
	ad := decimal.Decimal(a)
	return ad.IsNegative()
}

func (a Amount) Equal(b Amount) bool {
	ad := decimal.Decimal(a)
	bd := decimal.Decimal(b)
	return ad.Equal(bd)
}

func ZeroAmount() Amount {
	return Amount(decimal.Zero)
}
