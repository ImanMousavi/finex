package concerns

import (
	"github.com/shopspring/decimal"
)

type PrecisionValidator struct {
}

func (p PrecisionValidator) LessThanOrEqTo(value decimal.Decimal, precision int32) bool {
	value_rounded := value.Round(precision)
	return value.Equal(value_rounded)
}
