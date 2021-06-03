package models

import "time"

type CurrencyType = string

var (
	TypeCoin CurrencyType = "coin"
	TypeFiat CurrencyType = "fiat"
)

type Currency struct {
	ID                  string    `json:"id" gorm:"primaryKey"`
	Name                *string   `json:"name"`
	Description         *string   `json:"description"`
	Homepage            *string   `json:"homepage"`
	BlockchainKey       *string   `json:"blockchain_key"`
	ParentID            *string   `json:"parent_id"`
	Type                string    `json:"type" gorm:"default:coin"`
	DepositFee          float64   `json:"deposit_fee" gorm:"default:0.0"`
	MinDepositAmount    float64   `json:"min_deposit_amount" gorm:"default:0.0"`
	MinCollectionAmount float64   `json:"min_collection_amount" gorm:"default:0.0"`
	WithdrawFee         float64   `json:"withdraw_fee" gorm:"default:0.0"`
	MinWithdrawAmount   float64   `json:"min_withdraw_amount" gorm:"default:0.0"`
	WithdrawLimit24h    float64   `json:"withdraw_limit_24h" gorm:"default:0.0"`
	WithdrawLimit72h    float64   `json:"withdraw_limit_72h" gorm:"default:0.0"`
	DepositEnabled      bool      `json:"deposit_enabled"`
	WithdrawalEnabled   bool      `json:"withdrawal_enabled"`
	BaseFactor          int64     `json:"base_factor"`
	Precision           int16     `json:"precision"`
	IconURL             *string   `json:"icon_url"`
	Price               *float64  `json:"price"`
	Visible             bool      `json:"visible"`
	Position            int32     `json:"position" gorm:"default:0"`
	Options             *string   `json:"options"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
