package matching

import (
	"sync"
	"time"

	"github.com/ericlagergren/decimal"
)

// Trade book stores all daily trades in-memory.
// It flushes new trades periodically to persistent storage. (TODO)
type TradeBook struct {
	Instrument string

	trades      map[uint64]Trade
	tradeMutex  sync.RWMutex
	lastTradeID uint64
}

// Create a new trade book.
func NewTradeBook(instrument string) *TradeBook {
	trade := &TradeBook{
		Instrument:  instrument,
		trades:      make(map[uint64]Trade),
		lastTradeID: 0,
	}

	go trade.LoopPublishTicker()

	return trade
}

// Enter a new trade.
func (t *TradeBook) Enter(trade Trade) {
	t.tradeMutex.Lock()
	defer t.tradeMutex.Unlock()

	t.lastTradeID += 1
	trade.ID = t.lastTradeID
	t.trades[t.lastTradeID] = trade
}

// Return all daily trades in a trade book.
func (t *TradeBook) DailyTrades() []Trade {
	t.tradeMutex.RLock()
	defer t.tradeMutex.RUnlock()

	yesterday := time.Now().AddDate(0, 0, -1)
	tradesCopy := make([]Trade, 0)
	var i uint64 = 0
	for _, trade := range t.trades {
		if trade.Timestamp.UnixNano() >= yesterday.UnixNano() {
			tradesCopy[i] = trade
			i += 1
		}
	}
	t.lastTradeID = i
	return tradesCopy
}

func (t *TradeBook) GetLatestTrade() Trade {
	return t.trades[t.lastTradeID]
}

func (t *TradeBook) LoopPublishTicker() {
	var old_amount decimal.Big
	var old_total decimal.Big

	for {
		time.Sleep(3 * time.Second)

		t.DailyTrades()

		if len(t.trades) == 0 {
			continue
		}

		trade := t.GetLatestTrade()

		if trade.Qty.Cmp(&old_amount) != 0 {
			// publish Qty
		}
		if trade.Total.Cmp(&old_total) != 0 {
			// publish total
		}
	}
}
