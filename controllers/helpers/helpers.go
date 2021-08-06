package helpers

import (
	"github.com/gookit/validate"
	"github.com/zsmartex/finex/types"
)

type Errors struct {
	Errors []string `json:"errors"`
}

func (e Errors) Size() int {
	return len(e.Errors)
}

func Vaildate(payload interface{}, err_src *Errors) {
	v := validate.Struct(payload)
	if !v.Validate() {
		for _, errs := range v.Errors.All() {
			for _, err := range errs {
				err_src.Errors = append(err_src.Errors, err)
			}
		}
	}
}

func VaildateMessage(prefix string) map[string]string {
	return validate.MS{
		"uint":               prefix + ".non_integer_{field}",
		"ValidateOrderState": prefix + ".invalid_{field}",
		"ValidateType":       prefix + ".invalid_{field}",
		"ValidateOrderBy":    prefix + ".invalid_{field}",
	}
}

func VaildateTranslateFields() map[string]string {
	return validate.MS{
		"Market":    "market",
		"BaseUnit":  "base_unit",
		"QuoteUnit": "quote_unit",
		"Side":      "side",
		"State":     "state",
		"Type":      "type",
		"Limit":     "limit",
		"Page":      "page",
		"TimeFrom":  "time_from",
		"TimeTo":    "time_to",
		"OrderBy":   "order_by",
	}
}

func ValidateOrderState(val string) bool {
	switch val {
	case "pending":
		return true
	case "wait":
		return true
	case "cancel":
		return true
	case "done":
		return true
	case "reject":
		return true
	default:
		return false
	}
}

func ValidateOrderBy(val types.OrderBy) bool {
	if val == types.OrderByAsc {
		return true
	} else if val == types.OrderByDesc {
		return true
	}

	return false
}

func ValidateTakerType(val types.TakerType) bool {
	if val == types.TypeBuy {
		return true
	} else if val == types.TypeSell {
		return true
	}

	return false
}
