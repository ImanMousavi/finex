package matching

import (
	"strings"
	"sync"
	"time"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/shopspring/decimal"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/pkg"
	GrpcEngine "github.com/zsmartex/pkg/Grpc/engine"
	GrpcSymbol "github.com/zsmartex/pkg/Grpc/symbol"
	GrpcUtils "github.com/zsmartex/pkg/Grpc/utils"
)

var MIN_PERIOD_TO_SNAPSHOT = 10 * time.Second
var MAX_PERIOD_TO_SNAPSHOT = 60 * time.Second
var MIN_INCREMENT_COUNT_TO_SNAPSHOT int64 = 20

type Depth struct {
	depthMutex sync.RWMutex

	Symbol       pkg.Symbol
	Asks         *redblacktree.Tree
	Bids         *redblacktree.Tree
	Notification *Notification

	// default peatio ws
	SnapshotTime   time.Time
	IncrementCount int64
	// close
}

func NewDepth(symbol pkg.Symbol) *Depth {
	depth := &Depth{
		Symbol:       symbol,
		Asks:         redblacktree.NewWith(makeComparator),
		Bids:         redblacktree.NewWith(makeComparator),
		Notification: NewNotification(symbol),
	}

	return depth
}

func (d *Depth) Add(o *pkg.Order) {
	d.depthMutex.Lock()
	defer d.depthMutex.Unlock()
	var price_levels *redblacktree.Tree
	if o.Side == pkg.SideSell {
		price_levels = d.Asks
	} else {
		price_levels = d.Bids
	}

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

func (d *Depth) Remove(key *pkg.OrderKey) {
	d.depthMutex.Lock()
	defer d.depthMutex.Unlock()
	var price_levels *redblacktree.Tree
	if key.Side == pkg.SideSell {
		price_levels = d.Asks
	} else {
		price_levels = d.Bids
	}

	pl := NewPriceLevel(key.Side, key.Price)

	value, found := price_levels.Get(pl.Key())

	if !found {
		return
	}

	price_level := value.(*PriceLevel)
	remain_quantity := price_level.Remove(key)

	if price_level.Empty() || remain_quantity.IsZero() {
		price_levels.Remove(pl.Key())
	}

	d.Notification.Publish(pl.Side, pl.Price, remain_quantity)
}

func (d *Depth) FetchOrderBook(limit int64) *GrpcEngine.FetchOrderBookResponse {
	d.depthMutex.Lock()
	defer d.depthMutex.Unlock()

	result := &GrpcEngine.FetchOrderBookResponse{
		Symbol:   &GrpcSymbol.Symbol{BaseCurrency: d.Symbol.BaseCurrency, QuoteCurrency: d.Symbol.QuoteCurrency},
		Asks:     make([]*GrpcEngine.BookOrder, 0),
		Bids:     make([]*GrpcEngine.BookOrder, 0),
		Sequence: d.Notification.Sequence,
	}

	ait := d.Asks.Iterator()
	ait.End()
	var i int64
	for i = 0; ait.Prev() && i < limit; i++ {
		price_level := ait.Value().(*PriceLevel)
		price := price_level.Price
		quantity := price_level.Total()

		result.Asks = append(result.Asks, &GrpcEngine.BookOrder{
			PriceQuantity: []*GrpcUtils.Decimal{
				{
					Val: price.CoefficientInt64(),
					Exp: price.Exponent(),
				},
				{
					Val: quantity.CoefficientInt64(),
					Exp: quantity.Exponent(),
				},
			},
		})
	}

	bit := d.Bids.Iterator()
	bit.End()
	for i = 0; bit.Prev() && i < limit; i++ {
		price_level := bit.Value().(*PriceLevel)
		price := price_level.Price
		quantity := price_level.Total()

		result.Bids = append(result.Bids, &GrpcEngine.BookOrder{
			PriceQuantity: []*GrpcUtils.Decimal{
				{
					Val: price.CoefficientInt64(),
					Exp: price.Exponent(),
				},
				{
					Val: quantity.CoefficientInt64(),
					Exp: quantity.Exponent(),
				},
			},
		})
	}

	return result
}

func (d *Depth) PublishSnapshot() {
	d.SnapshotTime = time.Now()

	asks_depth := make([][]decimal.Decimal, 0)
	bids_depth := make([][]decimal.Decimal, 0)

	i := 0
	for _, r := range d.Asks.Values() {
		i++
		pl := r.(*PriceLevel)

		asks_depth = append(asks_depth, []decimal.Decimal{pl.Price, pl.Total()})
		if i >= 300 {
			break
		}
	}

	i = 0
	for _, r := range d.Bids.Values() {
		i++
		pl := r.(*PriceLevel)

		bids_depth = append(bids_depth, []decimal.Decimal{pl.Price, pl.Total()})
		if i >= 300 {
			break
		}
	}

	config.RangoClient.EnqueueEvent(pkg.EnqueueEventKindPublic, strings.ToLower(d.Symbol.ToSymbol("")), "ob-snap", pkg.DepthJSON{
		Asks:     asks_depth,
		Bids:     bids_depth,
		Sequence: d.Notification.Sequence,
	})
}

func makeComparator(a, b interface{}) int {
	aPriceLevel := a.(*PriceLevelKey)
	bPriceLevel := b.(*PriceLevelKey)

	switch {
	case aPriceLevel.Side == pkg.SideSell && aPriceLevel.Price.LessThan(bPriceLevel.Price):
		return -1

	case aPriceLevel.Side == pkg.SideSell && aPriceLevel.Price.GreaterThan(bPriceLevel.Price):
		return 1

	case aPriceLevel.Side == pkg.SideBuy && aPriceLevel.Price.LessThan(bPriceLevel.Price):
		return 1

	case aPriceLevel.Side == pkg.SideBuy && aPriceLevel.Price.GreaterThan(bPriceLevel.Price):
		return -1

	default:
		return 0
	}
}
