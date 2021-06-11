package matching

import (
	"sync"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

// PriceLevel .
type PriceLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
	Side     Side
	Count    int32
}

// PriceLevelKey .
type PriceLevelKey struct {
	Price decimal.Decimal
	Side  Side
}

// Key returns a key for PriceLevel.
func (pl *PriceLevel) Key() *PriceLevelKey {
	return &PriceLevelKey{
		Price: pl.Price,
		Side:  pl.Side,
	}
}

// Depth .
type Depth struct {
	Symbol       string
	Bids         *rbt.Tree
	Asks         *rbt.Tree
	Sequence     uint64
	Notification *Notification
	depthMutex   sync.RWMutex
}

// NewDepth returns a depth with specific scale.
func NewDepth(symbol string, notification *Notification) *Depth {
	return &Depth{
		Symbol:       symbol,
		Bids:         rbt.NewWith(PriceLevelComparator),
		Asks:         rbt.NewWith(PriceLevelComparator),
		Notification: notification,
	}
}

// UpdatePriceLevel updates depth with price level.
func (d *Depth) UpdatePriceLevel(side Side, price, quantity decimal.Decimal, count int32) {
	d.depthMutex.Lock()
	defer d.depthMutex.Unlock()

	var priceLevels *rbt.Tree
	pl := &PriceLevel{
		Price:    price,
		Side:     side,
		Quantity: quantity,
		Count:    count,
	}

	switch pl.Side {
	case SideSell:
		priceLevels = d.Asks

	case SideBuy:
		priceLevels = d.Bids

	default:
		log.Fatalf("[depth] invalid price level side %s", pl.Side)
	}

	foundPriceLevel, found := priceLevels.Get(pl.Key())
	if !found {
		priceLevels.Put(pl.Key(), pl)
		d.NotificationPublish(side, price, pl.Quantity)
		return
	}

	existedPriceLevel := foundPriceLevel.(*PriceLevel)
	existedPriceLevel.Quantity = existedPriceLevel.Quantity.Add(pl.Quantity)
	existedPriceLevel.Count += pl.Count

	if existedPriceLevel.Count == 0 || existedPriceLevel.Quantity.Equal(decimal.Zero) {
		priceLevels.Remove(existedPriceLevel.Key())
	}
	d.NotificationPublish(side, price, existedPriceLevel.Quantity)
}

func (d *Depth) NotificationPublish(side Side, price, quantity decimal.Decimal) {
	d.Notification.Publish(side, price, quantity)
}

// PriceLevelComparator .
func PriceLevelComparator(a, b interface{}) int {
	this := a.(*PriceLevelKey)
	that := b.(*PriceLevelKey)

	switch {
	case this.Side == SideSell && this.Price.LessThan(that.Price):
		return 1

	case this.Side == SideSell && this.Price.GreaterThan(that.Price):
		return -1

	case this.Side == SideBuy && this.Price.LessThan(that.Price):
		return -1

	case this.Side == SideBuy && this.Price.GreaterThan(that.Price):
		return 1

	default:
	}

	return 0
}
