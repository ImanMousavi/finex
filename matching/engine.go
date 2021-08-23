package matching

import (
	"encoding/json"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/pkg"
	"github.com/zsmartex/pkg/order"
)

type Engine struct {
	MatchingMutex sync.RWMutex
	Market        string
	OrderBook     *OrderBook
	Initialized   bool
}

func NewEngine(market string, price decimal.Decimal) *Engine {
	engine := &Engine{
		Market: market,
		OrderBook: NewOrderBook(
			market,
			price,
		),
		Initialized: false,
	}

	// NOTE: dont care about this it's only work for quantex-bot
	config.Nats.Subscribe("finex:"+market+":get_price", func(n *nats.Msg) {
		price_message, _ := json.Marshal(engine.OrderBook.MarketPrice)

		n.Respond(price_message)
	})

	return engine
}

func (e *Engine) Submit(o *order.Order) {
	e.MatchingMutex.Lock()
	defer e.MatchingMutex.Unlock()

	e.OrderBook.Add(o)
}

func (e *Engine) CancelWithKey(key *order.OrderKey) {
	e.MatchingMutex.Lock()
	defer e.MatchingMutex.Unlock()

	e.OrderBook.Remove(key)
}

func (e *Engine) Cancel(o *order.Order) {
	e.CancelWithKey(o.Key())
	e.PublishCancel(o)
}

func (e *Engine) PublishCancel(o *order.Order) error {
	order_processor_message, err := json.Marshal(map[string]interface{}{
		"action": pkg.ActionCancel,
		"order":  o,
	})

	if err != nil {
		return err
	}

	return config.Nats.Publish("order_processor", order_processor_message)
}
