package models

import (
	"time"

	"gitlab.com/zsmartex/finex/config"
)

type Member struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	UID         string    `json:"uid"`
	Email       string    `json:"email"`
	Level       int32     `json:"level" gorm:"default:0" validate:"min:0"`
	Role        string    `json:"role"`
	Group       string    `json:"group" gorm:"default:vip-1"`
	State       string    `json:"state"`
	ReferralUID *string   `json:"referral_uid"`
	Username    *string   `json:"username"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (m Member) GetAccount(currency Currency) Account {
	var account Account

	config.DataBase.FirstOrCreate(&account, Account{MemberID: m.ID, CurrencyID: currency.ID})

	return account
}
