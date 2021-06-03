package types

import "github.com/zsmartex/go-finex/matching"

type Depth struct {
	Asks     [][]float64 `json:"asks"`
	Bids     [][]float64 `json:"bids"`
	Sequence uint64      `json:"sequence"`
}

type PayloadAction = string

var (
	ActionSubmit PayloadAction = "submit"
	ActionCancel PayloadAction = "cancel"
)

type MatchingPayloadMessage struct {
	Action PayloadAction  `json:"action"`
	Order  matching.Order `json:"order"`
}

type OrderProcessorPayloadMessage = MatchingPayloadMessage

type OrderSide = string

var (
	SideBuy  OrderSide = "buy"
	SideSell OrderSide = "sell"
)
