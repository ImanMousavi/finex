package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Market struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	BaseUnit        string    `json:"base_unit"`
	QuoteUnit       string    `json:"quote_unit"`
	AmountPrecision int32     `json:"amount_precision"`
	PricePrecision  int32     `json:"price_precision"`
	TotalPrecision  int32     `json:"total_precision"`
	MaxPrice        float64   `json:"max_price"`
	MinPrice        float64   `json:"min_price"`
	MinAmount       float64   `json:"min_amount"`
	State           string    `json:"state"`
	EngineID        int64     `json:"engine_id"`
	Position        int32     `json:"position"`
	Data            string    `json:"data"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (m Market) round_price(val float64) float64 {
	value_rounded, _ := decimal.NewFromFloat(val).Round(int32(m.PricePrecision)).Float64()

	return value_rounded
}

func (m Market) round_amount(val float64) float64 {
	value_rounded, _ := decimal.NewFromFloat(val).Round(int32(m.AmountPrecision)).Float64()

	return value_rounded
}
