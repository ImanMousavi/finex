package matching

import (
	"encoding/json"
	"log"

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

func (e Engine) Submit(order *Order) {
	log.Printf("Submiting order_id: %v\n", order.ID)
	trades := e.OrderBook.InsertOrder(order)

	for _, trade := range trades {
		trade_message, _ := json.Marshal(trade)
		config.Nats.Publish("trade_executor", trade_message)
	}
}

func (e Engine) Cancel(order *Order) {
	e.OrderBook.CancelOrder(order)

	PublishCancel(order)
}

func PublishCancel(order *Order) {
	order_processor_message, err := json.Marshal(map[string]interface{}{
		"action": ActionCancel,
		"order":  order,
	})

	if err != nil {
		return
	}

	config.Nats.Publish("order_processor", order_processor_message)
}
