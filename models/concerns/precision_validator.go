package concerns

import (
	"github.com/shopspring/decimal"
)

type PrecisionValidator struct {
}

func (p PrecisionValidator) LessThanOrEqTo(value float64, precision int32) bool {
	value_rounded, _ := decimal.NewFromFloat(value).Round(precision).Float64()
	return value == value_rounded
}
