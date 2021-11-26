package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/types"
)

type OrderEntity struct {
	UUID            uuid.UUID           `json:"uuid"`
	Market          string              `json:"market"`
	Side            string              `json:"side"`
	OrdType         types.OrderType     `json:"ord_type"`
	Price           decimal.NullDecimal `json:"price"`
	StopPrice       decimal.NullDecimal `json:"stop_price"`
	AvgPrice        decimal.Decimal     `json:"avg_price"`
	State           string              `json:"state"`
	OriginVolume    decimal.Decimal     `json:"origin_volume"`
	RemainingVolume decimal.Decimal     `json:"remaining_volume"`
	ExecutedVolume  decimal.Decimal     `json:"executed_volume"`
	TradesCount     int64               `json:"trades_count"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}
