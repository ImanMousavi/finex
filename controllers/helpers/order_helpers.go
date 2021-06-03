package helpers

import (
	"github.com/gookit/validate"

	"gitlab.com/zsmartex/finex/config"
	"gitlab.com/zsmartex/finex/matching"
	"gitlab.com/zsmartex/finex/models"
	"gitlab.com/zsmartex/finex/types"
)

type CreateOrderPayload struct {
	Market    string             `json:"market" form:"market" validate:"required"`
	Side      types.OrderSide    `json:"side" form:"side" validate:"required|VaildateSide"`
	OrdType   matching.OrderType `json:"ord_type" form:"ord_type" default:"limit"`
	Price     float64            `json:"price" form:"price" validate:"required_if:OrdType,market|gt:0"`
	StopPrice float64            `json:"stop_price" form:"stop_price" validate:"gt:0"`
	Volume    float64            `json:"volume" form:"volume" validate:"required|gt:0"`
}

func (p CreateOrderPayload) Messages() map[string]string {
	invalid_message := "market.order.invalid_{field}"

	return validate.MS{
		"required":     invalid_message,
		"VaildateSide": invalid_message,
	}
}

func (p CreateOrderPayload) GetMarket() models.Market {
	var market models.Market

	config.DataBase.Find(&market, p.Market)

	return market
}

func (p CreateOrderPayload) VaildateSide(val string) bool {
	return p.Side == types.SideBuy || p.Side == types.SideSell
}

func (p CreateOrderPayload) BuildOrder(member *models.Member) *models.Order {
	market := p.GetMarket()

	order := &models.Order{
		MemberID:     member.ID,
		Ask:          market.BaseUnit,
		Bid:          market.QuoteUnit,
		MarketID:     market.ID,
		OrdType:      matching.TypeLimit,
		Volume:       p.Volume,
		OriginVolume: p.Volume,
	}

	return order
}

func (p CreateOrderPayload) CreateOrder(member *models.Member, err_src *Errors) (order *models.Order) {
	order = p.BuildOrder(member)

	Vaildate(order, err_src)
	order.Submit(&err_src.Errors)

	return order
}
