package matching

import (
	"sync"

	"github.com/shopspring/decimal"
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
}
