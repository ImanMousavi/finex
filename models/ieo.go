package models

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/types"
)

type IEO struct {
	ID                  int64
	CurrencyID          string
	MainPaymentCurrency string
	Price               decimal.Decimal
	MinAmount           decimal.Decimal
	State               types.MarketState
	ExecutedQuantity    decimal.Decimal
	OriginQuantity      decimal.Decimal
	LimitPerUser        decimal.Decimal
	StartTime           time.Time
	EndTime             time.Time
	Data                string
	BannerUrl           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
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

func (m *IEO) IsCompleted() bool {
	return m.ExecutedQuantity.Equal(m.OriginQuantity)
}

func (m *IEO) IsStarted() bool {
	return time.Now().After(m.StartTime)
}

func (m *IEO) PaymentCurrencies() []string {
	var currencies []*Currency

	config.DataBase.Find(&currencies, "id IN (SELECT currency_id FROM ieo_payment_currencies WHERE ieo_id = ?)", m.ID)

	ids := make([]string, 0)

	for _, currency := range currencies {
		ids = append(ids, currency.ID)
	}

	return ids
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

func (m *IEO) MemberBoughtQuantity(member_id int64) decimal.Decimal {
	var orders []*IEOOrder
	config.DataBase.Find(&orders, "ieo_id = ? member_id = ? AND state = ?", m.ID, member_id, StateDone)

	user_bought_quantity := decimal.Decimal{}
	for _, order := range orders {
		user_bought_quantity = user_bought_quantity.Add(order.Quantity)
	}

	return user_bought_quantity
}
