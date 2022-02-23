package models

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	api_admin_entities "github.com/zsmartex/finex/controllers/admin_controllers/entities"
	api_entities "github.com/zsmartex/finex/controllers/entities"
	"github.com/zsmartex/finex/types"
	"github.com/zsmartex/pkg"
	"gorm.io/gorm"
)

type Trade struct {
	ID           int64           `json:"id" gorm:"primaryKey"`
	Price        decimal.Decimal `json:"price" validate:"ValidatePrice"`
	Amount       decimal.Decimal `json:"amount" validate:"ValidateAmount"`
	Total        decimal.Decimal `json:"total" validate:"ValidateTotal"`
	MakerOrderID int64           `json:"maker_order_id"`
	TakerOrderID int64           `json:"taker_order_id"`
	MarketID     string          `json:"market_id"`
	MakerID      int64           `json:"maker_id"`
	TakerID      int64           `json:"taker_id"`
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
	var member *Member

	config.DataBase.First(&member, "id = ?", t.MakerID)

	return member
}

func (t *Trade) Taker() *Member {
	var member *Member

	config.DataBase.First(&member, "id = ?", t.TakerID)

	return member
}

func (t *Trade) MakerOrder() *Order {
	var order *Order
	if t.MakerOrderID == 0 {
		panic("lmao")
	}
	config.DataBase.First(&order, "id = ?", t.MakerOrderID)
	return order
}

func (t *Trade) TakerOrder() *Order {
	var order *Order
	config.DataBase.First(&order, "id = ?", t.TakerOrderID)
	return order
}

func (t *Trade) SellerOrder() *Order {
	maker_order := &Order{}
	taker_order := &Order{}

	if t.MakerOrderID > 0 {
		maker_order = t.MakerOrder()
	}
	if t.TakerOrderID > 0 {
		taker_order = t.TakerOrder()
	}

	if maker_order.Side() == types.TypeSell && t.MakerOrderID > 0 {
		return maker_order
	} else {
		return taker_order
	}
}

func (t *Trade) BuyerOrder() *Order {
	maker_order := &Order{}
	taker_order := &Order{}

	if t.MakerOrderID > 0 {
		maker_order = t.MakerOrder()
	}
	if t.TakerOrderID > 0 {
		taker_order = t.TakerOrder()
	}

	if maker_order.Side() == types.TypeBuy && t.MakerOrderID > 0 {
		return maker_order
	} else {
		return taker_order
	}
}

func (t *Trade) Side(member *Member) types.TakerType {
	return t.OrderForMember(member).Side()
}

