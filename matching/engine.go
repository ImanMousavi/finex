package matching

import (
	"encoding/json"

	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/mq_client"
)

type PayloadAction = string

var (
	ActionSubmit PayloadAction = "submit"
	ActionCancel PayloadAction = "cancel"
	ActionReload PayloadAction = "reload"
)

var ORDER_SUBMIT_MAX_ATTEMPTS = 3

type Engine struct {
	Market      string
	OrderBook   *OrderBook
	Initialized bool
}

func NewEngine(market string) *Engine {
	return &Engine{
		Market: market,
		OrderBook: NewOrderBook(
			market,
		),
		Initialized: false,
	}
}

func (e Engine) Submit(order *Order) error {
	trades := e.OrderBook.InsertOrder(order)

	for _, trade := range trades {
		trade_message, _ := json.Marshal(map[string]interface{}{
			"action": "execute",
			"trade": map[string]interface{}{
				"market_id":      trade.Symbol,
				"maker_order_id": trade.MakerOrderID,
				"taker_order_id": trade.TakerOrderID,
				"strike_price":   trade.Price,
				"amount":         trade.Quantity,
				"total":          trade.Total,
			},
		})
		config.Logger.Infof("%v, %v", trade.MakerOrderID, trade.TakerOrderID)
		mq_client.Enqueue("trade_executor", trade_message)
		// TODO: Fix finex trade_executor
		// if err := config.Nats.Publish("trade_executor", trade_message); err != nil {
		// 	return err
		// }
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
