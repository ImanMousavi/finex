package models

import (
	"time"

	"gitlab.com/zsmartex/finex/config"
)

type TakerType = string

var (
	TypeBuy  TakerType = "buy"
	TypeSell TakerType = "sell"
)

type Trade struct {
	ID           uint64    `json:"id" gorm:"primaryKey"`
	Price        float64   `json:"price" validate:"min:0"`
	Amount       float64   `json:"amount" validate:"min:0"`
	Total        float64   `json:"total" gorm:"default:0.0" validate:"min:0"`
	MakerOrderID uint64    `json:"maker_order_id"`
	TakerOrderID uint64    `json:"taker_order_id"`
	MarketID     string    `json:"market_id"`
	MakerID      uint64    `json:"maker_id"`
	TakerID      uint64    `json:"taker_id"`
	TakerType    TakerType `json:"taker_type"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (t Trade) Market() Market {
	var market Market

	config.DataBase.Find(&market, 1)

	return market
}

func (t Trade) MakerOrder() Order {
	var order Order
	config.DataBase.Find(&order, t.MakerOrderID)
	return order
}

func (t Trade) TakerOrder() Order {
	var order Order
	config.DataBase.Find(&order, t.TakerOrderID)
	return order
}

func (t Trade) SellerOrder() Order {
	maker_order := t.MakerOrder()
	taker_order := t.TakerOrder()

	if maker_order.Side() == TypeSell {
		return maker_order
	} else {
		return taker_order
	}
}

func (t Trade) BuyerOrder() Order {
	maker_order := t.MakerOrder()
	taker_order := t.TakerOrder()

	if maker_order.Side() == TypeBuy {
		return maker_order
	} else {
		return taker_order
	}
}

func (t Trade) TriggerEvent() {

}

func (t Trade) WriteToInflux() {

}

func (t Trade) RecordCompleteOperations() {
	seller_order := t.SellerOrder()
	buyer_order := t.BuyerOrder()
	reference := Reference{
		ID:   t.ID,
		Type: "Trade",
	}

	t.RecordLiabilityDebit(seller_order, buyer_order, reference)
	t.RecordLiabilityCredit(seller_order, buyer_order, reference)
	t.RecordLiabilityTransfer(seller_order, buyer_order, reference)
	t.RecordRevenues(seller_order, buyer_order, reference)
}

func (t Trade) RecordLiabilityDebit(seller_order, buyer_order Order, reference Reference) {
	seller_outcome := t.Amount
	buyer_outcome := t.Total

	LiabilityDebit(
		seller_outcome,
		seller_order.Currency(),
		reference,
		"locked",
		seller_order.MemberID,
	)
	LiabilityDebit(
		buyer_outcome,
		buyer_order.Currency(),
		reference,
		"locked",
		buyer_order.MemberID,
	)
}

func (t Trade) RecordLiabilityCredit(seller_order, buyer_order Order, reference Reference) {
	seller_income := t.Total - t.Total*t.OrderFee(seller_order)
	buyer_income := t.Amount - t.Amount*t.OrderFee(buyer_order)

	LiabilityDebit(
		buyer_income,
		buyer_order.IncomeCurrency(),
		reference,
		"main",
		buyer_order.MemberID,
	)
	LiabilityDebit(
		seller_income,
		seller_order.IncomeCurrency(),
		reference,
		"main",
		seller_order.MemberID,
	)
}

func (t Trade) RecordLiabilityTransfer(seller_order, buyer_order Order, reference Reference) {
	orders := []Order{seller_order, buyer_order}

	for _, order := range orders {
		if order.Volume == 0 || order.Locked != 0 {
			LiabilityTranfer(
				order.Locked,
				order.OutcomeCurrency(),
				reference,
				"locked",
				"main",
				order.MemberID,
			)
		}
	}
}

func (t Trade) RecordRevenues(seller_order, buyer_order Order, reference Reference) {
	seller_fee := t.Total * t.OrderFee(seller_order)
	buyer_fee := t.Amount * t.OrderFee(buyer_order)

	RevenueCredit(
		seller_fee,
		seller_order.IncomeCurrency(),
		reference,
		seller_order.MemberID,
	)
	RevenueCredit(
		buyer_fee,
		buyer_order.IncomeCurrency(),
		reference,
		buyer_order.MemberID,
	)
}

func (t Trade) OrderFee(order Order) float64 {
	if t.MakerOrderID == order.ID {
		return order.MakerFee
	} else {
		return order.TakerFee
	}
}
