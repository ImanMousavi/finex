package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type CurrencyType = string

var (
	TypeCoin CurrencyType = "coin"
	TypeFiat CurrencyType = "fiat"
)

type Currency struct {
	ID          string          `json:"id" gorm:"primaryKey"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Homepage    string          `json:"homepage"`
	Type        CurrencyType    `json:"type"`
	Precision   string          `json:"precision"`
	IconURL     string          `json:"icon_url"`
	Price       decimal.Decimal `json:"price"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
