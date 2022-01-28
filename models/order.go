package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gookit/validate"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/entities"
	"github.com/zsmartex/finex/models/concerns"
	"github.com/zsmartex/finex/mq_client"
	"github.com/zsmartex/finex/types"
	"github.com/zsmartex/pkg"
	"github.com/zsmartex/pkg/order"
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
	ID            int64               `json:"id" gorm:"primaryKey"`
	UUID          uuid.UUID           `json:"uuid" gorm:"default:gen_random_uuid()"`
	MemberID      int64               `json:"member_id" validate:"required"`
	Ask           string              `json:"ask" validate:"required"`
	Bid           string              `json:"bid" validate:"required"`
	RemoteId      sql.NullString      `json:"remote_id"`
	Price         decimal.NullDecimal `json:"price" validate:"PriceVaildator"`
	StopPrice     decimal.NullDecimal `json:"stop_price" validate:"StopPriceVaildator"`
	Volume        decimal.Decimal     `json:"volume" validate:"required"`
	OriginVolume  decimal.Decimal     `json:"origin_volume" validate:"OriginVolumeVaildator"`
	MakerFee      decimal.Decimal     `json:"maker_fee" gorm:"default:0.0"`
	TakerFee      decimal.Decimal     `json:"taker_fee" gorm:"default:0.0"`
	MarketID      string              `json:"market_id" validate:"required"`
	MarketType    types.AccountType   `json:"market_type" gorm:"default:spot" validate:"MarketTypeVaildator"`
	State         OrderState          `json:"state"`
	Type          OrderSide           `json:"type" validate:"required"`
	OrdType       types.OrderType     `json:"ord_type" validate:"OrdTypeVaildator"`
	Locked        decimal.Decimal     `json:"locked" gorm:"default:0.0"`
	OriginLocked  decimal.Decimal     `json:"origin_locked" gorm:"default:0.0"`
	FundsReceived decimal.Decimal     `json:"funds_received" gorm:"default:0.0"`
	TradesCount   int64               `json:"trades_count" gorm:"default:0"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

func (o Order) Message() map[string]string {
	invalid_message := "market.order.invalid_{field}"

	return validate.MS{
		"required": invalid_message,
	}
}

func (o Order) PriceVaildator(Price decimal.NullDecimal) bool {
	if o.OrdType == types.TypeMarket {
		return true // skip
	}

	dPrice := Price.Decimal

	market := o.Market()
	PricePrecision := int32(market.PricePrecision)

	if dPrice.LessThanOrEqual(decimal.Zero) {
		return false
	}

	if !precision_validator.LessThanOrEqTo(dPrice, PricePrecision) {
		return false
	}

	if dPrice.GreaterThan(market.MaxPrice) && market.MaxPrice.IsPositive() || dPrice.LessThan(market.MinPrice) && market.MinPrice.IsPositive() {
		return false
	}

	return true
}

func (o Order) StopPriceVaildator(StopPrice decimal.NullDecimal) bool {
	if o.OrdType == types.TypeMarket {
		return true // skip
	}

	dStopPrice := StopPrice.Decimal

	market := o.Market()
	PricePrecision := int32(market.PricePrecision)

	if dStopPrice.LessThanOrEqual(decimal.Zero) {
		return false
	}

	if !precision_validator.LessThanOrEqTo(dStopPrice, PricePrecision) {
		return false
	}

	if dStopPrice.GreaterThan(market.MaxPrice) && market.MaxPrice.IsPositive() || dStopPrice.LessThan(market.MinPrice) && market.MinPrice.IsPositive() {
		return false
	}

	return true
}

func (o Order) OriginVolumeVaildator(OriginVolume decimal.Decimal) bool {
	market := o.Market()
	AmountPrecision := int32(market.AmountPrecision)

	if !precision_validator.LessThanOrEqTo(OriginVolume, AmountPrecision) {
		return false
	}

	if OriginVolume.LessThan(market.MinAmount) {
		return false
	}

	return true
}

func (o Order) OrdTypeVaildator(ord_type types.OrderType) bool {
	if o.OrdType == types.TypeMarket {
		return !o.Price.Valid && !o.StopPrice.Valid
	}

	return true
}

func (o Order) MarketTypeVaildator(market_type types.AccountType) bool {
	supported_market_types := []types.AccountType{types.AccountTypeSpot, types.AccountTypeMargin, types.AccountTypeFutures}

	for _, t := range supported_market_types {
		if t == market_type {
			return true
		}
	}

	return false
}

func (o *Order) Market() *Market {
	market := &Market{}

	config.DataBase.First(market, "symbol = ?", o.MarketID)

	return market
}

func (o *Order) Member() *Member {
	var member *Member

	config.DataBase.First(&member, o.MemberID)

	return member
}

type PriceLevel struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
}

func GetDepth(side OrderSide, market string) [][]decimal.Decimal {
	depth := make([][]decimal.Decimal, 0)
	price_levels := make([]PriceLevel, 0)
	tx := config.DataBase.Model(&Order{}).Select("price, sum(volume) as amount").Where("market_id = ? AND ord_type = ? AND state = ? AND type = ?", market, types.TypeLimit, StateWait, side)

	switch side {
	case SideBuy:
		tx = tx.Order("price desc")
	default:
		tx = tx.Order("price asc")
	}

	tx.Group("price").Find(&price_levels)

	for _, row := range price_levels {
		depth = append(depth, []decimal.Decimal{row.Price, row.Amount})
	}

	return depth
}

func SubmitOrder(id int64) error {
	var account *Account
	var order *Order

	err := config.DataBase.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "orders"}}).Where("id = ?", id).First(&order)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("can't find order by id : %d", order.ID)
		}

		if order.State != StatePending {
			return nil
		}

		account_tx := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "accounts"}})
		account_tx.Where("member_id = ? AND currency_id = ?", order.MemberID, order.Currency().ID).FirstOrCreate(&account)
		if err := account.LockFunds(account_tx, order.Locked); err != nil {
			return err
		}

		order.RecordSubmitOperations()

		order.State = StateWait
		tx.Save(order)

		payload_matching_attrs, _ := json.Marshal(map[string]interface{}{
			"action": pkg.ActionSubmit,
			"order":  order.ToMatchingAttributes(),
		})
		config.Kafka.Publish("matching", payload_matching_attrs)

		return nil
	})

	if err != nil {
		result := config.DataBase.Where("id = ?", id).First(&order)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return err
		}

		order.State = StateReject
		config.DataBase.Save(&order)
	}

	return nil
}

func CancelOrder(id int64) error {
	var account *Account
	var order *Order

	err := config.DataBase.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "orders"}}).Where("id = ?", id).First(&order)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("can't find order by id : %d", order.ID)
		}

		if order.State != StateWait {
			return nil
		}

		account_tx := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "accounts"}})
		account_tx.Where("member_id = ? AND currency_id = ?", order.MemberID, order.Currency().ID).FirstOrCreate(&account)
		if err := account.UnlockFunds(tx, order.Locked); err != nil {
			return err
		}

		order.RecordCancelOperations()

		order.State = StateCancel
		tx.Save(order)

		return nil
	})

	return err
}

// Submit order to matching engine
func (o *Order) Submit() error {
	member_balance := o.MemberBalance()

	if member_balance.LessThan(o.Locked) {
		return errors.New("market.account.insufficient_balance")
	}

	config.DataBase.Save(&o)

	order_processor_payload, _ := json.Marshal(map[string]interface{}{
		"action": pkg.ActionSubmit,
		"id":     o.ID,
	})

	config.Kafka.Publish("order_processor", order_processor_payload)
	return nil
}

func (o *Order) BeforeSave(tx *gorm.DB) (err error) {
	o.TriggerEvent()

	return nil
}

func (o *Order) TriggerEvent() {
	if o.State == StatePending {
		return
	}

	member := o.Member()
	payload_message, _ := json.Marshal(o.ToJSON())

	mq_client.EnqueueEvent("private", member.UID, "order", payload_message)
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

func (o *Order) AskCurrency() *Currency {
	var currency *Currency

	config.DataBase.First(&currency, "id = ?", o.Ask)

	return currency
}

func (o *Order) BidCurrency() *Currency {
	var currency *Currency

	config.DataBase.First(&currency, "id = ?", o.Bid)

	return currency
}

func (o *Order) IncomeCurrency() *Currency {
	switch o.Type {
	case SideBuy:
		return o.AskCurrency()
	default:
		return o.BidCurrency()
	}
}

func (o *Order) OutcomeCurrency() *Currency {
	switch o.Type {
	case SideBuy:
		return o.BidCurrency()
	default:
		return o.AskCurrency()
	}
}

func (o *Order) Currency() *Currency {
	switch o.Type {
	case SideBuy:
		return o.BidCurrency()
	default:
		return o.AskCurrency()
	}
}

func (o *Order) MemberBalance() decimal.Decimal {
	return o.Member().GetAccount(o.Currency()).Balance
}

func (o *Order) ComputeLocked() (decimal.Decimal, error) {
	if o.OrdType == types.TypeLimit {
		if o.Type == SideBuy {
			return o.Price.Decimal.Mul(o.Volume), nil
		} else {
			return o.Volume, nil
		}
	} else if o.OrdType == types.TypeMarket {
		required_funds := decimal.Zero
		expected_volume := o.Volume

		var price_levels [][]decimal.Decimal
		if o.Type == SideBuy {
			price_levels = GetDepth(SideSell, o.MarketID)
		} else {
			price_levels = GetDepth(SideBuy, o.MarketID)
		}

		for !expected_volume.IsZero() && len(price_levels) != 0 {
			i := len(price_levels) - 1
			pl := price_levels[i]
			price_levels = append(price_levels[:i], price_levels[i+1:]...)

			level_price := pl[0]
			level_volume := pl[1]

			v := decimal.Min(expected_volume, level_volume)
			if o.Type == SideBuy {
				required_funds = required_funds.Add(level_price.Mul(v))
			} else {
				required_funds = required_funds.Add(v)
			}
			expected_volume = expected_volume.Sub(v)
		}

		if !expected_volume.IsZero() {
			return decimal.Zero, errors.New("market.order.insufficient_market_liquidity")
		}
		return required_funds, nil
	}

	return decimal.Zero, nil
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

func (o *Order) FundsUsed() decimal.Decimal {
	return o.OriginLocked.Sub(o.Locked)
}

func (o *Order) AvgPrice() decimal.Decimal {
	if o.Type == SideSell && o.FundsUsed().IsZero() || o.Type == SideBuy && o.FundsReceived.IsZero() {
		return decimal.Zero
	}

	market := o.Market()

	if o.Type == SideSell {
		return market.round_price(o.FundsReceived.Div(o.FundsUsed()))
	} else {
		return market.round_price(o.FundsUsed().Div(o.FundsReceived))
	}
}

func (o *Order) Side() types.TakerType {
	switch o.Type {
	case SideBuy:
		return types.TypeBuy
	default:
		return types.TypeSell
	}
}

func EstimateRequiredFunds() {

}

func (o *Order) ToJSON() entities.OrderEntity {
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

	return entities.OrderEntity{
		UUID:            o.UUID,
		Market:          o.MarketID,
		Side:            SideString,
		OrdType:         o.OrdType,
		Price:           o.Price,
		StopPrice:       o.StopPrice,
		AvgPrice:        o.AvgPrice(),
		State:           StateString,
		OriginVolume:    o.OriginVolume,
		RemainingVolume: o.Volume,
		ExecutedVolume:  o.OriginVolume.Sub(o.Volume),
		TradesCount:     o.TradesCount,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}
}

func FloatToString(input_num float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(input_num, 'f', 6, 64)
}

func (o *Order) ToMatchingAttributes() *order.Order {
	var side order.OrderSide
	if o.Type == SideBuy {
		side = order.SideBuy
	} else {
		side = order.SideSell
	}

	var orderType order.OrderType
	if o.OrdType == types.TypeLimit {
		orderType = order.TypeLimit
	} else if o.OrdType == types.TypeMarket {
		orderType = order.TypeMarket
	}

	return &order.Order{
		ID:             o.ID,
		UUID:           o.UUID,
		Symbol:         o.MarketID,
		MemberID:       o.MemberID,
		Side:           side,
		Type:           orderType,
		Price:          o.Price.Decimal,
		StopPrice:      o.StopPrice.Decimal,
		Quantity:       o.OriginVolume,
		FilledQuantity: o.OriginVolume.Sub(o.Volume),
		Cancelled:      o.State == StateCancel || o.State == StateDone,
		Fake:           false,
		CreatedAt:      o.CreatedAt,
	}
}
