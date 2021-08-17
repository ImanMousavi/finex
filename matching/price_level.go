package matching

import (
	"sync"

	"github.com/emirpasic/gods/lists/arraylist"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/pkg/order"
)

type PriceLevel struct {
	sync.Mutex
	Side   order.OrderSide
	Price  decimal.Decimal
	Orders *arraylist.List
}

type PriceLevelKey struct {
	Side  order.OrderSide
	Price decimal.Decimal
}

func NewPriceLevel(side order.OrderSide, price decimal.Decimal) *PriceLevel {
	return &PriceLevel{
		Side:   side,
		Price:  price,
		Orders: arraylist.New(),
	}
}

func (p *PriceLevel) Key() *PriceLevelKey {
	return &PriceLevelKey{
		Side:  p.Side,
		Price: p.Price,
	}
}

func (p *PriceLevel) Add(o *order.Order) {
	p.Lock()
	defer p.Unlock()

	index, _ := p.Orders.Find(func(index int, value interface{}) bool {
		order := value.(*order.Order)

		return order.UUID == o.UUID
	})

	if index == -1 {
		p.Orders.Add(o)
		p.Orders.Sort(OrderComparator)
	}
}

func (p *PriceLevel) Get(key *order.OrderKey) *order.Order {
	p.Lock()
	defer p.Unlock()
	index, value := p.Orders.Find(func(index int, value interface{}) bool {
		order := value.(*order.Order)

		return order.UUID == key.UUID
	})
	if index == -1 {
		return nil
	}

	return value.(*order.Order)
}

func (p *PriceLevel) Top() *order.Order {
	value, found := p.Orders.Get(0)
	if !found {
		return nil
	}
	return value.(*order.Order)
}

func (p *PriceLevel) Empty() bool {
	return p.Orders.Empty()
}

func (p *PriceLevel) Size() int {
	return p.Orders.Size()
}

func (p *PriceLevel) Total() decimal.Decimal {
	p.Lock()
	defer p.Unlock()
	total := decimal.Zero
	p.Orders.Each(func(index int, value interface{}) {
		order := value.(*order.Order)

		total = total.Add(order.UnfilledQuantity())
	})
	return total
}

func (p *PriceLevel) Remove(key *order.OrderKey) {
	p.Lock()
	defer p.Unlock()
	index, _ := p.Orders.Find(func(index int, value interface{}) bool {
		order := value.(*order.Order)

		return order.UUID == key.UUID
	})

	if index >= 0 {
		p.Orders.Remove(index)
	}
}

func OrderComparator(a, b interface{}) int {
	aKey := a.(*order.Order)
	bKey := b.(*order.Order)

	if aKey.CreatedAt.Before(bKey.CreatedAt) {
		return 1
	} else if aKey.CreatedAt.After(bKey.CreatedAt) {
		return -1
	}

	return 0
}
