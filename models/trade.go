package models

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/entities"
	"github.com/zsmartex/finex/types"
	"github.com/zsmartex/pkg/order"
)

type Trade struct {
	ID           uint64          `json:"id" gorm:"primaryKey"`
	Price        decimal.Decimal `json:"price" validate:"ValidatePrice"`
	Amount       decimal.Decimal `json:"amount" validate:"ValidateAmount"`
	Total        decimal.Decimal `json:"total" validate:"ValidateTotal"`
	MakerOrderID uint64          `json:"maker_order_id"`
	TakerOrderID uint64          `json:"taker_order_id"`
	MarketID     string          `json:"market_id"`
	MakerID      uint64          `json:"maker_id"`
	TakerID      uint64          `json:"taker_id"`
	TakerType    types.TakerType `json:"taker_type"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (t Trade) ValidatePrice(Price decimal.Decimal) bool {
	return Price.IsPositive()
}

func (t Trade) ValidateAmount(Amount decimal.Decimal) bool {
	return Amount.IsPositive()
}

func (t Trade) ValidateTotal(Total decimal.Decimal) bool {
	return Total.IsPositive()
}

func (t *Trade) Market() *Market {
	market := &Market{}

	config.DataBase.First(&market, "id = ?", t.MarketID)

	return market
}

func (t *Trade) Maker() *Member {
	member := &Member{}

	config.DataBase.First(&member, "id = ?", t.MakerID)

	return member
}

func (t *Trade) Taker() *Member {
	member := &Member{}

	config.DataBase.First(&member, "id = ?", t.TakerID)

	return member
}

func (t *Trade) MakerOrder() *Order {
	order := &Order{}
	config.DataBase.First(&order, "id = ?", t.MakerOrderID)
	return order
}

func (t *Trade) TakerOrder() *Order {
	order := &Order{}
	config.DataBase.First(&order, "id = ?", t.TakerOrderID)
	return order
}

func (t *Trade) SellerOrder() *Order {
	maker_order := t.MakerOrder()
	taker_order := t.TakerOrder()

	if maker_order.Side() == types.TypeSell {
		return maker_order
	} else {
		return taker_order
	}
}

func (t *Trade) BuyerOrder() *Order {
	maker_order := t.MakerOrder()
	taker_order := t.TakerOrder()

	if maker_order.Side() == types.TypeBuy {
		return maker_order
	} else {
		return taker_order
	}
}

func (t *Trade) Side(member *Member) types.TakerType {
	return t.OrderForMember(member).Side()
}

func (t *Trade) OrderForMember(member *Member) *Order {
	if member.ID == uint64(t.MakerID) {
		return t.MakerOrder()
	} else {
		return t.TakerOrder()
	}
}

func (t *Trade) WriteToInflux() {
	price, _ := t.Price.Float64()
	amount, _ := t.Amount.Float64()
	total, _ := t.Total.Float64()

	tags := map[string]string{"market": t.MarketID}
	fields := map[string]interface{}{
		"id":         int32(t.ID),
		"price":      price,
		"amount":     amount,
		"total":      total,
		"taker_type": t.TakerType,
		"created_at": t.CreatedAt,
	}

	config.InfluxDB.NewPoint("trades", tags, fields)
}

func (t *Trade) RecordCompleteOperations(seller_matching_order, buyer_matching_order *order.Order) {
	var seller_order *Order
	var buyer_order *Order
	if !seller_matching_order.IsFake() {
		seller_order = t.SellerOrder()
	}
	if !buyer_matching_order.IsFake() {
		buyer_order = t.BuyerOrder()
	}
	reference := Reference{
		ID:   t.ID,
		Type: "Trade",
	}

	t.RecordLiabilityDebit(seller_order, buyer_order, seller_matching_order, buyer_matching_order, reference)
	t.RecordLiabilityCredit(seller_order, buyer_order, seller_matching_order, buyer_matching_order, reference)
	t.RecordLiabilityTransfer(seller_order, buyer_order, seller_matching_order, buyer_matching_order, reference)
	t.RecordRevenues(seller_order, buyer_order, seller_matching_order, buyer_matching_order, reference)
}

func (t *Trade) RecordLiabilityDebit(seller_order, buyer_order *Order, seller_matching_order, buyer_matching_order *order.Order, reference Reference) {
	seller_outcome := t.Amount
	buyer_outcome := t.Total

	if !seller_matching_order.IsFake() {
		LiabilityDebit(
			seller_outcome,
			seller_order.OutcomeCurrency(),
			reference,
			"locked",
			seller_order.MemberID,
		)
	}

	if !buyer_matching_order.IsFake() {
		LiabilityDebit(
			buyer_outcome,
			buyer_order.OutcomeCurrency(),
			reference,
			"locked",
			buyer_order.MemberID,
		)
	}
}

func (t *Trade) RecordLiabilityCredit(seller_order, buyer_order *Order, seller_matching_order, buyer_matching_order *order.Order, reference Reference) {
	if !buyer_matching_order.IsFake() {
		buyer_income := t.Amount.Sub(t.Amount.Mul(t.OrderFee(t.BuyerOrder())))
		LiabilityDebit(
			buyer_income,
			buyer_order.IncomeCurrency(),
			reference,
			"main",
			buyer_order.MemberID,
		)
	}

	if !seller_matching_order.IsFake() {
		seller_income := t.Total.Sub(t.Total.Mul(t.OrderFee(t.SellerOrder())))
		LiabilityDebit(
			seller_income,
			seller_order.IncomeCurrency(),
			reference,
			"main",
			seller_order.MemberID,
		)
	}
}

// TODO: Fix it
func (t *Trade) RecordLiabilityTransfer(seller_order, buyer_order *Order, seller_matching_order, buyer_matching_order *order.Order, reference Reference) {
	if !seller_matching_order.IsFake() {
		if seller_order.Volume.IsZero() || !seller_order.Locked.IsZero() {
			LiabilityTranfer(
				seller_order.Locked,
				seller_order.OutcomeCurrency(),
				reference,
				"locked",
				"main",
				seller_order.MemberID,
			)
		}
	}

	if !buyer_matching_order.IsFake() {
		if buyer_order.Volume.IsZero() || !buyer_order.Locked.IsZero() {
			LiabilityTranfer(
				buyer_order.Locked,
				buyer_order.OutcomeCurrency(),
				reference,
				"locked",
				"main",
				buyer_order.MemberID,
			)
		}
	}
}

func (t *Trade) RecordRevenues(seller_order, buyer_order *Order, seller_matching_order, buyer_matching_order *order.Order, reference Reference) {
	seller_fee := t.Total.Mul(t.OrderFee(seller_order))
	buyer_fee := t.Amount.Mul(t.OrderFee(buyer_order))

	if !seller_matching_order.IsFake() {
		RevenueCredit(
			seller_fee,
			seller_order.IncomeCurrency(),
			reference,
			seller_order.MemberID,
		)
	}

	if !buyer_matching_order.IsFake() {
		RevenueCredit(
			buyer_fee,
			buyer_order.IncomeCurrency(),
			reference,
			buyer_order.MemberID,
		)
	}
}

func (t *Trade) OrderFee(order *Order) decimal.Decimal {
	if uint64(t.MakerOrderID) == order.ID {
		return order.MakerFee
	} else {
		return order.TakerFee
	}
}

func (t *Trade) ToJSON(member *Member) entities.TradeEntities {
	var fee_currency string
	var fee_amount decimal.Decimal
	order := t.OrderForMember(member)
	side := order.Side()

	if side == types.TypeBuy {
		fee_currency = order.Ask
		fee_amount = t.OrderFee(order).Mul(t.Amount)
	} else {
		fee_currency = order.Bid
		fee_amount = t.OrderFee(order).Mul(t.Total)
	}

	return entities.TradeEntities{
		ID:          t.ID,
		Market:      t.MarketID,
		Price:       t.Price,
		Amount:      t.Amount,
		Total:       t.Total,
		FeeCurrency: fee_currency,
		Fee:         t.OrderFee(order),
		FeeAmount:   fee_amount,
		TakerType:   t.TakerType,
		Side:        side,
		OrderID:     t.ID,
		CreatedAt:   t.CreatedAt,
	}
}

type TradeGlobalJSON struct {
	ID        uint64          `json:"id"`
	Market    string          `json:"market"`
	Price     decimal.Decimal `json:"price"`
	Amount    decimal.Decimal `json:"amount"`
	Total     decimal.Decimal `json:"total"`
	TakerType types.TakerType `json:"taker_type"`
	CreatedAt time.Time       `json:"created_at"`
}

func (t *Trade) TradeGlobalJSON() TradeGlobalJSON {
	return TradeGlobalJSON{
		ID:        t.ID,
		Market:    t.MarketID,
		Price:     t.Price,
		Amount:    t.Amount,
		Total:     t.Total,
		TakerType: t.TakerType,
		CreatedAt: t.CreatedAt,
	}
}
