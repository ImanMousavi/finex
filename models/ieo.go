package models

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/types"
)

type IEO struct {
	ID                  int64             `json:"id"`
	CurrencyID          string            `json:"currency_id"`
	MainPaymentCurrency string            `json:"main_payment_currency"`
	Description         string            `json:"description"`
	Price               decimal.Decimal   `json:"price"`
	MinAmount           decimal.Decimal   `json:"min_amount"`
	State               types.MarketState `json:"state"`
	StartTime           time.Time         `json:"start_time"`
	EndTime             time.Time         `json:"end_time"`
	Data                string            `json:"data"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

func (IEO) TableName() string {
	return "ieos"
}

func (m *IEO) IsEnabled() bool {
	return m.State == "enabled"
}

func (m *IEO) IsEnded() bool {
	return time.Now().After(m.EndTime)
}

func (m *IEO) IsStarted() bool {
	return time.Now().After(m.StartTime)
}

func (m *IEO) PaymentCurrencies() []*Currency {
	var currencies []*Currency

	config.DataBase.Find(&currencies, "id IN (SELECT currency_id FROM ieo_currencies WHERE ieo_id = ?)", m.ID)

	return currencies
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
