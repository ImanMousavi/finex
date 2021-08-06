package matching

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/mq_client"
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
	Symbol      string // instrument name
	Sequence    uint64
	BookCache   *Book // cache for notify to websocket
	NotifyMutex sync.RWMutex
}

func NewNotification(symbol string) *Notification {
	notification := &Notification{
		Symbol:   symbol,
		Sequence: 0,
		BookCache: &Book{
			Asks: make([][]decimal.Decimal, 0),
			Bids: make([][]decimal.Decimal, 0),
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
	amqp_connection, err := mq_client.CreateAMQP()
	if err != nil {
		return
	}

	channel, err := amqp_connection.Channel()
	if err != nil {
		return
	}

	for {
		time.Sleep(100 * time.Millisecond)

		if len(n.BookCache.Asks) == 0 && len(n.BookCache.Bids) == 0 {
			continue
		}

		n.NotifyMutex.Lock()

		n.Sequence++
		config.Redis.SetKey("finex:"+n.Symbol+":depth:sequence", n.Sequence, 0)

		asks_depth := make([][]decimal.Decimal, 0)
		bids_depth := make([][]decimal.Decimal, 0)

		asks_depth = append(asks_depth, n.BookCache.Asks...)
		bids_depth = append(bids_depth, n.BookCache.Bids...)

		payload := DepthJSON{
			Asks:     asks_depth,
			Bids:     bids_depth,
			Sequence: n.Sequence,
		}

		payload_message, err := json.Marshal(payload)

		if err != nil {
			config.Logger.Errorf("Error: %s", err.Error())
		}

		if err := mq_client.ChanEnqueueEvent(channel, "public", n.Symbol, "depth", payload_message); err != nil {
			config.Logger.Errorf("Error: %s", err.Error())
		}

		n.BookCache.Asks = make([][]decimal.Decimal, 0)
		n.BookCache.Bids = make([][]decimal.Decimal, 0)
		n.NotifyMutex.Unlock()
	}
}

func (n *Notification) Publish(side OrderSide, price, amount decimal.Decimal) {
	n.NotifyMutex.Lock()
	defer n.NotifyMutex.Unlock()

	if side == SideBuy {
		for _, o := range n.BookCache.Bids {
			if o[0].Equal(price) {
				o[1] = amount

				return
			}
		}

		n.BookCache.Bids = append(n.BookCache.Bids, []decimal.Decimal{price, amount})
	} else {
		for _, o := range n.BookCache.Asks {
			if o[0].Equal(price) {
				o[1] = amount

				return
			}
		}

		n.BookCache.Asks = append(n.BookCache.Asks, []decimal.Decimal{price, amount})
	}
}
