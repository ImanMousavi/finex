package entities

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/go-finex/types"
)

type TradeEntities struct {
	ID          uint64          `json:"id"`
	Market      string          `json:"market"`
	Price       decimal.Decimal `json:"price"`
	Amount      decimal.Decimal `json:"amount"`
	Total       decimal.Decimal `json:"total"`
	FeeCurrency string          `json:"fee_currency"`
	Fee         decimal.Decimal `json:"fee"`
	FeeAmount   decimal.Decimal `json:"fee_amount"`
	TakerType   types.TakerType `json:"taker_type"`
	Side        types.TakerType `json:"side"`
	OrderID     uint64          `json:"order_id"`
	CreatedAt   time.Time       `json:"created_at"`
}