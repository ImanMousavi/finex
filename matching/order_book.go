package matching

import (
	"os"
	"strconv"
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/pkg"
	GrpcEngine "github.com/zsmartex/pkg/Grpc/engine"
	GrpcOrder "github.com/zsmartex/pkg/Grpc/order"
	GrpcQuantex "github.com/zsmartex/pkg/Grpc/quantex"
	GrpcSymbol "github.com/zsmartex/pkg/Grpc/symbol"
	GrpcUtils "github.com/zsmartex/pkg/Grpc/utils"
	clientQuantex "github.com/zsmartex/pkg/client/quantex"
)

type OrderBook struct {
	matchMutex         sync.Mutex
	orderMutex         sync.Mutex
	Symbol             pkg.Symbol
	MarketPrice        decimal.Decimal
	Depth              *Depth
	StopBids           *redblacktree.Tree
	StopAsks           *redblacktree.Tree
	pendingOrdersQueue *OrderQueue
	quantexClient      *clientQuantex.GrpcQuantexClient
}

const (
	// pendingOrdersCap is the buffer size for pending orders.
	pendingOrdersCap int64 = 1024
)

// StopComparator is used for comparing Key.
func StopComparator(a, b interface{}) (result int) {
	this := a.(*pkg.OrderKey)
	that := b.(*pkg.OrderKey)

	if this.Side != that.Side {
		config.Logger.Errorf("[oceanbook.orderbook] compare order with different sides")
	}

	if this.ID == that.ID {
		return
	}

	// based on ask
	switch {
	case this.Side == pkg.SideSell && this.StopPrice.LessThan(that.StopPrice):
		result = 1

	case this.Side == pkg.SideSell && this.StopPrice.GreaterThan(that.StopPrice):
		result = -1

	case this.Side == pkg.SideBuy && this.StopPrice.LessThan(that.StopPrice):
		result = -1

	case this.Side == pkg.SideBuy && this.StopPrice.GreaterThan(that.StopPrice):
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

func NewOrderBook(symbol pkg.Symbol, market_price decimal.Decimal) *OrderBook {
	var quantex_client *clientQuantex.GrpcQuantexClient
	quantexEnabled, _ := strconv.ParseBool(os.Getenv("QUANTEX_ENABLED"))

	if quantexEnabled {
		quantex_client = clientQuantex.NewQuantexClient()
	}

	ob := &OrderBook{
		Symbol:             symbol,
		MarketPrice:        market_price,
		Depth:              NewDepth(symbol),
		StopBids:           redblacktree.NewWith(StopComparator),
		StopAsks:           redblacktree.NewWith(StopComparator),
		pendingOrdersQueue: NewOrderQueue(pendingOrdersCap),
		quantexClient:      quantex_client,
	}

	return ob
}

func (ob *OrderBook) CalcMarketOrder(side pkg.OrderSide, quantity decimal.NullDecimal, volume decimal.NullDecimal) *GrpcEngine.CalcMarketOrderResponse {
	zero := decimal.Zero

	var book *redblacktree.Tree
	if side == pkg.SideSell {
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
		return &GrpcEngine.CalcMarketOrderResponse{
			Quantity: &GrpcUtils.Decimal{
				Val: zero.CoefficientInt64(),
				Exp: zero.Exponent(),
			},
			Locked: &GrpcUtils.Decimal{
				Val: zero.CoefficientInt64(),
				Exp: zero.Exponent(),
			},
		}
	}

	order_quantity := decimal.Zero
	it := book.Iterator()
	for it.Next() && expected.IsPositive() {
		pl := it.Value().(*PriceLevel)

		if quantity.Valid {
			v := decimal.Min(pl.Total(), expected)
			order_quantity = order_quantity.Add(v)
			expected = expected.Sub(v)

			if pl.Side == pkg.SideSell {
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
		return &GrpcEngine.CalcMarketOrderResponse{
			Quantity: &GrpcUtils.Decimal{
				Val: zero.CoefficientInt64(),
				Exp: zero.Exponent(),
			},
			Locked: &GrpcUtils.Decimal{
				Val: zero.CoefficientInt64(),
				Exp: zero.Exponent(),
			},
		}
	}

	return &GrpcEngine.CalcMarketOrderResponse{
		Quantity: &GrpcUtils.Decimal{
			Val: order_quantity.CoefficientInt64(),
			Exp: order_quantity.Exponent(),
		},
		Locked: &GrpcUtils.Decimal{
			Val: required.CoefficientInt64(),
			Exp: required.Exponent(),
		},
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

			bestOrder := best.Value.(*pkg.Order)
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

			bestOrder := best.Value.(*pkg.Order)
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

func (ob *OrderBook) Add(o *pkg.Order) {
	ob.orderMutex.Lock()
	defer ob.orderMutex.Unlock()

	if o.StopPrice.IsPositive() {
		var book *redblacktree.Tree
		switch o.Side {
		case pkg.SideSell:
			book = ob.StopAsks
		case pkg.SideBuy:
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

func (ob *OrderBook) Remove(key *pkg.OrderKey) {
	ob.orderMutex.Lock()
	ob.matchMutex.Lock()
	defer ob.orderMutex.Unlock()
	defer ob.matchMutex.Unlock()

	ob.Depth.Remove(key)

	if !key.Fake {
		ob.PublishCancel(key)
	}
}

func (ob *OrderBook) PublishCancel(key *pkg.OrderKey) {
	config.KafkaProducer.Produce("order_processor", map[string]interface{}{
		"action": pkg.ActionCancel,
		"id":     key.ID,
	})
}

func (ob *OrderBook) Match(order *pkg.Order) {
	ob.matchMutex.Lock()
	defer ob.matchMutex.Unlock()
	var offers *redblacktree.Tree

	if order.IsAsk() {
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

		counter_order := price_level.Top()

		quantity := decimal.Min(order.UnfilledQuantity(), counter_order.UnfilledQuantity())

		if order.Type == pkg.TypeLimit {
			if !order.IsCrossed(counter_order.Price) {
				break
			}
		}

		order.Fill(quantity)
		counter_order.Fill(quantity)

		if counter_order.Filled() || counter_order.Cancelled {
			ob.Depth.Remove(counter_order.Key())
		} else {
			ob.Depth.Add(counter_order)
		}
		ob.setMarketPrice(counter_order.Price)

		if counter_order.IsFake() {
			if _, err := ob.quantexClient.UpdateOrder(&GrpcQuantex.UpdateOrderRequest{
				Order: &GrpcOrder.Order{
					Id:       counter_order.ID,
					Uuid:     counter_order.UUID[:],
					MemberId: counter_order.MemberID,
					Symbol:   &GrpcSymbol.Symbol{BaseCurrency: counter_order.Symbol.BaseCurrency, QuoteCurrency: counter_order.Symbol.QuoteCurrency},
					Side:     string(counter_order.Side),
					Type:     string(counter_order.Type),
					Price: &GrpcUtils.Decimal{
						Val: counter_order.Price.CoefficientInt64(),
						Exp: counter_order.Price.Exponent(),
					},
					StopPrice: &GrpcUtils.Decimal{
						Val: counter_order.StopPrice.CoefficientInt64(),
						Exp: counter_order.StopPrice.Exponent(),
					},
					Quantity: &GrpcUtils.Decimal{
						Val: counter_order.Quantity.CoefficientInt64(),
						Exp: counter_order.Quantity.Exponent(),
					},
					FilledQuantity: &GrpcUtils.Decimal{
						Val: counter_order.FilledQuantity.CoefficientInt64(),
						Exp: counter_order.FilledQuantity.Exponent(),
					},
					Fake:      counter_order.Fake,
					Cancelled: counter_order.Cancelled,
					CreatedAt: timestamppb.New(counter_order.CreatedAt),
				},
			}); err != nil {
				config.Logger.Errorf("[orderbook] update order %d failed: %s", counter_order.ID, err)
			}
		}

		trade := &pkg.Trade{
			Symbol:   ob.Symbol,
			Price:    counter_order.Price,
			Quantity: quantity,
			Total:    counter_order.Price.Mul(quantity),
		}

		ob.PublishTrade(order, counter_order, trade)

		if order.Filled() {
			return
		}
	}

	if order.UnfilledQuantity().IsPositive() && order.Type == pkg.TypeLimit {
		ob.Depth.Add(order)
		if order.IsFake() {
			if _, err := ob.quantexClient.UpdateOrder(&GrpcQuantex.UpdateOrderRequest{
				Order: &GrpcOrder.Order{
					Id:       order.ID,
					Uuid:     order.UUID[:],
					MemberId: order.MemberID,
					Symbol:   &GrpcSymbol.Symbol{BaseCurrency: order.Symbol.BaseCurrency, QuoteCurrency: order.Symbol.QuoteCurrency},
					Side:     string(order.Side),
					Type:     string(order.Type),
					Price: &GrpcUtils.Decimal{
						Val: order.Price.CoefficientInt64(),
						Exp: order.Price.Exponent(),
					},
					StopPrice: &GrpcUtils.Decimal{
						Val: order.StopPrice.CoefficientInt64(),
						Exp: order.StopPrice.Exponent(),
					},
					Quantity: &GrpcUtils.Decimal{
						Val: order.Quantity.CoefficientInt64(),
						Exp: order.Quantity.Exponent(),
					},
					FilledQuantity: &GrpcUtils.Decimal{
						Val: order.FilledQuantity.CoefficientInt64(),
						Exp: order.FilledQuantity.Exponent(),
					},
					Fake:      order.Fake,
					Cancelled: order.Cancelled,
					CreatedAt: timestamppb.New(order.CreatedAt),
				},
			}); err != nil {
				config.Logger.Errorf("[orderbook] update order %d failed: %s", order.ID, err)
			}
		}
	}
}

func (ob *OrderBook) PublishTrade(order, counter_order *pkg.Order, trade *pkg.Trade) {
	var maker_order pkg.Order
	var taker_order pkg.Order

	if order.CreatedAt.Before(counter_order.CreatedAt) {
		maker_order = *order
		taker_order = *counter_order
	} else {
		maker_order = *counter_order
		taker_order = *order
	}

	trade.MakerOrder = maker_order
	trade.TakerOrder = taker_order

	config.KafkaProducer.Produce("trade_executor", trade)
}
