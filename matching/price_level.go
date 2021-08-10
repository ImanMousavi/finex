package matching

import (
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"github.com/shopspring/decimal"
)

type onChange func(side OrderSide, price decimal.Decimal, amount decimal.Decimal)

type PriceLevel struct {
	sync.Mutex
	Side     OrderSide
	Price    decimal.Decimal
	Orders   *redblacktree.Tree
	onChange onChange
}

type PriceLevelKey struct {
	Side  OrderSide
	Price decimal.Decimal
}

func NewPriceLevel(side OrderSide, price decimal.Decimal, onChange onChange) *PriceLevel {
	return &PriceLevel{
		Side:     side,
		Price:    price,
		Orders:   redblacktree.NewWith(OrderComparator),
		onChange: onChange,
	}
}

func (p *PriceLevel) Key() *PriceLevelKey {
	return &PriceLevelKey{
		Side:  p.Side,
		Price: p.Price,
	}
}

func (p *PriceLevel) Add(order *Order) {
	p.Lock()
	defer p.Unlock()
	p.Orders.Put(order.Key(), order)

	p.onChange(p.Side, p.Price, p.Total())
}

func (p *PriceLevel) Top() *Order {
	if p.Empty() {
		return nil
	}

	return p.Orders.Right().Value.(*Order)
}

func (p *PriceLevel) Empty() bool {
	return p.Orders.Empty()
}

func (p *PriceLevel) Size() int {
	return p.Orders.Size()
}

func (p *PriceLevel) Total() decimal.Decimal {
	total := decimal.Zero
	it := p.Orders.Iterator()
	for it.Next() {
		order := it.Value().(*Order)

		total = total.Add(order.UnfilledQuantity())
	}

	return total
}

func (p *PriceLevel) Remove(order *Order) {
	p.Lock()
	defer p.Unlock()
	p.Orders.Remove(order.Key())

	p.onChange(p.Side, p.Price, p.Total())
}

func OrderComparator(a, b interface{}) int {
	aKey := a.(*OrderKey)
	bKey := b.(*OrderKey)

	if aKey.ID == bKey.ID {
		return 0
	}

	// based on ask
	switch {
	case aKey.Side == SideSell && aKey.Price.LessThan(bKey.Price):
		return 1

	case aKey.Side == SideSell && aKey.Price.GreaterThan(bKey.Price):
		return -1

	case aKey.Side == SideBuy && aKey.Price.LessThan(bKey.Price):
		return -1

	case aKey.Side == SideBuy && aKey.Price.GreaterThan(bKey.Price):
		return 1

	default:
		switch {
		case aKey.CreatedAt.Before(bKey.CreatedAt):
			return 1

		case aKey.CreatedAt.After(bKey.CreatedAt):
			return -1

		default:
			return utils.UInt64Comparator(aKey.ID, bKey.ID) * -1
		}
	}
}
