package matching

import (
	"encoding/json"

	"github.com/zsmartex/go-finex/config"
)

type PayloadAction = string

var (
	ActionSubmit PayloadAction = "submit"
	ActionCancel PayloadAction = "cancel"
	ActionReload PayloadAction = "reload"
)

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
		),
	}
}

func (e Engine) Submit(order *Order) error {
	trades := e.OrderBook.InsertOrder(order)

	for _, trade := range trades {
		trade_message, _ := json.Marshal(trade)
		if err := config.Nats.Publish("trade_executor", trade_message); err != nil {
			return err
		}
	}

	return nil
}

func (e Engine) Cancel(order *Order) error {
	e.OrderBook.CancelOrder(order)

	return PublishCancel(order)
}

func PublishCancel(order *Order) error {
	order_processor_message, err := json.Marshal(map[string]interface{}{
		"action": ActionCancel,
		"order":  order,
	})

	if err != nil {
		return err
	}

	return config.Nats.Publish("order_processor", order_processor_message)
}
