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
	Asks [][]decimal.Decimal
	Bids [][]decimal.Decimal
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
			Asks: [][]decimal.Decimal{},
			Bids: [][]decimal.Decimal{},
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
	config.Nats.Subscribe("fetch_depth", func(m *nats.Msg) {
		depth_message, _ := json.Marshal(map[string]interface{}{
			"market": n.Symbol,
			"depth": DepthJSON{
				Asks:     n.Book.Asks,
				Bids:     n.Book.Bids,
				Sequence: n.Sequence,
			},
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

		payload := DepthJSON{
			Asks:     n.BookCache.Asks,
			Bids:     n.BookCache.Bids,
			Sequence: n.Sequence,
		}

		payload_message, _ := json.Marshal(payload)

		mq_client.EnqueueEvent("public", n.Symbol, "depth", payload_message)

		depth_cache_message, _ := json.Marshal(map[string]interface{}{
			"market": n.Symbol,
			"depth":  payload,
		})

		config.Nats.Publish("depth_cache", depth_cache_message)

		n.BookCache.Asks = [][]decimal.Decimal{}
		n.BookCache.Bids = [][]decimal.Decimal{}
	}
}

func (n *Notification) Publish(side Side, price, amount decimal.Decimal) {
	if side == SideBuy {
		for i, item := range n.BookCache.Bids {
			if price.Equal(item[0]) {
				n.BookCache.Bids = append(n.BookCache.Bids[:i], n.BookCache.Bids[i+1:]...)
			}
		}

		n.BookCache.Bids = append(n.BookCache.Bids, []decimal.Decimal{price, amount})

		for i, item := range n.Book.Bids {
			if price.Equal(item[0]) {
				n.BookCache.Bids = append(n.Book.Bids[:i], n.Book.Bids[i+1:]...)
			}
		}

		if amount.IsPositive() {
			n.Book.Bids = append(n.Book.Bids, []decimal.Decimal{price, amount})
		}
	} else {
		for i, item := range n.BookCache.Asks {
			if price.Equal(item[0]) {
				n.BookCache.Asks = append(n.BookCache.Asks[:i], n.BookCache.Asks[i+1:]...)
			}
		}

		n.BookCache.Asks = append(n.BookCache.Asks, []decimal.Decimal{price, amount})

		for i, item := range n.Book.Asks {
			if price.Equal(item[0]) {
				n.BookCache.Asks = append(n.Book.Asks[:i], n.Book.Asks[i+1:]...)
			}
		}

		if amount.IsPositive() {
			n.Book.Asks = append(n.Book.Asks, []decimal.Decimal{price, amount})
		}
	}
}
