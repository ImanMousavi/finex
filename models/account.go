package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/zsmartex/go-finex/config"
	"gorm.io/gorm"
)

var Zero float64 = 0

type Account struct {
	ID         uint64    `json:"id" gorm:"primaryKey"`
	MemberID   uint64    `json:"member_id"`
	CurrencyID string    `json:"currency_id"`
	Balance    float64   `json:"balance" gorm:"default:0.0" validate:"min:0"`
	Locked     float64   `json:"lock" gorm:"default:0.0" validate:"min:0"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (a *Account) Currency() Currency {
	var currency Currency

	config.DataBase.Model(&Market{}).Find(&currency, a.CurrencyID)

	return currency
}

func (a *Account) Member() Member {
	var member Member

	config.DataBase.Model(&Member{}).Find(&member, a.CurrencyID)

	return member
}

func (a *Account) BeforeSave(tx *gorm.DB) (err error) {
	return
}

func (a *Account) PlusFunds(tx *gorm.DB, amount float64) error {
	if amount <= Zero {
		return errors.New("Cannot add funds (member id: " + string(a.MemberID) + ", currency id: " + string(a.CurrencyID) + ", amount: " + fmt.Sprintf("%f", amount) + ", balance: " + fmt.Sprintf("%f", a.Balance) + ").")
	}

	tx.Model(&a).Updates(Account{
		Balance: a.Balance + amount,
	})

	return nil
}

func (a *Account) PlusLockedFunds(tx *gorm.DB, amount float64) error {
	if amount <= Zero {
		return errors.New("Cannot add funds (member id: " + string(a.MemberID) + ", currency id: " + string(a.CurrencyID) + ", amount: " + fmt.Sprintf("%f", amount) + ", locked: " + fmt.Sprintf("%f", a.Locked) + ").")
	}

	tx.Model(&a).Updates(Account{
		Locked: a.Locked + amount,
	})

	return nil
}

func (a *Account) SubFunds(tx *gorm.DB, amount float64) error {
	if amount <= Zero || amount > a.Balance {
		return errors.New("Cannot subtract funds (member id: " + string(a.MemberID) + ", currency id: " + string(a.CurrencyID) + ", amount: " + fmt.Sprintf("%f", amount) + ", balance: " + fmt.Sprintf("%f", a.Balance) + ").")
	}

	tx.Model(&a).Updates(Account{
		Balance: a.Balance - amount,
	})

	return nil
}

func (a *Account) LockFunds(tx *gorm.DB, amount float64) error {
	if amount <= Zero || amount > a.Balance {
		return errors.New("Cannot lock funds (member id: " + string(a.MemberID) + ", currency id: " + string(a.CurrencyID) + ", amount: " + fmt.Sprintf("%f", amount) + ", balance: " + fmt.Sprintf("%f", a.Balance) + ", locked: " + fmt.Sprintf("%f", a.Locked) + ").")
	}

	tx.Model(&a).Updates(Account{
		Balance: a.Balance - amount,
		Locked:  a.Locked + amount,
	})

	return nil
}

func (a *Account) UnlockFunds(tx *gorm.DB, amount float64) error {
	if amount <= Zero || amount > a.Locked {
		return errors.New("Cannot unlock funds (member id: " + string(a.MemberID) + ", currency id: " + string(a.CurrencyID) + ", amount: " + fmt.Sprintf("%f", amount) + ", balance: " + fmt.Sprintf("%f", a.Balance) + ", locked: " + fmt.Sprintf("%f", a.Locked) + ").")
	}

	tx.Model(&a).Updates(Account{
		Balance: a.Balance + amount,
		Locked:  a.Locked - amount,
	})

	return nil
}

func (a *Account) UnlockAndSubFunds(tx *gorm.DB, amount float64) error {
	if amount <= Zero || amount > a.Locked {
		return errors.New("Cannot unlock funds (member id: " + string(a.MemberID) + ", currency id: " + string(a.CurrencyID) + ", amount: " + fmt.Sprintf("%f", amount) + ", locked: " + fmt.Sprintf("%f", a.Locked) + ").")
	}

	tx.Model(&a).Updates(Account{
		Locked: a.Locked - amount,
	})

	return nil
}

func (a *Account) Amount() float64 {
	return a.Balance + a.Locked
}
