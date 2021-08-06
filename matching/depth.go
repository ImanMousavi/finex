package matching

import (
	"encoding/json"
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
)

type Depth struct {
	depthMutex   sync.Mutex
	Symbol       string
	Asks         *redblacktree.Tree
	Bids         *redblacktree.Tree
	Notification *Notification
}

func NewDepth(symbol string) *Depth {
	depth := &Depth{
		Symbol:       symbol,
		Asks:         redblacktree.NewWith(makeComparator),
		Bids:         redblacktree.NewWith(makeComparator),
		Notification: NewNotification(symbol),
	}

	depth.SubscribeFetch()

	return depth
}

func (d *Depth) Add(order *Order) {
	d.depthMutex.Lock()
	defer d.depthMutex.Unlock()
	var price_levels *redblacktree.Tree
	if order.Side == SideSell {
		price_levels = d.Asks
	} else {
		price_levels = d.Bids
	}

	pl := NewPriceLevel(order.Side, order.Price)

	value, found := price_levels.Get(pl.Key())

	if !found {
		pl.Add(order)
		price_levels.Put(pl.Key(), pl)
		d.Notification.Publish(order.Side, order.Price, pl.Total())
		return
	}

	price_level := value.(*PriceLevel)
	price_level.Add(order)
	d.Notification.Publish(order.Side, order.Price, price_level.Total())
}

func (d *Depth) Remove(order *Order) {
	d.depthMutex.Lock()
	defer d.depthMutex.Unlock()
	var price_levels *redblacktree.Tree
	if order.Side == SideSell {
		price_levels = d.Asks
	} else {
		price_levels = d.Bids
	}

	pl := NewPriceLevel(order.Side, order.Price)

	value, found := price_levels.Get(pl.Key())

	if !found {
		return
	}

	price_level := value.(*PriceLevel)
	price_level.Remove(order)
	d.Notification.Publish(order.Side, order.Price, price_level.Total())

	if price_level.Size() == 0 {
		price_levels.Remove(pl.Key())
	}
}

type GetDepthPayload struct {
	Market string `json:"market"`
	Limit  int    `json:"limit"`
}

func (d *Depth) SubscribeFetch() {
	config.Nats.Subscribe("finex:depth:"+d.Symbol, func(m *nats.Msg) {
		d.depthMutex.Lock()
		defer d.depthMutex.Unlock()

		var payload GetDepthPayload
		json.Unmarshal(m.Data, &payload)

		depth := DepthJSON{
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
	case aPriceLevel.Side == SideSell && aPriceLevel.Price.LessThan(bPriceLevel.Price):
		return 1

	case aPriceLevel.Side == SideSell && aPriceLevel.Price.GreaterThan(bPriceLevel.Price):
		return -1

	case aPriceLevel.Side == SideBuy && aPriceLevel.Price.LessThan(bPriceLevel.Price):
		return -1

	case aPriceLevel.Side == SideBuy && aPriceLevel.Price.GreaterThan(bPriceLevel.Price):
		return 1

	default:
		return 0
	}
}
