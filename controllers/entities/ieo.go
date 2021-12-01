package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

type IEO struct {
	ID                  int64           `json:"id"`
	CurrencyID          string          `json:"currency_id"`
	Price               decimal.Decimal `json:"price"`
	MainPaymentCurrency string          `json:"main_payment_currency"`
	PaymentCurrencies   []string        `json:"payment_currencies"`
	ExecutedQuantity    decimal.Decimal `json:"executed_quantity"`
	OriginQuantity      decimal.Decimal `json:"origin_quantity"`
	LimitPerUser        decimal.Decimal `json:"limit_per_user"`
	MinAmount           decimal.Decimal `json:"min_amount"`
	StartTime           int64           `json:"start_time"`
	EndTime             int64           `json:"end_time"`
	Ended               bool            `json:"ended"`
	BoughtQuantity      decimal.Decimal `json:"bought_quantity,omitempty"`
	BannerUrl           string          `json:"banner_url"`
	Data                string          `json:"data"`
	Distributors        int64           `json:"distributors"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}
