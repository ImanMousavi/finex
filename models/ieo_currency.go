package models

import (
	"time"
)

type IEOCurrency struct {
	ID         int64     `json:"id"`
	CurrencyID string    `json:"currency"`
	IEOID      int64     `json:"ieo_id" gorm:"column:ieo_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (IEOCurrency) TableName() string {
	return "ieo_currencies"
}
