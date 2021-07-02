package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/mq_client"
	"gorm.io/gorm"
)

var Zero float64 = 0

type Account struct {
	ID         uint64          `json:"id" gorm:"primaryKey"`
	MemberID   uint64          `json:"member_id"`
	CurrencyID string          `json:"currency_id"`
	Balance    decimal.Decimal `json:"balance" validate:"ValidateBalance"`
	Locked     decimal.Decimal `json:"locked" validate:"ValidateLocked"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (a Account) ValidateBalance(Balance decimal.Decimal) bool {
	return Balance.GreaterThanOrEqual(decimal.Zero)
}

func (a Account) ValidateLocked(Locked decimal.Decimal) bool {
	return Locked.GreaterThanOrEqual(decimal.Zero)
}

func (a *Account) Currency() *Currency {
	currency := &Currency{}

	config.DataBase.First(currency, "id = ?", a.CurrencyID)

	return currency
}

func (a *Account) Member() *Member {
	member := &Member{}

	config.DataBase.First(member, "id = ?", a.MemberID)

	return member
}

func (a *Account) BeforeSave(tx *gorm.DB) (err error) {
	a.TriggerEvent()

	return
}

func (a *Account) TriggerEvent() {
	member := a.Member()
	payload_message, _ := json.Marshal(a.ToJSON())

	mq_client.EnqueueEvent("private", member.UID, "balance", payload_message)
}

func (a *Account) PlusFunds(tx *gorm.DB, amount decimal.Decimal) error {
	if !amount.IsPositive() {
		return fmt.Errorf("cannot add funds (member id: %d, currency id: %s, amount: %s, balance: %s)", a.MemberID, a.CurrencyID, amount.String(), a.Balance.String())
	}

	a.Balance = a.Balance.Add(amount)
	return tx.Save(&a).Error
}

func (a *Account) PlusLockedFunds(tx *gorm.DB, amount decimal.Decimal) error {
	if !amount.IsPositive() {
		return fmt.Errorf("cannot add funds (member id: %d, currency id: %s, amount: %s, locked: %s)", a.MemberID, a.CurrencyID, amount.String(), a.Locked.String())
	}

	a.Locked = a.Locked.Add(amount)
	return tx.Save(&a).Error
}

func (a *Account) SubFunds(tx *gorm.DB, amount decimal.Decimal) error {
	if !amount.IsPositive() || amount.GreaterThan(a.Balance) {
		return fmt.Errorf("cannot subtract funds (member id: %d, currency id: %s, amount: %s, balance: %s)", a.MemberID, a.CurrencyID, amount.String(), a.Balance.String())
	}

	a.Balance = a.Balance.Sub(amount)
	return tx.Save(&a).Error
}

func (a *Account) LockFunds(tx *gorm.DB, amount decimal.Decimal) error {
	if !amount.IsPositive() || amount.GreaterThan(a.Balance) {
		return fmt.Errorf("cannot lock funds (member id: %d, currency id: %s, amount: %s, balance: %s, locked: %s)", a.MemberID, a.CurrencyID, amount.String(), a.Balance.String(), a.Locked.String())
	}

	a.Balance = a.Balance.Sub(amount)
	a.Locked = a.Locked.Add(amount)
	return tx.Save(&a).Error
}

func (a *Account) UnlockFunds(tx *gorm.DB, amount decimal.Decimal) error {
	if !amount.IsPositive() || amount.GreaterThan(a.Locked) {
		return fmt.Errorf("cannot unlock funds (member id: %d, currency id: %s, amount: %s, balance: %s, locked: %s)", a.MemberID, a.CurrencyID, amount.String(), a.Balance.String(), a.Locked.String())
	}

	a.Balance = a.Balance.Add(amount)
	a.Locked = a.Locked.Sub(amount)
	return tx.Save(&a).Error
}

func (a *Account) UnlockAndSubFunds(tx *gorm.DB, amount decimal.Decimal) error {
	if !amount.IsPositive() || amount.GreaterThan(a.Locked) {
		return fmt.Errorf("cannot unlock and sub funds (member id: %d, currency id: %s, amount: %s, balance: %s, locked: %s)", a.MemberID, a.CurrencyID, amount.String(), a.Balance.String(), a.Locked.String())
	}

	a.Locked = a.Locked.Sub(amount)
	return tx.Save(&a).Error
}

func (a *Account) Amount() decimal.Decimal {
	return a.Balance.Add(a.Locked)
}

type AccountJSON struct {
	Currency string          `json:"currency"`
	Balance  decimal.Decimal `json:"balance"`
	Locked   decimal.Decimal `json:"locked"`
}

func (a *Account) ToJSON() AccountJSON {
	return AccountJSON{
		Currency: a.CurrencyID,
		Balance:  a.Balance,
		Locked:   a.Locked,
	}
}
