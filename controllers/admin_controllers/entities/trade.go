package entities

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/types"
)

type TradeEntity struct {
	ID     uint64          `json:"id"`
	Price  decimal.Decimal `json:"price"`
	Amount decimal.Decimal `json:"amount"`
	Total  decimal.Decimal `json:"total"`
	Market string          `json:"market"`
	// MarketType       string          `json:"market_type"` TODO: next update
	TakerType        types.TakerType `json:"taker_type"`
	MakerOrderEmail  string          `json:"maker_order_email"`
	TakerOrderEmail  string          `json:"taker_order_email"`
	MakerUID         string          `json:"maker_uid"`
	TakerUID         string          `json:"taker_uid"`
	MakerFee         decimal.Decimal `json:"maker_fee"`
	MakerFeeAmount   decimal.Decimal `json:"maker_fee_amount"`
	MakerFeeCurrency string          `json:"maker_fee_currency"`
	TakerFee         decimal.Decimal `json:"taker_fee"`
	TakerFeeAmount   decimal.Decimal `json:"taker_fee_amount"`
	TakerFeeCurrency string          `json:"taker_fee_currency"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}
