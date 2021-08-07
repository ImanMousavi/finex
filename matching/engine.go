package matching

import (
	"sync"

	"github.com/shopspring/decimal"
)

type PayloadAction = string

var (
	ActionSubmit PayloadAction = "submit"
	ActionCancel PayloadAction = "cancel"
	ActionReload PayloadAction = "reload"
)

type Engine struct {
	MatchingMutex sync.RWMutex
	Market        string
	OrderBook     *OrderBook
	Initialized   bool
}

func NewEngine(market string, price decimal.Decimal) *Engine {
	return &Engine{
		Market: market,
		OrderBook: NewOrderBook(
			market,
			price,
		),
		Initialized: false,
	}
}

func (e *Engine) Submit(order *Order) {
	e.MatchingMutex.Lock()
	defer e.MatchingMutex.Unlock()
	e.OrderBook.Add(order)
}

func (e *Engine) Cancel(order *Order) {
	e.MatchingMutex.Lock()
	defer e.MatchingMutex.Unlock()

	e.OrderBook.Remove(order)
}
