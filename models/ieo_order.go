package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/mq_client"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IEOOrder struct {
	ID                int64
	UUID              uuid.UUID
	IEOID             int64 `gorm:"column:ieo_id"`
	MemberID          int64
	PaymentCurrencyID string
	Price             decimal.Decimal
	Quantity          decimal.Decimal
	State             OrderState
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (IEOOrder) TableName() string {
	return "ieo_orders"
}

func (o *IEOOrder) MemberBalance() decimal.Decimal {
	return o.Member().GetAccount(o.OutcomeCurrency()).Balance
}

func (o *IEOOrder) Member() *Member {
	var member *Member

	config.DataBase.First(&member, o.MemberID)

	return member
}

func (o *IEOOrder) Total() decimal.Decimal {
	return o.Price.Mul(o.Quantity)
}

func (o *IEOOrder) IncomeCurrency() *Currency {
	var ieo *IEO
	var currency *Currency

	config.DataBase.First(&ieo, o.IEOID)

	config.DataBase.First(&currency, "id = ?", ieo.CurrencyID)

	return currency
}

func (o *IEOOrder) OutcomeCurrency() *Currency {
	var currency *Currency

	config.DataBase.First(&currency, "id = ?", o.PaymentCurrencyID)

	return currency
}

func SubmitIEOOrder(id int64) error {
	var outcome_account *Account
	var order *IEOOrder

	err := config.DataBase.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "ieo_orders"}}).Where("id = ?", id).First(&order)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("can't find ieo order by id : %d", order.ID)
		}

		if order.State != StatePending {
			return nil
		}

		account_tx := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "accounts"}})
		account_tx.Where("member_id = ? AND currency_id = ?", order.MemberID, order.OutcomeCurrency().ID).FirstOrCreate(&outcome_account)
		if err := outcome_account.LockFunds(account_tx, order.Total()); err != nil {
			return err
		}

		order.RecordSubmitOperations()

		order.State = StateWait
		tx.Save(&order)

		payload_ieo_order_executor_attrs, _ := json.Marshal(order.ToJSON())
		config.Kafka.Publish("ieo_order_executor", payload_ieo_order_executor_attrs)

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (o *IEOOrder) RecordSubmitOperations() {
	LiabilityTranfer(
		o.Total(),
		o.OutcomeCurrency(),
		Reference{
			ID:   o.ID,
			Type: "IEOOrder",
		},
		"main",
		"locked",
		o.MemberID,
	)
}

func (o *IEOOrder) Strike() error {
	err := config.DataBase.Transaction(func(tx *gorm.DB) error {
		var accounts []*Account
		var ieo *IEO
		var member *Member

		accounts_table := make(map[string]*Account)

		config.DataBase.First(&member, o.MemberID)
		config.DataBase.First(&ieo, o.IEOID)

		member.GetAccount(o.OutcomeCurrency())
		member.GetAccount(o.IncomeCurrency())

		tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "accounts"},
		}).Where(
			"member_id = ? AND currency_id IN ?",
			o.MemberID,
			[]string{
				o.OutcomeCurrency().ID,
				o.IncomeCurrency().ID,
			},
		).Find(&accounts)

		for _, account := range accounts {
			accounts_table[account.CurrencyID] = account
		}

		if err := accounts_table[o.OutcomeCurrency().ID].UnlockAndSubFunds(tx, o.Total()); err != nil {
			return err
		}

		if err := accounts_table[o.IncomeCurrency().ID].PlusFunds(tx, o.Quantity); err != nil {
			return err
		}

		o.State = StateDone

		o.RecordCompleteOperations()

		ieo.ExecutedQuantity = ieo.ExecutedQuantity.Add(o.Quantity)

		tx.Save(&o)
		tx.Save(&ieo)

		payload_msg, _ := json.Marshal(o.ToJSON())
		mq_client.EnqueueEvent("private", member.UID, "ieo", payload_msg)

		return nil
	})

	if err != nil {
		config.DataBase.Transaction(func(tx *gorm.DB) error {
			var outcome_account *Account

			account_tx := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "accounts"}})
			account_tx.Where("member_id = ? AND currency_id = ?", o.MemberID, o.OutcomeCurrency().ID).FirstOrCreate(&outcome_account)

			o.State = StateReject

			outcome_account.UnlockFunds(account_tx, o.Total())
			tx.Save(&o)

			return nil
		})

		return err
	}

	return nil
}

func (o *IEOOrder) RecordCompleteOperations() {
	reference := Reference{
		ID:   o.ID,
		Type: "IEOOrder",
	}

	LiabilityDebit(
		o.Total(),
		o.OutcomeCurrency(),
		reference,
		"locked",
		o.MemberID,
	)

	LiabilityDebit(
		o.Quantity,
		o.IncomeCurrency(),
		reference,
		"main",
		o.MemberID,
	)
}

type IEOOrderJSON struct {
	ID                int64           `json:"id"`
	UUID              uuid.UUID       `json:"uuid"`
	IEOID             int64           `json:"ieo_id"`
	PaymentCurrencyID string          `json:"payment_currency_id"`
	Price             decimal.Decimal `json:"price"`
	Quantity          decimal.Decimal `json:"quantity"`
	State             OrderState      `json:"state"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (o *IEOOrder) ToJSON() *IEOOrderJSON {
	return &IEOOrderJSON{
		ID:                o.ID,
		UUID:              o.UUID,
		IEOID:             o.IEOID,
		PaymentCurrencyID: o.PaymentCurrencyID,
		Price:             o.Price,
		Quantity:          o.Quantity,
		State:             o.State,
		CreatedAt:         o.CreatedAt,
		UpdatedAt:         o.UpdatedAt,
	}
}
