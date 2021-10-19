package matching

import (
	"encoding/json"
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/pkg"
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
	ob := &OrderBook{
		Symbol:             symbol,
		MarketPrice:        market_price,
		Depth:              NewDepth(symbol),
		StopBids:           redblacktree.NewWith(StopComparator),
		StopAsks:           redblacktree.NewWith(StopComparator),
		pendingOrdersQueue: NewOrderQueue(pendingOrdersCap),
	}

	ob.SubscribeCalcMarketOrder()

	return ob
}

type CalculateMarketOrder struct {
	Side     order.OrderSide     `json:"side"`
	Quantity decimal.NullDecimal `json:"quantity"`
	Volume   decimal.NullDecimal `json:"volume"`
}

type CalculateMarketOrderResult struct {
	Quantity decimal.Decimal `json:"quantity"`
	Locked   decimal.Decimal `json:"locked"`
}

func (ob *OrderBook) SubscribeCalcMarketOrder() {
	config.Nats.Subscribe("finex:calc_market_order:"+ob.Symbol, func(m *nats.Msg) {
		ob.orderMutex.Lock()
		defer ob.orderMutex.Unlock()

		var calc_market_order_payload *CalculateMarketOrder
		json.Unmarshal(m.Data, &calc_market_order_payload)

		side := calc_market_order_payload.Side
		quantity := calc_market_order_payload.Quantity
		volume := calc_market_order_payload.Volume

		var book *redblacktree.Tree
		if side == order.SideSell {
			book = ob.Depth.Bids
		} else {
			book = ob.Depth.Asks
		}

		var expected decimal.Decimal
		required := decimal.Zero
		if quantity.Valid {
			expected = quantity.Decimal
		} else {
			expected = volume.Decimal
		}

		if expected.IsZero() {
			payload, _ := json.Marshal(CalculateMarketOrderResult{
				Quantity: decimal.Zero,
				Locked:   decimal.Zero,
			})

			m.Respond(payload)
			return
		}

		order_quantity := decimal.Zero
		it := book.Iterator()
		for it.Next() && expected.IsPositive() {
			pl := it.Value().(*PriceLevel)

			if quantity.Valid {
				v := decimal.Min(pl.Total(), expected)
				order_quantity = order_quantity.Add(v)
				expected = expected.Sub(v)

				if pl.Side == order.SideSell {
					required = required.Add(pl.Price.Mul(v))
				} else {
					required = required.Add(v)
				}
			} else {
				// Not ready now
				v := decimal.Min(pl.Price.Mul(pl.Total()), expected)

				required = required.Add(v)
				expected = expected.Sub(v)
				order_quantity = order_quantity.Add(v.Div(pl.Price))
			}
		}

		if !expected.IsZero() {
			payload, _ := json.Marshal(CalculateMarketOrderResult{
				Quantity: decimal.Zero,
				Locked:   decimal.Zero,
			})

			m.Respond(payload)
			return
		}

		payload, _ := json.Marshal(CalculateMarketOrderResult{
			Quantity: order_quantity,
			Locked:   required,
		})

		m.Respond(payload)
	})
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
	ob.matchMutex.Lock()
	defer ob.orderMutex.Unlock()
	defer ob.matchMutex.Unlock()

	ob.Depth.Remove(key)

	if !key.Fake {
		ob.PublishCancel(key)
	}
}

func (ob *OrderBook) PublishCancel(key *order.OrderKey) error {
	order_processor_message, err := json.Marshal(map[string]interface{}{
		"action": pkg.ActionCancel,
		"id":     key.ID,
	})

	if err != nil {
		return err
	}

	return config.Nats.Publish("order_processor", order_processor_message)
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
			MakerOrder: *maker,
			TakerOrder: *taker,
		}

		if taker.Filled() || taker.Cancelled {
			ob.Depth.Remove(taker.Key())
		} else {
			ob.Depth.Add(taker)
		}
		ob.setMarketPrice(taker.Price)

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

	if maker.UnfilledQuantity().IsPositive() && maker.Type == order.TypeLimit {
		ob.Depth.Add(maker)
		if maker.IsFake() {
			if order_message, err := json.Marshal(maker); err == nil {
				config.Nats.Publish("quantex:update:order", order_message)
			}
		}
	}
}
