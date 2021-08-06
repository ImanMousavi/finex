package queries

import (
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/types"
)

type TradeFilters struct {
	Market   string          `query:"market"`
	Type     types.TakerType `query:"type" validate:"ValidateType"`
	Limit    int             `query:"limit" validate:"uint"`
	Page     int             `query:"page" validate:"uint"`
	TimeFrom int64           `query:"time_from" validate:"uint"`
	TimeTo   int64           `query:"time_to" validate:"uint"`
	OrderBy  types.OrderBy   `query:"order_by" validate:"ValidateOrderBy"`
}

func (t TradeFilters) ValidateType(val types.TakerType) bool {
	return helpers.ValidateTakerType(val)
}

func (t TradeFilters) ValidateOrderBy(val types.OrderBy) bool {
	return helpers.ValidateOrderBy(val)
}

func (t TradeFilters) Messages() map[string]string {
	return helpers.VaildateMessage("market.trade")
}

func (t TradeFilters) Translates() map[string]string {
	return helpers.VaildateTranslateFields()
}
