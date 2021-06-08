package matching

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
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
	Symbol   string // instrument name
	Sequence uint64
	Book     *Book
}

func NewNotification(symbol string) *Notification {
	notification := &Notification{
		Symbol:   symbol,
		Sequence: 0,
		Book: &Book{
			Asks: [][]decimal.Decimal{},
			Bids: [][]decimal.Decimal{},
		},
	}

	config.Redis.GetKey("finex:"+symbol+":depth:sequence", &notification.Sequence)
	notification.Start()

	return notification
}

func (n *Notification) Start() {
	go n.StartLoop()
}

func (n *Notification) StartLoop() {
	for {
		time.Sleep(100 * time.Millisecond)

		if len(n.Book.Asks) == 0 && len(n.Book.Bids) == 0 {
			continue
		}

		n.Sequence++
		config.Redis.SetKey("finex:"+n.Symbol+":depth:sequence", n.Sequence, redis.KeepTTL)

		payload := DepthJSON{
			Asks:     n.Book.Asks,
			Bids:     n.Book.Bids,
			Sequence: n.Sequence,
		}

		payload_message, _ := json.Marshal(payload)

		mq_client.EnqueueEvent("public", n.Symbol, "depth", payload_message)

		depth_cache_message, _ := json.Marshal(map[string]interface{}{
			"market": n.Symbol,
			"depth":  payload,
		})

		mq_client.Enqueue("depth_cache", depth_cache_message)

		n.Book.Asks = [][]decimal.Decimal{}
		n.Book.Bids = [][]decimal.Decimal{}
	}
}

func (n *Notification) Publish(side Side, price, amount decimal.Decimal) {
	if side == SideBuy {
		for i, item := range n.Book.Bids {
			if price.Equal(item[0]) {
				n.Book.Bids = append(n.Book.Bids[:i], n.Book.Bids[i+1:]...)
			}
		}

		n.Book.Bids = append(n.Book.Bids, []decimal.Decimal{price, amount})
	} else {
		for i, item := range n.Book.Asks {
			if price.Equal(item[0]) {
				n.Book.Asks = append(n.Book.Asks[:i], n.Book.Asks[i+1:]...)
			}
		}

		n.Book.Asks = append(n.Book.Asks, []decimal.Decimal{price, amount})
	}
}
