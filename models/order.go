package models

import (
	"encoding/json"
	"log"
	"math"
	"time"

	"github.com/ericlagergren/decimal"
	"github.com/google/uuid"
	"github.com/gookit/validate"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/matching"
	"github.com/zsmartex/go-finex/models/concerns"
	"github.com/zsmartex/go-finex/mq_client"
	"github.com/zsmartex/go-finex/types"
)

var precision_validator = &concerns.PrecisionValidator{}

type OrderSide string
type OrderState int32

const (
	StatePending OrderState = 0
	StateWait    OrderState = 100
	StateDone    OrderState = 200
	StateCancel  OrderState = -100
	StateReject  OrderState = -200
)

const (
	SideBuy  OrderSide = "OrderBid"
	SideSell OrderSide = "OrderAsk"
)

type Order struct {
	ID            uint64             `json:"id" gorm:"primaryKey"`
	UUID          uuid.UUID          `json:"uuid" gorm:"default:gen_random_uuid()"`
	MemberID      uint64             `json:"member_id" validate:"required"`
	Ask           string             `json:"ask" validate:"required"`
	Bid           string             `json:"bid" validate:"required"`
	RemoteId      string             `json:"remote_id" validate:"required"`
	Price         *float64           `json:"price" validate:"PriceVaildator"`
	StopPrice     *float64           `json:"stop_price"` // TODO: add vaildate
	Volume        float64            `json:"volume" validate:"required"`
	OriginVolume  float64            `json:"origin_volume" validate:"gt:0|OriginVolumeVaildator"`
	MakerFee      float64            `json:"maker_fee" gorm:"default:0.0"`
	TakerFee      float64            `json:"taker_fee" gorm:"default:0.0"`
	MarketID      string             `json:"market_id" validate:"required"`
	State         OrderState         `json:"state" validate:"required"`
	Type          OrderSide          `json:"type" validate:"required"`
	OrdType       matching.OrderType `json:"ord_type" validate:"MarketOrderVaildator"`
	Locked        float64            `json:"locked" gorm:"default:0.0"`
	OriginLocked  float64            `json:"origin_locked" gorm:"default:0.0"`
	FundsReceived float64            `json:"funds_received" gorm:"default:0.0"`
	TradesCount   uint64             `json:"trades_count" gorm:"default:0"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

func (o *Order) Message() map[string]string {
	invalid_message := "market.order.invalid_{field}"

	return validate.MS{
		"required": invalid_message,
	}
}

// func (o Order) Validate(err_src *Errors) {
// 	v := validate.Struct(o)

// 	if !v.Validate() {
// 		for _, errs := range v.Errors.All() {
// 			for _, err := range errs {
// 				errors = append(errors, err)
// 			}
// 		}
// 	}
// }

func (o *Order) PriceVaildator(Price float64) bool {
	market := o.Market()

	if !o.IsLimitOrder() {
		return false
	}

	if Price <= 0 {
		return false
	}

	if !precision_validator.LessThanOrEqTo(Price, market.PricePrecision) {
		return false
	}

	if Price > market.MaxPrice && market.MaxPrice != 0 || Price < market.MinPrice && market.MinPrice != 0 {
		return false
	}

	return true
}

func (o *Order) OriginVolumeVaildator(OriginVolume float64) bool {
	market := o.Market()

	if !precision_validator.LessThanOrEqTo(OriginVolume, market.AmountPrecision) {
		return false
	}

	if OriginVolume < market.MinAmount {
		return false
	}

	return true
}

func (o *Order) MarketOrderVaildator(ord_type string) bool {
	return o.Price != nil
}

func (o Order) Market() Market {
	var market Market

	config.DataBase.Find(&market, o.MarketID)

	return market
}

func (o *Order) Member() Member {
	var member Member

	config.DataBase.Find(&member, o.MemberID)

	return member
}

type DepthRow struct {
	Price  float64
	Amount float64
}

func GetDepth(side OrderSide, market string) [][]float64 {
	depth := make([][]float64, 0)
	depthl := make([]DepthRow, 0)
	tx := config.DataBase.Model(&Order{}).Select("price, sum(volume) as amount").Where("ord_type = ? AND state = ? AND type = ?", matching.TypeLimit, StateWait, side)

	switch side {
	case SideBuy:
		tx = tx.Order("price desc")
	default:
		tx = tx.Order("price asc")
	}

	tx.Group("price").Find(&depthl)

	for _, row := range depthl {
		depth = append(depth, []float64{row.Price, row.Amount})
	}

	return depth
}

func SubmitOrder(id uint64) {
	var order *Order

	err := config.DataBase.Transaction(func(tx *gorm.DB) error {
		tx.Clauses(clause.Locking{Strength: "UPDATE"})

		tx.Find(&order, id)
		if order.State != StatePending {
			return nil
		}

		order.HoldAccount(tx).LockFunds(tx, order.Locked)
		order.RecordSubmitOperations()

		tx.Update("state", StateWait)

		return nil
	})

	if err != nil {
		config.DataBase.Find(&order, id)
		if order != nil {
			config.DataBase.Updates(Order{State: StateReject})
		}

		log.Println(err)
	}
}

func CancelOrder(id uint64) {
	var order *Order

	err := config.DataBase.Transaction(func(tx *gorm.DB) error {
		tx.Clauses(clause.Locking{Strength: "UPDATE"})

		tx.Where("id = ?", id).Find(&order)
		if order.State != StateWait {
			return nil
		}

		order.HoldAccount(tx).UnlockFunds(tx, order.Locked)
		order.RecordCancelOperations()

		tx.Update("state", StateCancel)

		return nil
	})

	if err != nil {
		config.DataBase.Find(&order, id)
		if order != nil {
			config.DataBase.Updates(Order{State: StateReject})
		}

		log.Println(err)
	}
}

// Submit order to matching engine
func (o *Order) Submit(err_src *[]string) {
	LOCKING_BUFFER_FACTOR := 1.1
	var locked float64
	member_balance := o.MemberBalance()

	if o.OrdType == matching.TypeMarket && o.Type == SideBuy {
		locked = math.Min(o.ComputeLocked()*LOCKING_BUFFER_FACTOR, member_balance)
	} else {
		locked = o.ComputeLocked()
	}

	if member_balance >= locked {
		*err_src = append(*err_src, "market.account.insufficient_balance")
	}

	o.Locked = locked
	o.OriginLocked = locked

	config.DataBase.Save(&o)

	order_processor_payload, _ := json.Marshal(types.OrderProcessorPayloadMessage{
		Action: types.ActionSubmit,
		Order:  o.ToMatchingAttributes(),
	})

	if len(*err_src) > 0 {
		return
	}

	mq_client.Enqueue("order_processor", order_processor_payload)
}

func (o *Order) TriggerEvent() {

}

func (o *Order) RecordSubmitOperations() {
	LiabilityTranfer(
		o.Locked,
		o.Currency(),
		Reference{
			ID:   o.ID,
			Type: string(o.Type),
		},
		"main",
		"locked",
		o.MemberID,
	)
}

func (o Order) RecordCancelOperations() {
	LiabilityTranfer(
		o.Locked,
		o.Currency(),
		Reference{
			ID:   o.ID,
			Type: string(o.Type),
		},
		"locked",
		"main",
		o.MemberID,
	)
}

func (o *Order) AskCurrency() Currency {
	var currency Currency

	config.DataBase.Where("code = ?", o.Ask).Find(&currency, 1)

	return currency
}

func (o *Order) BidCurrency() Currency {
	var currency Currency

	config.DataBase.Where("code = ?", o.Bid).Find(&currency, 1)

	return currency
}

func (o *Order) IncomeCurrency() Currency {
	switch o.Type {
	case SideBuy:
		return o.AskCurrency()
	default:
		return o.BidCurrency()
	}
}

func (o *Order) OutcomeCurrency() Currency {
	switch o.Type {
	case SideBuy:
		return o.BidCurrency()
	default:
		return o.AskCurrency()
	}
}

func (o *Order) Currency() Currency {
	switch o.Type {
	case SideBuy:
		return o.AskCurrency()
	default:
		return o.BidCurrency()
	}
}

func (o *Order) MemberBalance() float64 {
	return o.Member().GetAccount(o.Currency()).Balance
}

func (o *Order) HoldAccount(tx *gorm.DB) *Account {
	var account *Account

	switch o.Type {
	case SideBuy:
		tx.Where("member_id = ? AND bid = ?", o.MemberID, o.Bid).FirstOrCreate(&account)
	case SideSell:
		tx.Where("member_id = ? AND ask = ?", o.MemberID, o.Ask).FirstOrCreate(&account)
	}

	return account
}

func (o *Order) ComputeLocked() float64 {
	switch o.OrdType {
	case matching.TypeLimit:
		switch o.Type {
		case SideBuy:
			return *o.Price * o.Volume
		default:
			return o.Volume
		}
	default:
		switch o.Type {
		case SideBuy:
			return o.Volume
		default:
			return 0.0
		}
	}
}

func (o *Order) IsLimitOrder() bool {
	return o.OrdType == "limit"
}

func (o *Order) IsMarketOrder() bool {
	return o.OrdType == "market"
}

func (o *Order) IsCancelled() bool {
	return o.State == StateCancel
}

func (o *Order) FundsUsed() float64 {
	return o.OriginLocked - o.Locked
}

func (o *Order) AvgPrice() float64 {
	if o.Type == SideSell && o.FundsUsed() == 0 || o.Type == SideBuy && o.FundsReceived == 0 {
		return 0
	}

	if o.Type == SideSell {
		return o.Market().round_price(o.FundsReceived / o.FundsUsed())
	} else {
		return o.Market().round_price(o.FundsUsed() / o.FundsReceived)
	}
}

func (o *Order) Side() string {
	switch o.Type {
	case SideBuy:
		return TypeBuy
	default:
		return TypeSell
	}
}

func EstimateRequiredFunds() {

}

type OrderUserJSON struct {
	Id              uint64             `json:"id"`
	Uuid            uuid.UUID          `json:"uuid"`
	Market          string             `json:"market"`
	Side            string             `json:"side"`
	OrdType         matching.OrderType `json:"ord_type"`
	Price           *float64           `json:"price"`
	AvgPrice        float64            `json:"avg_price"`
	State           string             `json:"state"`
	OriginVolume    float64            `json:"origin_volume"`
	RemainingVolume float64            `json:"remaining_volume"`
	ExecutedVolume  float64            `json:"executed_volume"`
	TradesCount     uint64             `json:"trades_count"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

func (o *Order) ToJSON() OrderUserJSON {
	var StateString string
	var SideString string

	switch o.State {
	case StatePending:
		StateString = "pending"
	case StateWait:
		StateString = "wait"
	case StateDone:
		StateString = "done"
	case StateCancel:
		StateString = "cancel"
	case StateReject:
		StateString = "reject"
	}

	switch o.Type {
	case SideBuy:
		SideString = "buy"
	case SideSell:
		SideString = "sell"
	}

	return OrderUserJSON{
		Id:              o.ID,
		Uuid:            o.UUID,
		Market:          o.MarketID,
		Side:            SideString,
		OrdType:         o.OrdType,
		Price:           o.Price,
		AvgPrice:        o.AvgPrice(),
		State:           StateString,
		OriginVolume:    o.OriginVolume,
		RemainingVolume: o.Volume,
		ExecutedVolume:  (o.OriginVolume - o.Volume),
		TradesCount:     o.TradesCount,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}
}

func (o *Order) ToMatchingAttributes() matching.Order {
	var Params matching.OrderParams

	if IsStopLimitOrder := o.StopPrice != nil; IsStopLimitOrder {
		Params = matching.ParamStop
	} else {
		Params = 0
	}

	decimalCtx := decimal.Big{}

	return matching.Order{
		ID:         o.ID,
		Instrument: o.MarketID,
		CustomerID: o.MemberID,

		Type:      o.OrdType,
		Params:    Params,
		Qty:       *decimalCtx.SetFloat64(o.OriginVolume),
		FilledQty: *decimalCtx.SetFloat64(o.Volume),
		Price:     *decimalCtx.SetFloat64(*o.Price),
		StopPrice: *decimalCtx.SetFloat64(*o.StopPrice),
		Side:      o.Type == SideBuy, // true for Bid || Buy
		Cancelled: o.IsCancelled(),

		Timestamp: o.CreatedAt,
	}
}
