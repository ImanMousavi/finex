package queries

import "github.com/zsmartex/finex/controllers/helpers"

type DepthQuery struct {
	Limit int `query:"limit" validate:"uint"`
}

func (t DepthQuery) Messages() map[string]string {
	return helpers.VaildateMessage("public.market_depth")
}

func (t DepthQuery) Translates() map[string]string {
	return helpers.VaildateTranslateFields()
}
