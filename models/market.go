package models

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/pkg"
)

type Market struct {
	ID              int64           `json:"id" gorm:"primaryKey"`
	Symbol          string          `json:"symbol"`
	Type            string          `json:"type"`
	BaseUnit        string          `json:"base_unit"`
	QuoteUnit       string          `json:"quote_unit"`
	AmountPrecision int             `json:"amount_precision"`
	PricePrecision  int             `json:"price_precision"`
	TotalPrecision  int             `json:"total_precision"`
	MaxPrice        decimal.Decimal `json:"max_price"`
	MinPrice        decimal.Decimal `json:"min_price"`
	MinAmount       decimal.Decimal `json:"min_amount"`
	State           string          `json:"state"`
	EngineID        int64           `json:"engine_id"`
	Position        int32           `json:"position"`
	Data            string          `json:"data"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (m *Market) GetSymbol() pkg.Symbol {
	return pkg.Symbol{BaseCurrency: m.BaseUnit, QuoteCurrency: m.QuoteUnit}
}

func (m Market) round_price(val decimal.Decimal) decimal.Decimal {
	value_rounded := val.Round(int32(m.PricePrecision))

	return value_rounded
}

func (m Market) round_amount(val decimal.Decimal) decimal.Decimal {
	value_rounded := val.Round(int32(m.AmountPrecision))

	return value_rounded
}
