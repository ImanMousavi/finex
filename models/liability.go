package models

import (
	"time"

	"gitlab.com/zsmartex/finex/config"
)

type Liability struct {
	ID            uint64    `json:"id"`
	Code          int32     `json:"code"`
	CurrencyID    string    `json:"currency_id"`
	MemberID      uint64    `json:"member_id"`
	ReferenceType string    `json:"reference_type"`
	ReferenceID   uint64    `json:"reference_id"`
	Debit         float64   `json:"debit"`
	Credit        float64   `json:"credit"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func GetOperationsCode(currency Currency, kind string) int32 {
	var operations_account OperationsAccount
	config.DataBase.Where("type = ? AND kind = ? AND currency_type = ?", TypeLiability, kind, currency.Type).Find(&operations_account)

	return operations_account.Code
}

func LiabilityCredit(amount float64, currency Currency, reference Reference, kind string, member_id uint64) {
	code := GetOperationsCode(currency, kind)

	liability := Liability{
		Code:          code,
		CurrencyID:    currency.ID,
		ReferenceType: reference.Type,
		ReferenceID:   reference.ID,
		Credit:        amount,
		MemberID:      member_id,
	}

	config.DataBase.Create(&liability)
}

func LiabilityDebit(amount float64, currency Currency, reference Reference, kind string, member_id uint64) {
	code := GetOperationsCode(currency, kind)

	liability := Liability{
		Code:          code,
		CurrencyID:    currency.ID,
		ReferenceType: reference.Type,
		ReferenceID:   reference.ID,
		Debit:         amount,
		MemberID:      member_id,
	}

	config.DataBase.Create(&liability)
}

func LiabilityTranfer(amount float64, currency Currency, reference Reference, from_kind, to_kind string, member_id uint64) {
	LiabilityCredit(amount, currency, reference, from_kind, member_id)
	LiabilityDebit(amount, currency, reference, to_kind, member_id)
}
