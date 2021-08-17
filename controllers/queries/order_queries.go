package queries

import (
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/types"
)

type OrderFilters struct {
	Market    string          `query:"market"`
	BaseUnit  string          `query:"base_unit"`
	QuoteUnit string          `query:"quote_unit"`
	State     string          `query:"state" validate:"ValidateOrderState"`
	Limit     int             `query:"limit" validate:"uint"`
	Page      int             `query:"page" validate:"uint"`
	Type      types.OrderSide `query:"type" validate:"ValidateSide"`
	TimeFrom  int64           `query:"time_from" validate:"uint"`
	TimeTo    int64           `query:"time_to" validate:"uint"`
	OrderBy   types.OrderBy   `query:"order_by" validate:"ValidateOrderBy"`
}

func (t OrderFilters) ValidateOrderBy(val types.OrderBy) bool {
	return helpers.ValidateOrderBy(val)
}

func (t OrderFilters) ValidateSide(val types.OrderSide) bool {
	return helpers.ValidateSide(val)
}

func (t OrderFilters) Messages() map[string]string {
	return helpers.VaildateMessage("market.order")
}

func (t OrderFilters) Translates() map[string]string {
	return helpers.VaildateTranslateFields()
}

type CancelOrderParams struct {
	Market string          `json:"market" form:"market" validate:"ValidateType"`
	Side   types.TakerType `json:"side" form:"side"`
}

func (t CancelOrderParams) ValidateType(val types.TakerType) bool {
	return helpers.ValidateTakerType(val)
}

func (t CancelOrderParams) Messages() map[string]string {
	return helpers.VaildateMessage("market.order")
}

func (t CancelOrderParams) Translates() map[string]string {
	return helpers.VaildateTranslateFields()
}
