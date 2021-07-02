package matching

import (
	"sync"

	// log level and settings
	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/go-finex/config"
)

// OrderBook is the order book.
type OrderBook struct {
	sync.RWMutex
	Symbol string
	Price  decimal.Decimal

	Bids     *rbt.Tree
	Asks     *rbt.Tree
	StopBids *rbt.Tree
	StopAsks *rbt.Tree

	pendingOrdersQueue *OrderQueue
	cancelOrdersQueue  map[uint64]*Order

	notification *Notification
	depth        *Depth
}

const (
	// pendingOrdersCap is the buffer size for pending orders.
	pendingOrdersCap int64 = 1024
)

// NewOrderBook returns a pointer to an orderbook.
func NewOrderBook(symbol string) *OrderBook {
	orderQueue := NewOrderQueue(pendingOrdersCap)
	notification := NewNotification(symbol)
	return &OrderBook{
		Symbol:             symbol,
		Bids:               rbt.NewWith(Comparator),
		Asks:               rbt.NewWith(Comparator),
		StopBids:           rbt.NewWith(StopComparator),
		StopAsks:           rbt.NewWith(StopComparator),
		pendingOrdersQueue: &orderQueue,
		cancelOrdersQueue:  make(map[uint64]*Order, 1024),
		notification:       notification,
		depth:              NewDepth(symbol, notification),
	}
}

// InsertOrder inserts new order into orderbook.
func (od *OrderBook) InsertOrder(newOrder *Order) []*Trade {
	od.Lock()
	defer od.Unlock()

	config.Logger.Debugf("[oceanbook.orderbook] insert order with id %d - %s * %s, side %s", newOrder.ID, newOrder.Price, newOrder.Quantity, newOrder.Side)

	if newOrder.StopPrice.IsPositive() {
		od.insertStopOrder(newOrder)

		return []*Trade{}
	}

	trades := od.insertOrder(newOrder)

	pendingOrders := od.pendingOrdersQueue.Values()
	for i := range pendingOrders {
		pendingOrder := pendingOrders[i]

		config.Logger.Debugf("[oceanbook.orderbook] insert stop order with id %d - %s * %s, side %s", pendingOrder.ID, pendingOrder.Price, pendingOrder.Quantity, pendingOrder.Side)

		newTrades := od.insertOrder(pendingOrder)
		trades = append(trades, newTrades...)
	}
	od.pendingOrdersQueue.Clear()

	return trades
}

func (od *OrderBook) insertOrder(newOrder *Order) []*Trade {
	trades := []*Trade{}

	var takerBooks, makerBooks *rbt.Tree
	switch newOrder.Side {
	case SideSell:
		takerBooks = od.Asks
		makerBooks = od.Bids
	case SideBuy:
		takerBooks = od.Bids
		makerBooks = od.Asks
	default:
		config.Logger.Errorf("[oceanbook.orderbook] invalid order side %s", newOrder.Side)
		return trades
	}

	_, found := takerBooks.Get(newOrder.Key())
	if found {
		return trades
	}

	for {
		if newOrder == nil {
			break
		}

		best := makerBooks.Right()
		if best == nil {
			break
		}

		bestOrder := best.Value.(*Order)
		newTrade := bestOrder.Match(newOrder)

		if newTrade == nil {
			break
		}

		trades = append(trades, newTrade)
		config.Logger.Debugf("[oceanbook.orderbook] new trade with price %s", newTrade.Price)

		var count int32
		if bestOrder.Filled() {
			count = -1
		} else {
			count = 0
		}

		od.depth.UpdatePriceLevel(
			bestOrder.Side,
			newTrade.Price,
			newTrade.Quantity.Neg(),
			count,
		)

		if bestOrder.Filled() {
			makerBooks.Remove(bestOrder.Key())
			delete(od.cancelOrdersQueue, bestOrder.ID)
		}

		od.setMarketPrice(newTrade.Price)

		if newOrder.Filled() {
			return trades
		}
	}

	// if the order is immediate or cancel order, it is not supposed to insert
	// into the orderbooks.
	if newOrder.ImmediateOrCancel {
		return trades
	}

	od.depth.UpdatePriceLevel(
		newOrder.Side,
		newOrder.Price,
		newOrder.PendingQuantity(),
		1,
	)
	takerBooks.Put(newOrder.Key(), newOrder)
	od.cancelOrdersQueue[newOrder.ID] = newOrder

	return trades
}

func (od *OrderBook) insertStopOrder(newOrder *Order) {
	var takerBooks *rbt.Tree
	switch newOrder.Side {
	case SideSell:
		takerBooks = od.StopAsks

	case SideBuy:
		takerBooks = od.StopBids

	default:
		config.Logger.Errorf("[oceanbook.orderbook] invalid stop order side %s", newOrder.Side)
		return
	}

	_, found := takerBooks.Get(newOrder.Key())
	if found {
		return
	}

	takerBooks.Put(newOrder.Key(), newOrder)
}

func (od *OrderBook) setMarketPrice(newPrice decimal.Decimal) {
	previousPrice := od.Price
	od.Price = newPrice

	if previousPrice.IsZero() {
		return
	}

	switch {
	case newPrice.LessThan(previousPrice):
		// price gone done, check stop asks
		for {
			best := od.StopBids.Right()
			if best == nil {
				break
			}

			bestOrder := best.Value.(*Order)
			if bestOrder.StopPrice.LessThan(newPrice) {
				break
			}

			config.Logger.Debugf("[oceanbook.orderbook] bid order %d with stop price %s enqueued", bestOrder.ID, bestOrder.Price)

			od.StopBids.Remove(best.Key)
			od.pendingOrdersQueue.Push(bestOrder)
		}

	case newPrice.GreaterThan(previousPrice):
		// price gone done, check stop asks
		for {
			best := od.StopAsks.Right()
			if best == nil {
				break
			}

			bestOrder := best.Value.(*Order)
			if bestOrder.StopPrice.GreaterThan(newPrice) {
				break
			}

			config.Logger.Debugf("[oceanbook.orderbook] ask order %d with stop price %s enqueued", bestOrder.ID, bestOrder.Price)

			od.StopAsks.Remove(best.Key)
			od.pendingOrdersQueue.Push(bestOrder)
		}

	default:
		// previous price equals to new price
		return
	}
}

// CancelOrder removes order with specified id.
func (od *OrderBook) CancelOrder(o *Order) {
	od.Lock()
	defer od.Unlock()

	targetOrder, ok := od.cancelOrdersQueue[o.ID]
	if !ok {
		return
	}

	switch targetOrder.Side {
	case SideSell:
		od.Asks.Remove(targetOrder.Key())

	case SideBuy:
		od.Bids.Remove(targetOrder.Key())

	default:
		od.Asks.Remove(targetOrder.Key())
		od.Bids.Remove(targetOrder.Key())
	}

	od.depth.UpdatePriceLevel(
		o.Side,
		o.Price,
		o.PendingQuantity().Neg(),
		-1,
	)
}

// GetDepth returns the order book depth.
func (od *OrderBook) GetDepth() *Depth {
	od.RLock()
	defer od.RUnlock()
	return od.depth
}
