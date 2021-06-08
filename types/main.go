package types

import "github.com/shopspring/decimal"

type Depth struct {
	Asks     [][]decimal.Decimal `json:"asks"`
	Bids     [][]decimal.Decimal `json:"bids"`
	Sequence uint64              `json:"sequence"`
}

type OrderSide string

var (
	SideBuy  OrderSide = "buy"
	SideSell OrderSide = "sell"
)

type GlobalPrice map[string]map[string]float64

type OrderBy string

var (
	OrderByAsc  OrderBy = "asc"
	OrderByDesc OrderBy = "desc"
)

type TakerType string

var (
	TypeBuy  TakerType = "buy"
	TypeSell TakerType = "sell"
)

type OrderType string

const (
	TypeLimit  OrderType = "limit"
	TypeMarket OrderType = "market"
)
