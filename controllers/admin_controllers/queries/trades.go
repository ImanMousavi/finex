package queries

import (
	"github.com/zsmartex/finex/types"
)

type TradeFilters struct {
	Market     string          `query:"market"`
	Type       types.TakerType `query:"type"`
	MarketType string          `query:"market"`
	OrderID    uint64          `query:"order_id"`
	UID        string          `query:"uid"`
	Limit      int             `query:"limit"`
	Page       int             `query:"page"`
	TimeFrom   int64           `query:"time_from"`
	TimeTo     int64           `query:"time_to"`
	OrderBy    types.OrderBy   `query:"order_by"`
}