func (t *Trade) OrderForMember(member *Member) *Order {
	if member.ID == int64(t.MakerID) {
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

func (t *Trade) RecordCompleteOperations(seller_matching_order, buyer_matching_order pkg.Order, tx *gorm.DB) error {
	var seller_order *Order
	var buyer_order *Order

	is_seller_fake := seller_matching_order.IsFake()
	is_buyer_fake := buyer_matching_order.IsFake()

	if !is_seller_fake {
		seller_order = t.SellerOrder()
	}
	if !is_buyer_fake {
		buyer_order = t.BuyerOrder()
	}
	reference := Reference{
		ID:   t.ID,
		Type: "Trade",
	}

	t.RecordLiabilityDebit(seller_order, buyer_order, is_seller_fake, is_buyer_fake, reference)
	t.RecordLiabilityCredit(seller_order, buyer_order, is_seller_fake, is_buyer_fake, reference)
	t.RecordLiabilityTransfer(seller_order, buyer_order, is_seller_fake, is_buyer_fake, reference)

	seller_fee := decimal.Zero
	buyer_fee := decimal.Zero

	if !is_seller_fake {
		seller_fee = t.Total.Mul(t.OrderFee(seller_order))
	}

	if !is_buyer_fake {
		buyer_fee = t.Amount.Mul(t.OrderFee(buyer_order))
	}

	s_fee, b_fee, err := t.RecordReferrals(
		seller_fee,
		buyer_fee,
		seller_order,
		buyer_order,
		is_seller_fake,
		is_buyer_fake,
		reference,
		tx,
	)

	if err != nil {
		return err
	}

	seller_fee = s_fee
	buyer_fee = b_fee

	t.RecordRevenues(seller_fee, buyer_fee, seller_order, buyer_order, is_seller_fake, is_buyer_fake, reference, tx)

	return nil
}

func (t *Trade) RecordLiabilityDebit(seller_order, buyer_order *Order, is_seller_fake, is_buyer_fake bool, reference Reference) {
	seller_outcome := t.Amount
	buyer_outcome := t.Total

	if !is_seller_fake {
		LiabilityDebit(
			seller_outcome,
			seller_order.OutcomeCurrency(),
			reference,
			"locked",
			seller_order.MemberID,
		)
	}

	if !is_buyer_fake {
		LiabilityDebit(
			buyer_outcome,
			buyer_order.OutcomeCurrency(),
			reference,
			"locked",
			buyer_order.MemberID,
		)
	}
}

func (t *Trade) RecordLiabilityCredit(seller_order, buyer_order *Order, is_seller_fake, is_buyer_fake bool, reference Reference) {
	if !is_seller_fake {
		seller_income := t.Total.Sub(t.Total.Mul(t.OrderFee(t.SellerOrder())))
		LiabilityDebit(
			seller_income,
			seller_order.IncomeCurrency(),
			reference,
			"main",
			seller_order.MemberID,
		)
	}

	if !is_buyer_fake {
		buyer_income := t.Amount.Sub(t.Amount.Mul(t.OrderFee(t.BuyerOrder())))
		LiabilityDebit(
			buyer_income,
			buyer_order.IncomeCurrency(),
			reference,
			"main",
			buyer_order.MemberID,
		)
	}
}

// TODO: Fix it
func (t *Trade) RecordLiabilityTransfer(seller_order, buyer_order *Order, is_seller_fake, is_buyer_fake bool, reference Reference) {
	if !is_seller_fake {
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

	if !is_buyer_fake {
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

func (t *Trade) RecordReferrals(seller_fee, buyer_fee decimal.Decimal, seller_order, buyer_order *Order, is_seller_fake, is_buyer_fake bool, reference Reference, tx *gorm.DB) (decimal.Decimal, decimal.Decimal, error) {
	if !config.Referral.Enabled {
		return seller_fee, buyer_fee, nil
	}

	sort.Slice(config.Referral.Rewards, func(i, j int) bool {
		return config.Referral.Rewards[i].HoldAmount.GreaterThan(config.Referral.Rewards[j].HoldAmount)
	})
	var refCurrency *Currency
	config.DataBase.First(&refCurrency, "id = ?", strings.ToLower(config.Referral.Currency))

	for _, reward := range config.Referral.Rewards {
		if !is_seller_fake && seller_fee.IsPositive() {
			member := seller_order.Member()
			if !member.HavingReferraller() {
				break
			}

			refMember := member.GetRefMember()
			refHoldAccount := refMember.GetAccount(refCurrency)

			if refHoldAccount.Balance.GreaterThanOrEqual(reward.HoldAmount) {
				reward_amount := seller_fee.Mul(reward.Reward).Round(8)
				if err := refMember.GetAccount(seller_order.IncomeCurrency()).PlusFunds(tx, reward_amount); err != nil {
					return seller_fee, buyer_fee, err
				}
				seller_fee = seller_fee.Sub(reward_amount)

				if result := tx.Create(
					&Commission{
						AccountType:     "spot",
						MemberID:        refMember.ID,
						FriendUID:       member.UID,
						EarnAmount:      reward_amount,
						CurrencyID:      seller_order.IncomeCurrency().ID,
						ParentID:        t.ID,
						ParentCreatedAt: t.CreatedAt,
					},
				); result.Error != nil {
					return seller_fee, buyer_fee, result.Error
				}

				break
			}
		}
	}

	for _, reward := range config.Referral.Rewards {
		if !is_buyer_fake && buyer_fee.IsPositive() {
			member := buyer_order.Member()
			if !member.HavingReferraller() {
				break
			}

			refMember := member.GetRefMember()
			refHoldAccount := refMember.GetAccount(refCurrency)

			if refHoldAccount.Balance.GreaterThanOrEqual(reward.HoldAmount) {
				reward_amount := buyer_fee.Mul(reward.Reward).Round(8)
				if err := refMember.GetAccount(buyer_order.IncomeCurrency()).PlusFunds(tx, reward_amount); err != nil {
					return seller_fee, buyer_fee, err
				}
				buyer_fee = buyer_fee.Sub(reward_amount)

				if result := tx.Create(
					&Commission{
						AccountType:     "spot",
						MemberID:        refMember.ID,
						FriendUID:       member.UID,
						EarnAmount:      reward_amount,
						CurrencyID:      buyer_order.IncomeCurrency().ID,
						ParentID:        t.ID,
						ParentCreatedAt: t.CreatedAt,
					},
				); result.Error != nil {
					return seller_fee, buyer_fee, result.Error
				}

				break
			}
		}
	}

	return seller_fee, buyer_fee, nil
}

func (t *Trade) RecordRevenues(seller_fee, buyer_fee decimal.Decimal, seller_order, buyer_order *Order, is_seller_fake, is_buyer_fake bool, reference Reference, tx *gorm.DB) {
	if !is_seller_fake && seller_fee.IsPositive() {
		RevenueCredit(
			seller_fee,
			seller_order.IncomeCurrency(),
			reference,
			seller_order.MemberID,
		)
	}

	if !is_buyer_fake && buyer_fee.IsPositive() {
		RevenueCredit(
			buyer_fee,
			buyer_order.IncomeCurrency(),
			reference,
			buyer_order.MemberID,
		)
	}
}

func (t *Trade) OrderFee(order *Order) decimal.Decimal {
	if int64(t.MakerOrderID) == order.ID {
		return order.MakerFee
	} else {
		return order.TakerFee
	}
}

func GetLastTradeFromInflux(market string) *Trade {
	var trades []map[string]interface{}
	config.InfluxDB.Query("SELECT LAST(*) FROM \"trades\" WHERE \"market\"='"+market+"'", &trades)
	if len(trades) > 0 {
		id, _ := trades[0]["last_id"].(json.Number).Int64()
		last_price, _ := trades[0]["last_price"].(json.Number).Float64()
		last_amount, _ := trades[0]["last_amount"].(json.Number).Float64()
		last_total, _ := trades[0]["last_total"].(json.Number).Float64()
		last_created_at, _ := trades[0]["last_created_at"].(json.Number).Int64()

		return &Trade{
			ID:        int64(id),
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
	ID        int64           `json:"id"`
	Market    string          `json:"market"`
	Price     decimal.Decimal `json:"price"`
	Amount    decimal.Decimal `json:"amount"`
	Total     decimal.Decimal `json:"total"`
	TakerType types.TakerType `json:"taker_type"`
	CreatedAt int64           `json:"created_at"`
}

func (t *Trade) TradeGlobalJSON() interface{} {
	if os.Getenv("UI") == "BASEAPP" {
		return map[string]interface{}{
			"id":         t.ID,
			"market":     t.MarketID,
			"price":      t.Price,
			"amount":     t.Amount,
			"total":      t.Total,
			"taker_type": t.TakerType,
			"date":       t.CreatedAt.Unix(),
		}
	} else {
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
