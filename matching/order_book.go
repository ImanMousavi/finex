package matching

import (
	"encoding/json"
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
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
	this := a.(*OrderKey)
	that := b.(*OrderKey)

	if this.Side != that.Side {
		config.Logger.Errorf("[oceanbook.orderbook] compare order with different sides")
	}

	if this.ID == that.ID {
		return
	}

	// based on ask
	switch {
	case this.Side == SideSell && this.StopPrice.LessThan(that.StopPrice):
		result = 1

	case this.Side == SideSell && this.StopPrice.GreaterThan(that.StopPrice):
		result = -1

	case this.Side == SideBuy && this.StopPrice.LessThan(that.StopPrice):
		result = -1

	case this.Side == SideBuy && this.StopPrice.GreaterThan(that.StopPrice):
		result = 1

	default:
		switch {
		case this.CreatedAt.Before(that.CreatedAt):
			result = 1

		case this.CreatedAt.After(that.CreatedAt):
			result = -1

		default:
			result = utils.UInt64Comparator(this.ID, that.ID) * -1
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

func (o *OrderBook) setMarketPrice(newPrice decimal.Decimal) {
	previousPrice := o.MarketPrice
	o.MarketPrice = newPrice

	if previousPrice.IsZero() {
		return
	}

	switch {
	case newPrice.LessThan(previousPrice):
		// price gone done, check stop asks
		for {
			best := o.StopBids.Right()
			if best == nil {
				break
			}

			bestOrder := best.Value.(*Order)
			if bestOrder.StopPrice.LessThan(newPrice) {
				break
			}

			config.Logger.Debugf("[oceanbook.orderbook] bid order %d with stop price %s enqueued", bestOrder.ID, bestOrder.Price)

			o.StopBids.Remove(best.Key)
			o.pendingOrdersQueue.Push(bestOrder)
		}

	case newPrice.GreaterThan(previousPrice):
		// price gone done, check stop asks
		for {
			best := o.StopAsks.Right()
			if best == nil {
				break
			}

			bestOrder := best.Value.(*Order)
			if bestOrder.StopPrice.GreaterThan(newPrice) {
				break
			}

			config.Logger.Debugf("[oceanbook.orderbook] ask order %d with stop price %s enqueued", bestOrder.ID, bestOrder.Price)

			o.StopAsks.Remove(best.Key)
			o.pendingOrdersQueue.Push(bestOrder)
		}

	default:
		// previous price equals to new price
		return
	}
}

func (o *OrderBook) Add(order *Order) {
	o.orderMutex.Lock()
	defer o.orderMutex.Unlock()
	if order.StopPrice.IsPositive() {
		var book *redblacktree.Tree
		switch order.Side {
		case SideSell:
			book = o.StopAsks
		case SideBuy:
			book = o.StopBids
		}

		_, found := book.Get(order.Key())
		if found {
			return
		}

		book.Put(order.Key(), order)

		return
	}

	o.Match(order)

	pendingOrders := o.pendingOrdersQueue.Values()
	for i := range pendingOrders {
		pendingOrder := pendingOrders[i]

		config.Logger.Debugf("[oceanbook.orderbook] insert stop order with id %d - %s * %s, side %s", pendingOrder.ID, pendingOrder.Price, pendingOrder.Quantity, pendingOrder.Side)

		o.Match(pendingOrder)
	}
	o.pendingOrdersQueue.Clear()
}

func (o *OrderBook) Remove(order *Order) {
	o.orderMutex.Lock()
	defer o.orderMutex.Unlock()
	o.Depth.Remove(order)
	o.PublishCancel(order)
}

func (o *OrderBook) Match(order *Order) {
	o.matchMutex.Lock()
	defer o.matchMutex.Unlock()
	var offers *redblacktree.Tree

	if order.IsAsk() {
		offers = o.Depth.Bids
	} else {
		offers = o.Depth.Asks
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

		opposite_order := price_level.Top()
		quantity := decimal.Min(order.UnfilledQuantity(), opposite_order.UnfilledQuantity())

		if order.Type == TypeLimit {
			if !order.IsCrossed(opposite_order.Price) {
				break
			}
		} else if order.Type == TypeMarket {
			total := opposite_order.Price.Mul(quantity)

			if order.IsBid() && total.GreaterThan(order.Quantity) {
				order.Cancel()
				o.PublishCancel(order)

				break
			}
		}

		order.Fill(quantity)
		opposite_order.Fill(quantity)
		trade := &Trade{
			Symbol:       o.Symbol,
			Price:        opposite_order.Price,
			Quantity:     quantity,
			Total:        opposite_order.Price.Mul(quantity),
			MakerOrderID: order.ID,
			TakerOrderID: opposite_order.ID,
			MakerID:      order.MemberID,
			TakerID:      opposite_order.MemberID,
		}
		o.setMarketPrice(opposite_order.Price)
		if opposite_order.Filled() {
			o.Depth.Remove(opposite_order)
		} else {
			o.Depth.Add(opposite_order)
		}

		trade_message, _ := json.Marshal(trade)
		config.Nats.Publish("trade_executor", trade_message)

		if order.Filled() {
			break
		}
	}

	if order.UnfilledQuantity().IsPositive() {
		o.Depth.Add(order)
	}
}

func (o *OrderBook) PublishCancel(order *Order) error {
	order_processor_message, err := json.Marshal(map[string]interface{}{
		"action": ActionCancel,
		"order":  order,
	})

	if err != nil {
		return err
	}

	return config.Nats.Publish("order_processor", order_processor_message)
}
