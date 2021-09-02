package models

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/models/datatypes"
	"github.com/zsmartex/finex/types"
)

type IEO struct {
	ID                  uint64                         `json:"id"`
	CurrencyID          string                         `json:"currency_id"`
	MainPaymentCurrency string                         `json:"main_payment_currency"`
	Price               decimal.Decimal                `json:"price"`
	PaymentCurrencies   datatypes.IEOPaymentCurrencies `json:"payment_currencies"`
	MinAmount           decimal.Decimal                `json:"min_amount"`
	State               types.MarketState              `json:"state"`
	StartTime           time.Time                      `json:"start_time"`
	EndTime             time.Time                      `json:"end_time"`
	CreatedAt           time.Time                      `json:"created_at"`
	UpdatedAt           time.Time                      `json:"updated_at"`
}

func (m *IEO) IsEnabled() bool {
	return m.State == "enabled"
}

func (m *IEO) GetPriceByParent(currency_id string) decimal.Decimal {
	var price decimal.Decimal

	if currency_id == m.MainPaymentCurrency {
		price = m.Price
	} else {
		var currency *Currency

		config.DataBase.First(&currency, "id = ?", currency_id)
		price = m.Price.Div(currency.Price).Round(8)
	}

	return price
}
