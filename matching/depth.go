package matching

import (
	"encoding/json"
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/pkg"
	"github.com/zsmartex/pkg/order"
)

type Depth struct {
	depthMutex       sync.Mutex
	fetchOrdersMutex sync.Mutex

	Symbol       string
	Asks         *redblacktree.Tree
	Bids         *redblacktree.Tree
	Notification *Notification
	fakeOrders   map[uuid.UUID]*order.OrderKey
}

func NewDepth(symbol string) *Depth {
	depth := &Depth{
		Symbol:       symbol,
		Asks:         redblacktree.NewWith(makeComparator),
		Bids:         redblacktree.NewWith(makeComparator),
		Notification: NewNotification(symbol),
		fakeOrders:   make(map[uuid.UUID]*order.OrderKey),
	}

	depth.SubscribeFetch()

	return depth
}

func (d *Depth) Add(o *order.Order) {
	d.depthMutex.Lock()
	defer d.depthMutex.Unlock()
	var price_levels *redblacktree.Tree
	if o.Side == order.SideSell {
		price_levels = d.Asks
	} else {
		price_levels = d.Bids
	}

	d.fetchOrdersMutex.Lock()
	if o.Fake {
		d.fakeOrders[o.UUID] = o.Key()
	}
	d.fetchOrdersMutex.Unlock()

	pl := NewPriceLevel(o.Side, o.Price)

	value, found := price_levels.Get(pl.Key())

	if !found {
		pl.Add(o)
		price_levels.Put(pl.Key(), pl)
		d.Notification.Publish(pl.Side, pl.Price, pl.Total())
		return
	}

	price_level := value.(*PriceLevel)
	price_level.Add(o)
	d.Notification.Publish(price_level.Side, price_level.Price, price_level.Total())
}

func (d *Depth) Remove(key *order.OrderKey) {
	d.depthMutex.Lock()
	defer d.depthMutex.Unlock()
	var price_levels *redblacktree.Tree
	if key.Side == order.SideSell {
		price_levels = d.Asks
	} else {
		price_levels = d.Bids
	}

	d.fetchOrdersMutex.Lock()
	if _, ok := d.fakeOrders[key.UUID]; !ok && key.Fake {
		delete(d.fakeOrders, key.UUID)
	}
	d.fetchOrdersMutex.Unlock()

	pl := NewPriceLevel(key.Side, key.Price)

	value, found := price_levels.Get(pl.Key())

	if !found {
		return
	}

	price_level := value.(*PriceLevel)
	price_level.Remove(key)
	d.Notification.Publish(price_level.Side, price_level.Price, price_level.Total())

	if price_level.Size() == 0 {
		price_levels.Remove(pl.Key())
	}
}

func (d *Depth) SubscribeFetch() {
	config.Nats.Subscribe("finex:order:"+d.Symbol, func(m *nats.Msg) {
		d.fetchOrdersMutex.Lock()
		d.depthMutex.Lock()
		defer d.fetchOrdersMutex.Unlock()
		defer d.depthMutex.Unlock()

		var payload pkg.GetFakeOrderPayload

		key, ok := d.fakeOrders[payload.UUID]
		if !ok {
			order_message, _ := json.Marshal(nil)
			m.Respond(order_message)
			return
		}

		var price_levels *redblacktree.Tree
		if key.Side == order.SideSell {
			price_levels = d.Asks
		} else {
			price_levels = d.Bids
		}
		pl := NewPriceLevel(key.Side, key.Price)
		value, found := price_levels.Get(pl)
		if !found {
			order_message, _ := json.Marshal(nil)
			m.Respond(order_message)
			return
		}

		price_level := value.(*PriceLevel)
		order := price_level.Get(key)

		order_message, _ := json.Marshal(order)
		m.Respond(order_message)
	})

	config.Nats.Subscribe("finex:depth:"+d.Symbol, func(m *nats.Msg) {
		d.depthMutex.Lock()
		defer d.depthMutex.Unlock()

		var payload pkg.GetDepthPayload
		json.Unmarshal(m.Data, &payload)

		depth := pkg.DepthJSON{
			Asks:     make([][]decimal.Decimal, 0),
			Bids:     make([][]decimal.Decimal, 0),
			Sequence: 0,
		}

		ait := d.Asks.Iterator()
		ait.End()
		for i := 0; ait.Prev() && i < payload.Limit; i++ {
			price_level := ait.Value().(*PriceLevel)
			depth.Asks = append(depth.Asks, []decimal.Decimal{price_level.Price, price_level.Total()})
		}

		bit := d.Bids.Iterator()
		bit.End()
		for i := 0; bit.Prev() && i < payload.Limit; i++ {
			price_level := bit.Value().(*PriceLevel)
			depth.Bids = append(depth.Bids, []decimal.Decimal{price_level.Price, price_level.Total()})
		}
		depth.Sequence = d.Notification.Sequence

		depth_message, err := json.Marshal(depth)

		if err != nil {
			config.Logger.Errorf("Error: %s", err.Error())
		}

		m.Respond(depth_message)
	})
}

func makeComparator(a, b interface{}) int {
	aPriceLevel := a.(*PriceLevelKey)
	bPriceLevel := b.(*PriceLevelKey)

	switch {
	case aPriceLevel.Side == order.SideSell && aPriceLevel.Price.LessThan(bPriceLevel.Price):
		return 1

	case aPriceLevel.Side == order.SideSell && aPriceLevel.Price.GreaterThan(bPriceLevel.Price):
		return -1

	case aPriceLevel.Side == order.SideBuy && aPriceLevel.Price.LessThan(bPriceLevel.Price):
		return -1

	case aPriceLevel.Side == order.SideBuy && aPriceLevel.Price.GreaterThan(bPriceLevel.Price):
		return 1

	default:
		return 0
	}
}
