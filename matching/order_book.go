package matching

import (
	"encoding/json"
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/pkg/order"
	"github.com/zsmartex/pkg/trade"
)

type OrderBook struct {
	matchMutex         sync.Mutex
	orderMutex         sync.Mutex
	Symbol             string
	MarketPrice        decimal.Decimal
	Depth              *Depth
	StopBids           *redblacktree.Tree
	StopAsks           *redblacktree.Tree
	pendingOrdersQueue *OrderQueue
}

const (
	// pendingOrdersCap is the buffer size for pending orders.
	pendingOrdersCap int64 = 1024
)

// StopComparator is used for comparing Key.
func StopComparator(a, b interface{}) (result int) {
	this := a.(*order.OrderKey)
	that := b.(*order.OrderKey)

	if this.Side != that.Side {
		config.Logger.Errorf("[oceanbook.orderbook] compare order with different sides")
	}

	if this.ID == that.ID {
		return
	}

	// based on ask
	switch {
	case this.Side == order.SideSell && this.StopPrice.LessThan(that.StopPrice):
		result = 1

	case this.Side == order.SideSell && this.StopPrice.GreaterThan(that.StopPrice):
		result = -1

	case this.Side == order.SideBuy && this.StopPrice.LessThan(that.StopPrice):
		result = -1

	case this.Side == order.SideBuy && this.StopPrice.GreaterThan(that.StopPrice):
		result = 1

	default:
		if this.CreatedAt.Before(that.CreatedAt) {
			result = 1
		} else {
			result = -1
		}
	}

	return
}

func NewOrderBook(symbol string, market_price decimal.Decimal) *OrderBook {
	return &OrderBook{
		Symbol:             symbol,
		MarketPrice:        market_price,
		Depth:              NewDepth(symbol),
		StopBids:           redblacktree.NewWith(StopComparator),
		StopAsks:           redblacktree.NewWith(StopComparator),
		pendingOrdersQueue: NewOrderQueue(pendingOrdersCap),
	}
}

func (ob *OrderBook) setMarketPrice(newPrice decimal.Decimal) {
	previousPrice := ob.MarketPrice
	ob.MarketPrice = newPrice

	if previousPrice.IsZero() {
		return
	}

	switch {
	case newPrice.LessThan(previousPrice):
		// price gone done, check stop asks
		for {
			best := ob.StopBids.Right()
			if best == nil {
				break
			}

			bestOrder := best.Value.(*order.Order)
			if bestOrder.StopPrice.LessThan(newPrice) {
				break
			}

			config.Logger.Debugf("[oceanbook.orderbook] bid order %d with stop price %s enqueued", bestOrder.ID, bestOrder.Price)

			ob.StopBids.Remove(best.Key)
			ob.pendingOrdersQueue.Push(bestOrder)
		}

	case newPrice.GreaterThan(previousPrice):
		// price gone done, check stop asks
		for {
			best := ob.StopAsks.Right()
			if best == nil {
				break
			}

			bestOrder := best.Value.(*order.Order)
			if bestOrder.StopPrice.GreaterThan(newPrice) {
				break
			}

			config.Logger.Debugf("[oceanbook.orderbook] ask order %d with stop price %s enqueued", bestOrder.ID, bestOrder.Price)

			ob.StopAsks.Remove(best.Key)
			ob.pendingOrdersQueue.Push(bestOrder)
		}

	default:
		// previous price equals to new price
		return
	}
}

func (ob *OrderBook) Add(o *order.Order) {
	ob.orderMutex.Lock()
	defer ob.orderMutex.Unlock()

	if o.StopPrice.IsPositive() {
		var book *redblacktree.Tree
		switch o.Side {
		case order.SideSell:
			book = ob.StopAsks
		case order.SideBuy:
			book = ob.StopBids
		}

		_, found := book.Get(o.Key())
		if found {
			return
		}

		book.Put(o.Key(), o)

		return
	}

	ob.Match(o)

	pendingOrders := ob.pendingOrdersQueue.Values()
	for i := range pendingOrders {
		pendingOrder := pendingOrders[i]

		config.Logger.Debugf("[oceanbook.orderbook] insert stop order with id %d - %s * %s, side %s", pendingOrder.ID, pendingOrder.Price, pendingOrder.Quantity, pendingOrder.Side)

		ob.Match(pendingOrder)
	}
	ob.pendingOrdersQueue.Clear()
}

func (ob *OrderBook) Remove(key *order.OrderKey) {
	ob.orderMutex.Lock()
	defer ob.orderMutex.Unlock()
	ob.Depth.Remove(key)
}

func (ob *OrderBook) Match(maker *order.Order) {
	ob.matchMutex.Lock()
	defer ob.matchMutex.Unlock()
	var offers *redblacktree.Tree

	if maker.IsAsk() {
		offers = ob.Depth.Bids
	} else {
		offers = ob.Depth.Asks
	}

	for {
		best := offers.Right()
		if best == nil {
			break
		}

		price_level := best.Value.(*PriceLevel)
		if price_level.Size() == 0 {
			continue
		}

		taker := price_level.Top()
		quantity := decimal.Min(maker.UnfilledQuantity(), taker.UnfilledQuantity())

		if maker.Type == order.TypeLimit {
			if !maker.IsCrossed(taker.Price) {
				break
			}
		}

		maker.Fill(quantity)
		taker.Fill(quantity)
		trade := &trade.Trade{
			Symbol:     ob.Symbol,
			Price:      taker.Price,
			Quantity:   quantity,
			Total:      taker.Price.Mul(quantity),
			MakerOrder: maker,
			TakerOrder: taker,
		}
		ob.setMarketPrice(taker.Price)
		if taker.Filled() {
			ob.Depth.Remove(taker.Key())
		} else {
			ob.Depth.Add(taker)
		}

		if taker.IsFake() {
			if order_message, err := json.Marshal(taker); err == nil {
				config.Nats.Publish("quantex:update:order", order_message)
			}
		}

		trade_message, _ := json.Marshal(trade)
		config.Nats.Publish("trade_executor", trade_message)

		if maker.Filled() {
			return
		}
	}

	if maker.UnfilledQuantity().IsPositive() {
		ob.Depth.Add(maker)
		if maker.IsFake() {
			if order_message, err := json.Marshal(maker); err == nil {
				config.Nats.Publish("quantex:update:order", order_message)
			}
		}
	}
}
