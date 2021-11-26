package models

import (
	"database/sql"
	"time"

	"github.com/zsmartex/finex/config"
)

type Member struct {
	ID          int64          `json:"id" gorm:"primaryKey"`
	UID         string         `json:"uid"`
	Email       string         `json:"email"`
	Level       int32          `json:"level" gorm:"default:0" validate:"min:0"`
	Role        string         `json:"role"`
	Group       string         `json:"group" gorm:"default:vip-1"`
	State       string         `json:"state"`
	ReferralUID sql.NullString `json:"referral_uid"`
	Username    sql.NullString `json:"username"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (m *Member) GetAccount(currency *Currency) *Account {
	var account *Account

	config.DataBase.FirstOrCreate(&account, Account{MemberID: m.ID, CurrencyID: currency.ID})

	return account
}

func (m *Member) HavingReferraller() bool {
	return m.ReferralUID.Valid
}

func (m *Member) GetRefMember() *Member {
	if !m.ReferralUID.Valid {
		return nil
	}

	var member *Member

	config.DataBase.First(&member, "uid = ?", m.ReferralUID)

	return member
}
