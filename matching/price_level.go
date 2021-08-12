package matching

import (
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/pkg/order"
)

type onChange func(side order.OrderSide, price decimal.Decimal, amount decimal.Decimal)

type PriceLevel struct {
	sync.Mutex
	Side   order.OrderSide
	Price  decimal.Decimal
	Orders *redblacktree.Tree
}

type PriceLevelKey struct {
	Side  order.OrderSide
	Price decimal.Decimal
}

func NewPriceLevel(side order.OrderSide, price decimal.Decimal) *PriceLevel {
	return &PriceLevel{
		Side:   side,
		Price:  price,
		Orders: redblacktree.NewWith(OrderComparator),
	}
}

func (p *PriceLevel) Key() *PriceLevelKey {
	return &PriceLevelKey{
		Side:  p.Side,
		Price: p.Price,
	}
}

func (p *PriceLevel) Add(order *order.Order) {
	p.Lock()
	defer p.Unlock()
	p.Orders.Put(order.Key(), order)
}

func (p *PriceLevel) Get(key *order.OrderKey) *order.Order {
	p.Lock()
	defer p.Unlock()
	value, found := p.Orders.Get(key)
	if !found {
		return nil
	}

	return value.(*order.Order)
}

func (p *PriceLevel) Top() *order.Order {
	if p.Empty() {
		return nil
	}

	return p.Orders.Right().Value.(*order.Order)
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
		order := it.Value().(*order.Order)

		total = total.Add(order.UnfilledQuantity())
	}

	return total
}

func (p *PriceLevel) Remove(key *order.OrderKey) {
	p.Lock()
	defer p.Unlock()
	p.Orders.Remove(key)
}

func OrderComparator(a, b interface{}) int {
	aKey := a.(*order.OrderKey)
	bKey := b.(*order.OrderKey)

	if aKey.UUID == bKey.UUID {
		return 0
	}

	if aKey.CreatedAt.Before(bKey.CreatedAt) {
		return 1
	} else if aKey.CreatedAt.After(bKey.CreatedAt) {
		return -1
	}

	return 0
}
