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
	ID        uint64
	UUID      uuid.UUID
	IEOID     uint64 `gorm:"column:ieo_id"`
	MemberID  uint64
	Ask       string
	Bid       string
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	Bouns     decimal.Decimal
	State     OrderState
	CreatedAt time.Time
	UpdatedAt time.Time
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

func (o *IEOOrder) AskCurrency() *Currency {
	var currency *Currency

	config.DataBase.First(&currency, "id = ?", o.Ask)

	return currency
}

func (o *IEOOrder) BidCurrency() *Currency {
	var currency *Currency

	config.DataBase.First(&currency, "id = ?", o.Bid)

	return currency
}

func (o *IEOOrder) IncomeCurrency() *Currency {
	return o.AskCurrency()
}

func (o *IEOOrder) OutcomeCurrency() *Currency {
	return o.BidCurrency()
}

func (o *IEOOrder) BeforeSave(tx *gorm.DB) (err error) {
	o.TriggerEvent()

	return nil
}

func (o *IEOOrder) TriggerEvent() {
	if o.State == StatePending {
		return
	}

	member := o.Member()
	payload_message, _ := json.Marshal(o.ToJSON())

	mq_client.EnqueueEvent("private", member.UID, "ieo_order", payload_message)
}

func SubmitIEOOrder(id uint64) error {
	var outcome_account *Account
	var order *IEOOrder

	err := config.DataBase.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "orders"}}).Where("id = ?", id).First(&order)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("can't find ieo order by id : %d", order.ID)
		}

		if order.State != StatePending {
			return nil
		}

		account_tx := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "accounts"}})
		account_tx.Where("member_id = ? AND currency_id = ?", order.MemberID, order.OutcomeCurrency().ID).FirstOrCreate(&outcome_account)
		if err := outcome_account.SubFunds(account_tx, order.Total()); err != nil {
			return err
		}

		order.RecordSubmitOperations()

		order.State = StateWait
		tx.Save(&order)

		payload_ieo_order_executor_attrs, _ := json.Marshal(order)
		config.Nats.Publish("ieo_order_executor", payload_ieo_order_executor_attrs)

		// send to engine

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
		var income_account *Account
		var outcome_account *Account

		account_tx := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "accounts"}})
		account_tx.Where("member_id = ? AND currency_id = ?", o.MemberID, o.OutcomeCurrency().ID).FirstOrCreate(&outcome_account)
		account_tx.Where("member_id = ? AND currency_id = ?", o.MemberID, o.IncomeCurrency().ID).FirstOrCreate(&income_account)
		if err := outcome_account.UnlockAndSubFunds(account_tx, o.Total()); err != nil {
			return err
		}
		if err := income_account.PlusFunds(account_tx, o.Quantity.Add(o.Quantity.Mul(o.Bouns))); err != nil {
			return err
		}

		o.State = StateDone

		o.RecordCompleteOperations()

		tx.Save(&o)

		return nil
	})

	if err != nil {
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
	UUID      uuid.UUID       `json:"uuid"`
	IEOID     uint64          `json:"ieo_id"`
	MemberID  uint64          `json:"member_id"`
	Ask       string          `json:"ask"`
	Bid       string          `json:"bid"`
	Price     decimal.Decimal `json:"price"`
	Quantity  decimal.Decimal `json:"quantity"`
	Bouns     decimal.Decimal `json:"decimal.Decimal"`
	State     OrderState      `json:"state"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (o *IEOOrder) ToJSON() *IEOOrderJSON {
	return &IEOOrderJSON{
		UUID:      o.UUID,
		IEOID:     o.IEOID,
		MemberID:  o.MemberID,
		Ask:       o.Ask,
		Bid:       o.Bid,
		Price:     o.Price,
		Quantity:  o.Quantity,
		Bouns:     o.Bouns,
		State:     o.State,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}
