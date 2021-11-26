package entities

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/types"
)

type IEO struct {
	ID                  int64             `json:"id"`
	CurrencyID          string            `json:"currency_id"`
	Price               decimal.Decimal   `json:"price"`
	MainPaymentCurrency string            `json:"main_payment_currency"`
	PaymentCurrencies   []string          `json:"payment_currencies"`
	ExecutedQuantity    decimal.Decimal   `json:"executed_quantity"`
	OriginQuantity      decimal.Decimal   `json:"origin_quantity"`
	LimitPerUser        decimal.Decimal   `json:"limit_per_user"`
	MinAmount           decimal.Decimal   `json:"min_amount"`
	State               types.MarketState `json:"state"`
	StartTime           int64             `json:"start_time"`
	EndTime             int64             `json:"end_time"`
	Data                string            `json:"data"`
	BannerUrl           string            `json:"banner_url"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}
