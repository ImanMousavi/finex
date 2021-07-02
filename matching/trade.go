package matching

import (
	"github.com/shopspring/decimal"
)

// Trade .
type Trade struct {
	ID           uint64          `json:"id"`
	Symbol       string          `json:"symbol"`
	Price        decimal.Decimal `json:"price"`
	Quantity     decimal.Decimal `json:"quantity"`
	Total        decimal.Decimal `json:"total"`
	MakerOrderID uint64          `json:"maker_order_id"`
	TakerOrderID uint64          `json:"taker_order_id"`
	MakerID      uint64          `json:"maker_id"`
	TakerID      uint64          `json:"taker_id"`
}
