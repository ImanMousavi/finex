package queries

import "github.com/zsmartex/finex/controllers/helpers"

type CommissionQueries struct {
	Limit int `query:"limit" validate:"uint"`
	Page  int `query:"page" validate:"uint"`
}

func (t CommissionQueries) Messages() map[string]string {
	return helpers.VaildateMessage("referral.commission")
}

func (t CommissionQueries) Translates() map[string]string {
	return helpers.VaildateTranslateFields()
}
