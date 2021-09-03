package models

import (
	"encoding/json"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	api_admin_entities "github.com/zsmartex/finex/controllers/admin_controllers/entities"
	api_entities "github.com/zsmartex/finex/controllers/entities"
	"github.com/zsmartex/finex/types"
	"github.com/zsmartex/pkg/order"
	"gorm.io/gorm"
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
		"created_at": t.CreatedAt.Unix(),
	}

	config.InfluxDB.NewPoint("trades", tags, fields)
}

func (t *Trade) RecordCompleteOperations(seller_matching_order, buyer_matching_order order.Order, tx *gorm.DB) {
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
	t.RecordRevenues(seller_order, buyer_order, seller_matching_order, buyer_matching_order, reference, tx)
}

func (t *Trade) RecordLiabilityDebit(seller_order, buyer_order *Order, seller_matching_order, buyer_matching_order order.Order, reference Reference) {
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

func (t *Trade) RecordLiabilityCredit(seller_order, buyer_order *Order, seller_matching_order, buyer_matching_order order.Order, reference Reference) {
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
func (t *Trade) RecordLiabilityTransfer(seller_order, buyer_order *Order, seller_matching_order, buyer_matching_order order.Order, reference Reference) {
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

func (t *Trade) RecordRevenues(seller_order, buyer_order *Order, seller_matching_order, buyer_matching_order order.Order, reference Reference, tx *gorm.DB) {
	seller_fee := t.Total.Mul(t.OrderFee(seller_order))
	buyer_fee := t.Amount.Mul(t.OrderFee(buyer_order))

	if config.Referral.Enabled {
		sort.Slice(config.Referral.Rewards, func(i, j int) bool {
			return config.Referral.Rewards[i].HoldAmount.GreaterThan(config.Referral.Rewards[j].HoldAmount)
		})
		var refCurrency *Currency
		config.DataBase.First(&refCurrency, "id = ?", strings.ToLower(config.Referral.CurrencyID))

		for _, reward := range config.Referral.Rewards {
			if !seller_matching_order.IsFake() && seller_fee.IsPositive() {
				member := seller_order.Member()
				if !member.HavingReferraller() {
					break
				}

				refMember := member.GetRefMember()
				refHoldAccount := refMember.GetAccount(refCurrency)

				if refHoldAccount.Balance.GreaterThanOrEqual(reward.HoldAmount) {
					reward_amount := seller_fee.Mul(reward.Reward)
					refMember.GetAccount(seller_order.IncomeCurrency()).PlusFunds(tx, reward_amount)
					seller_fee = seller_fee.Sub(reward_amount)
					break
				}
			}
		}

		for _, reward := range config.Referral.Rewards {
			if !buyer_matching_order.IsFake() && buyer_fee.IsPositive() {
				member := buyer_order.Member()
				if member.HavingReferraller() {
					break
				}

				refMember := member.GetRefMember()
				refHoldAccount := refMember.GetAccount(refCurrency)

				if refHoldAccount.Balance.GreaterThanOrEqual(reward.HoldAmount) {
					reward_amount := buyer_fee.Mul(reward.Reward)
					refMember.GetAccount(seller_order.IncomeCurrency()).PlusFunds(tx, reward_amount)
					buyer_fee = buyer_fee.Sub(reward_amount)
					break
				}
			}
		}
	}

	if !seller_matching_order.IsFake() && seller_fee.IsPositive() {
		RevenueCredit(
			seller_fee,
			seller_order.IncomeCurrency(),
			reference,
			seller_order.MemberID,
		)
	}

	if !buyer_matching_order.IsFake() && buyer_fee.IsPositive() {
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

func GetLastTradeFromInflux(market string) *Trade {
	var trades []map[string]interface{}
	config.InfluxDB.Query("SELECT LAST(*) FROM \"trades\" WHERE \"market\"='"+market+"'", &trades)
	if len(trades) > 0 {
		log.Println(trades[0]["last_price"])

		id, _ := trades[0]["last_id"].(json.Number).Int64()
		last_price, _ := trades[0]["last_price"].(json.Number).Float64()
		last_amount, _ := trades[0]["last_amount"].(json.Number).Float64()
		last_total, _ := trades[0]["last_total"].(json.Number).Float64()
		last_created_at, _ := trades[0]["last_created_at"].(json.Number).Int64()

		return &Trade{
			ID:        uint64(id),
			Price:     decimal.NewFromFloat(last_price),
			Amount:    decimal.NewFromFloat(last_amount),
			Total:     decimal.NewFromFloat(last_total),
			CreatedAt: time.Unix(last_created_at, 0),
		}
	}

	return nil
}

func (t *Trade) ForUser(member *Member) api_entities.TradeEntity {
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

	return api_entities.TradeEntity{
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
	CreatedAt int64           `json:"created_at"`
}

func (t *Trade) TradeGlobalJSON() TradeGlobalJSON {
	return TradeGlobalJSON{
		ID:        t.ID,
		Market:    t.MarketID,
		Price:     t.Price,
		Amount:    t.Amount,
		Total:     t.Total,
		TakerType: t.TakerType,
		CreatedAt: t.CreatedAt.Unix(),
	}
}

func (t *Trade) ToJSON() api_admin_entities.TradeEntity {
	var maker_email string
	var maker_uid string
	var maker_fee = decimal.Zero
	var maker_fee_amount = decimal.Zero
	var maker_fee_currency string

	if t.MakerOrderID > 0 {
		maker := t.Maker()
		maker_order := t.MakerOrder()

		maker_email = maker.Email
		maker_uid = maker.UID
		maker_fee = maker_order.MakerFee
		maker_fee_amount = t.OrderFee(maker_order)

		if maker_order.Side() == types.TypeBuy {
			maker_fee_currency = maker_order.Ask
		} else {
			maker_fee_currency = maker_order.Bid
		}
	}

	var taker_email string
	var taker_uid string
	var taker_fee = decimal.Zero
	var taker_fee_amount = decimal.Zero
	var taker_fee_currency string

	if t.MakerOrderID > 0 {
		taker := t.Taker()
		taker_order := t.TakerOrder()

		taker_email = taker.Email
		taker_uid = taker.UID
		taker_fee = taker_order.MakerFee
		taker_fee_amount = t.OrderFee(taker_order)

		if taker_order.Side() == types.TypeBuy {
			taker_fee_currency = taker_order.Ask
		} else {
			taker_fee_currency = taker_order.Bid
		}
	}

	return api_admin_entities.TradeEntity{
		ID:               t.ID,
		Price:            t.Price,
		Amount:           t.Amount,
		Total:            t.Total,
		Market:           t.MarketID,
		TakerType:        t.TakerType,
		MakerOrderEmail:  maker_email,
		TakerOrderEmail:  taker_email,
		MakerUID:         maker_uid,
		TakerUID:         taker_uid,
		MakerFee:         maker_fee,
		MakerFeeAmount:   maker_fee_amount,
		MakerFeeCurrency: maker_fee_currency,
		TakerFee:         taker_fee,
		TakerFeeAmount:   taker_fee_amount,
		TakerFeeCurrency: taker_fee_currency,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}
