package matching

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/mq_client"
)

type DepthJSON struct {
	Asks     [][]decimal.Decimal `json:"asks"`
	Bids     [][]decimal.Decimal `json:"bids"`
	Sequence uint64              `json:"sequence"`
}

type Book struct {
	Asks map[decimal.Decimal]decimal.Decimal
	Bids map[decimal.Decimal]decimal.Decimal
}

type Notification struct {
	Symbol    string // instrument name
	Sequence  uint64
	Book      *Book // cache for fetch depth
	BookCache *Book // cache for notify to websocket
}

func NewNotification(symbol string) *Notification {
	notification := &Notification{
		Symbol:   symbol,
		Sequence: 0,
		BookCache: &Book{
			Asks: make(map[decimal.Decimal]decimal.Decimal),
			Bids: make(map[decimal.Decimal]decimal.Decimal),
		},
		Book: &Book{
			Asks: make(map[decimal.Decimal]decimal.Decimal),
			Bids: make(map[decimal.Decimal]decimal.Decimal),
		},
	}

	config.Redis.GetKey("finex:"+symbol+":depth:sequence", &notification.Sequence)
	notification.Start()
	notification.SubscribeFetch()

	return notification
}

func (n *Notification) Start() {
	go n.StartLoop()
}

func (n *Notification) SubscribeFetch() {
	config.Nats.Subscribe("depth:"+n.Symbol, func(m *nats.Msg) {
		asks_depth := make([][]decimal.Decimal, 0)
		bids_depth := make([][]decimal.Decimal, 0)

		for price, amount := range n.Book.Asks {
			asks_depth = append(asks_depth, []decimal.Decimal{price, amount})
		}
		for price, amount := range n.Book.Bids {
			bids_depth = append(bids_depth, []decimal.Decimal{price, amount})
		}

		depth_message, _ := json.Marshal(DepthJSON{
			Asks:     asks_depth,
			Bids:     bids_depth,
			Sequence: n.Sequence,
		})

		m.Respond(depth_message)
	})
}

func (n *Notification) StartLoop() {
	for {
		time.Sleep(100 * time.Millisecond)

		if len(n.BookCache.Asks) == 0 && len(n.BookCache.Bids) == 0 {
			continue
		}

		n.Sequence++
		config.Redis.SetKey("finex:"+n.Symbol+":depth:sequence", n.Sequence, redis.KeepTTL)

		asks_depth := make([][]decimal.Decimal, 0)
		bids_depth := make([][]decimal.Decimal, 0)

		for price, amount := range n.BookCache.Asks {
			asks_depth = append(asks_depth, []decimal.Decimal{price, amount})
		}
		for price, amount := range n.BookCache.Bids {
			bids_depth = append(bids_depth, []decimal.Decimal{price, amount})
		}

		payload := DepthJSON{
			Asks:     asks_depth,
			Bids:     bids_depth,
			Sequence: n.Sequence,
		}

		payload_message, _ := json.Marshal(payload)

		mq_client.EnqueueEvent("public", n.Symbol, "depth", payload_message)

		depth_cache_message, _ := json.Marshal(map[string]interface{}{
			"market": n.Symbol,
			"depth":  payload,
		})

		config.Nats.Publish("depth_cache", depth_cache_message)

		n.BookCache.Asks = make(map[decimal.Decimal]decimal.Decimal)
		n.BookCache.Bids = make(map[decimal.Decimal]decimal.Decimal)
	}
}

func (n *Notification) Publish(side Side, price, amount decimal.Decimal) {
	if side == SideBuy {
		for bprice, _ := range n.BookCache.Bids {
			if price.Equal(bprice) {
				delete(n.BookCache.Bids, bprice)
			}
		}

		n.BookCache.Bids[price] = amount

		for bprice, _ := range n.Book.Bids {
			if price.Equal(bprice) {
				delete(n.Book.Bids, bprice)
			}
		}

		if amount.IsPositive() {
			n.Book.Bids[price] = amount
		}
	} else {
		for bprice, _ := range n.BookCache.Asks {
			if price.Equal(bprice) {
				delete(n.BookCache.Asks, bprice)
			}
		}

		n.BookCache.Asks[price] = amount

		for bprice, _ := range n.Book.Asks {
			if price.Equal(bprice) {
				delete(n.Book.Asks, bprice)
			}
		}

		if amount.IsPositive() {
			n.Book.Asks[price] = amount
		}
	}
}
