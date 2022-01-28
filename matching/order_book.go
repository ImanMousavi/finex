package matching

import (
	"encoding/json"
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
	GrpcUtils "github.com/zsmartex/pkg/Grpc/utils"
	clientQuantex "github.com/zsmartex/pkg/client/quantex"
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
	quantexClient      *clientQuantex.GrpcQuantexClient
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

func (ob *OrderBook) GetOrder()

func (ob *OrderBook) CalcMarketOrder(side order.OrderSide, quantity decimal.NullDecimal, volume decimal.NullDecimal) *GrpcEngine.CalcMarketOrderResponse {
	zero := decimal.Zero

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

	return config.Kafka.Publish("order_processor", order_processor_message)
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
			ob.quantexClient.UpdateOrder(&GrpcQuantex.UpdateOrderRequest{
				Order: &GrpcOrder.Order{
					Id:       taker.ID,
					Uuid:     taker.UUID[:],
					MemberId: taker.MemberID,
					Symbol:   taker.Symbol,
					Side:     string(taker.Side),
					Type:     string(taker.Type),
					Price: &GrpcUtils.Decimal{
						Val: taker.Price.CoefficientInt64(),
						Exp: taker.Price.Exponent(),
					},
					StopPrice: &GrpcUtils.Decimal{
						Val: taker.StopPrice.CoefficientInt64(),
						Exp: taker.StopPrice.Exponent(),
					},
					Quantity: &GrpcUtils.Decimal{
						Val: taker.Quantity.CoefficientInt64(),
						Exp: taker.Quantity.Exponent(),
					},
					FilledQuantity: &GrpcUtils.Decimal{
						Val: taker.FilledQuantity.CoefficientInt64(),
						Exp: taker.FilledQuantity.Exponent(),
					},
					Fake:      taker.Fake,
					Cancelled: taker.Cancelled,
					CreatedAt: timestamppb.New(taker.CreatedAt),
				},
			})
		}

		trade_message, _ := json.Marshal(trade)
		config.Kafka.Publish("trade_executor", trade_message)

		if maker.Filled() {
			return
		}
	}

	if maker.UnfilledQuantity().IsPositive() && maker.Type == order.TypeLimit {
		ob.Depth.Add(maker)
		if maker.IsFake() {
			ob.quantexClient.UpdateOrder(&GrpcQuantex.UpdateOrderRequest{
				Order: &GrpcOrder.Order{
					Id:       maker.ID,
					Uuid:     maker.UUID[:],
					MemberId: maker.MemberID,
					Symbol:   maker.Symbol,
					Side:     string(maker.Side),
					Type:     string(maker.Type),
					Price: &GrpcUtils.Decimal{
						Val: maker.Price.CoefficientInt64(),
						Exp: maker.Price.Exponent(),
					},
					StopPrice: &GrpcUtils.Decimal{
						Val: maker.StopPrice.CoefficientInt64(),
						Exp: maker.StopPrice.Exponent(),
					},
					Quantity: &GrpcUtils.Decimal{
						Val: maker.Quantity.CoefficientInt64(),
						Exp: maker.Quantity.Exponent(),
					},
					FilledQuantity: &GrpcUtils.Decimal{
						Val: maker.FilledQuantity.CoefficientInt64(),
						Exp: maker.FilledQuantity.Exponent(),
					},
					Fake:      maker.Fake,
					Cancelled: maker.Cancelled,
					CreatedAt: timestamppb.New(maker.CreatedAt),
				},
			})
		}
	}
}
