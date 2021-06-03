package matching

import (
	"log"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/ericlagergren/decimal"
)

// PriceLevel .
type PriceLevel struct {
	Price    decimal.Big
	Quantity decimal.Big
	Side     OrderSide
	Count    uint64
}

// PriceLevelKey .
type PriceLevelKey struct {
	Price decimal.Big
	Side  OrderSide
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
	Symbol string
	Bids   *rbt.Tree
	Asks   *rbt.Tree
}

// NewDepth returns a depth with specific scale.
func NewDepth(symbol string) *Depth {
	return &Depth{
		Symbol: symbol,
		Bids:   rbt.NewWith(PriceLevelComparator),
		Asks:   rbt.NewWith(PriceLevelComparator),
	}
}

// UpdatePriceLevel updates depth with price level.
func (d *Depth) UpdatePriceLevel(side OrderSide, price decimal.Big, quantity decimal.Big, addLessCount bool) {
	var priceLevels *rbt.Tree
	pl := &PriceLevel{
		Price:    price,
		Side:     side,
		Quantity: quantity,
	}

	switch side {
	case SideSell:
		priceLevels = d.Asks

	case SideBuy:
		priceLevels = d.Bids

	default:
		log.Fatalf("[depth] invalid price level side %s", side)
	}

	foundPriceLevel, found := priceLevels.Get(pl.Key())
	if !found {
		priceLevels.Put(pl.Key(), pl)
		return
	}

	existedPriceLevel := foundPriceLevel.(*PriceLevel)
	BaseContext.Add(&existedPriceLevel.Quantity, &existedPriceLevel.Quantity, &pl.Quantity)
	if addLessCount {
		existedPriceLevel.Count++
	} else {
		existedPriceLevel.Count--
	}

	if existedPriceLevel.Count == 0 || existedPriceLevel.Quantity.Sign() == 0 {
		priceLevels.Remove(existedPriceLevel.Key())
	}
}

// PriceLevelComparator .
func PriceLevelComparator(a, b interface{}) int {
	this := a.(*PriceLevelKey)
	that := b.(*PriceLevelKey)

	switch {
	case this.Side == SideSell && this.Price.Cmp(&that.Price) < 0:
		return 1

	case this.Side == SideSell && this.Price.Cmp(&that.Price) > 0:
		return -1

	case this.Side == SideBuy && this.Price.Cmp(&that.Price) < 0:
		return -1

	case this.Side == SideBuy && this.Price.Cmp(&that.Price) > 0:
		return 1

	default:
	}

	return 0
}
