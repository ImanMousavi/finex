package models

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
)

type Revenue struct {
	ID            int64           `json:"id"`
	Code          int32           `json:"code"`
	CurrencyID    string          `json:"currency_id"`
	MemberID      int64           `json:"member_id"`
	ReferenceType string          `json:"reference_type"`
	ReferenceID   int64           `json:"reference_id"`
	Debit         decimal.Decimal `json:"debit"`
	Credit        decimal.Decimal `json:"credit"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func GetRevenueCode(currency *Currency) int32 {
	var operations_account OperationsAccount
	config.DataBase.Where("type = ? AND currency_type = ?", TypeRevenue, currency.Type).Find(&operations_account)

	return operations_account.Code
}

func RevenueCredit(amount decimal.Decimal, currency *Currency, reference Reference, member_id int64) {
	code := GetRevenueCode(currency)

	revenue := Revenue{
		Code:          code,
		CurrencyID:    currency.ID,
		ReferenceType: reference.Type,
		ReferenceID:   reference.ID,
		Credit:        amount,
		MemberID:      member_id,
	}

	config.DataBase.Create(&revenue)
}

func RevenueDebit(amount decimal.Decimal, currency *Currency, reference Reference, member_id int64) {
	code := GetRevenueCode(currency)

	revenue := Revenue{
		Code:          code,
		CurrencyID:    currency.ID,
		ReferenceType: reference.Type,
		ReferenceID:   reference.ID,
		Debit:         amount,
		MemberID:      member_id,
	}

	config.DataBase.Create(&revenue)
}

func RevenueTranfer(amount decimal.Decimal, currency *Currency, reference Reference, from_kind, to_kind string, member_id int64) {
	RevenueCredit(amount, currency, reference, member_id)
	RevenueDebit(amount, currency, reference, member_id)
}
