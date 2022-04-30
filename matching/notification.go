package matching

import (
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/pkg"
)

type Book struct {
	Asks [][]decimal.Decimal
	Bids [][]decimal.Decimal
}

type Notification struct {
	Symbol    pkg.Symbol // instrument name
	Sequence  int64
	BookCache *Book // cache for notify to websocket

	NotifyMutex sync.RWMutex
}

func NewNotification(symbol pkg.Symbol) *Notification {
	notification := &Notification{
		Symbol:   symbol,
		Sequence: 0,
		BookCache: &Book{
			Asks: make([][]decimal.Decimal, 0),
			Bids: make([][]decimal.Decimal, 0),
		},
	}

	result, err := config.Redis.Get("finex:" + strings.ToLower(symbol.ToSymbol("")) + ":depth:sequence")
	if err != nil {
		panic(err)
	}

	sq, err := result.Int64()
	if err != nil {
		panic(err)
	}

	notification.Sequence = sq

	notification.Start()

	return notification
}

func (n *Notification) Start() {
	go n.StartLoop()
}

func (n *Notification) StartLoop() {
	for {
		time.Sleep(100 * time.Millisecond)

		if len(n.BookCache.Asks) == 0 && len(n.BookCache.Bids) == 0 {
			continue
		}

		n.NotifyMutex.Lock()

		n.Sequence++
		config.Redis.Set("finex:"+strings.ToLower(n.Symbol.ToSymbol(""))+":depth:sequence", n.Sequence, 0)

		asks_depth := make([][]decimal.Decimal, 0)
		bids_depth := make([][]decimal.Decimal, 0)

		asks_depth = append(asks_depth, n.BookCache.Asks...)
		bids_depth = append(bids_depth, n.BookCache.Bids...)

		config.RangoClient.EnqueueEvent(pkg.EnqueueEventKindPublic, strings.ToLower(n.Symbol.ToSymbol("")), "depth", pkg.DepthJSON{
			Asks:     asks_depth,
			Bids:     bids_depth,
			Sequence: n.Sequence,
		})

		n.BookCache.Asks = make([][]decimal.Decimal, 0)
		n.BookCache.Bids = make([][]decimal.Decimal, 0)
		n.NotifyMutex.Unlock()
	}
}

func (n *Notification) Publish(side pkg.OrderSide, price, amount decimal.Decimal) {
	n.NotifyMutex.Lock()
	defer n.NotifyMutex.Unlock()

	if side == pkg.SideBuy {
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
