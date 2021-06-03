package matching

import (
	"encoding/json"

	"github.com/ericlagergren/decimal"
	"github.com/zsmartex/go-finex/mq_client"
)

type PayloadAction = string

var (
	ActionSubmit PayloadAction = "submit"
	ActionCancel PayloadAction = "cancel"
)

type MatchingPayloadMessage struct {
	Action PayloadAction `json:"action"`
	Order  Order         `json:"order"`
}

var ORDER_SUBMIT_MAX_ATTEMPTS = 3

type Engine struct {
	Market    string
	OrderBook *OrderBook
}

func NewEngine(market string) *Engine {
	return &Engine{
		Market: market,
		OrderBook: NewOrderBook(
			market,
			decimal.New(0, 0),
			NewTradeBook(market),
			NOPOrderRepository,
		),
	}
}

func (e Engine) Submit(order Order, attempt int) {
	_, err := e.OrderBook.Add(order)

	if err != nil {
		if attempt > ORDER_SUBMIT_MAX_ATTEMPTS {
			PublishCancel(order)
		} else {
			e.Submit(order, attempt+1)
		}
	}
}

func (e Engine) Cancel(order Order) {
	e.OrderBook.Cancel(order.ID)

	PublishCancel(order)
}

func PublishCancel(order Order) {
	order_processor_message, err := json.Marshal(MatchingPayloadMessage{
		Action: "cancel",
		Order:  order,
	})

	if err != nil {
		return
	}

	mq_client.Enqueue("order_processor", order_processor_message)
}
