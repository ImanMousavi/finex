package types

import "github.com/shopspring/decimal"

type Depth struct {
	Asks     [][]decimal.Decimal `json:"asks"`
	Bids     [][]decimal.Decimal `json:"bids"`
	Sequence int64               `json:"sequence"`
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

type Config struct {
	Referral *Referral `yaml:"referral"`
}

type Referral struct {
	Enabled  bool                   `yaml:"enabled"`
	Currency string                 `yaml:"currency"`
	Rewards  []ConfigReferralReward `yaml:"rewards"`
}

type ConfigReferralReward struct {
	HoldAmount decimal.Decimal `yaml:"hold_amount"`
	Reward     decimal.Decimal `yaml:"reward"`
}

type MarketState string

var (
	MarketStateEndabled MarketState = "enabled"
	MarketStateDisabled MarketState = "disabled"
)

type AccountType string

var (
	AccountTypeSpot    AccountType = "spot"
	AccountTypeP2P     AccountType = "p2p"
	AccountTypeMargin  AccountType = "margin"
	AccountTypeFutures AccountType = "futures"
)
