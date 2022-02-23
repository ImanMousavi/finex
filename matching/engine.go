package matching

import (
	"sync"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/pkg"
)

type Engine struct {
	MatchingMutex sync.RWMutex
	Symbol        pkg.Symbol
	OrderBook     *OrderBook
	Initialized   bool
}

func NewEngine(symbol pkg.Symbol, price decimal.Decimal) *Engine {
	engine := &Engine{
		Symbol: symbol,
		OrderBook: NewOrderBook(
			symbol,
			price,
		),
		Initialized: false,
	}

	return engine
}

func (e *Engine) Submit(o *pkg.Order) {
	e.MatchingMutex.Lock()
	defer e.MatchingMutex.Unlock()

	e.OrderBook.Add(o)
}

func (e *Engine) CancelWithKey(key *pkg.OrderKey) {
	e.MatchingMutex.Lock()
	defer e.MatchingMutex.Unlock()

	e.OrderBook.Remove(key)
}

func (e *Engine) Cancel(o *pkg.Order) {
	e.CancelWithKey(o.Key())
}
