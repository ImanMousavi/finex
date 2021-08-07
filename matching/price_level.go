package matching

import (
	"sort"
	"sync"

	"github.com/shopspring/decimal"
)

type onChange func(side OrderSide, price decimal.Decimal, amount decimal.Decimal)

type PriceLevel struct {
	sync.Mutex
	Side     OrderSide
	Price    decimal.Decimal
	Orders   []*Order
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
		Orders:   make([]*Order, 0),
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
	for _, o := range p.Orders {
		if o.ID == order.ID {
			return
		}
	}

	p.Orders = append(p.Orders, order)
	sort.Slice(p.Orders, func(i, j int) bool {
		return p.Orders[i].ID < p.Orders[j].ID
	})

	p.onChange(p.Side, p.Price, p.Total())
}

func (p *PriceLevel) Top() *Order {
	if p.Empty() {
		return nil
	}

	return p.Orders[0]
}

func (p *PriceLevel) Empty() bool {
	return len(p.Orders) == 0
}

func (p *PriceLevel) Size() int {
	return len(p.Orders)
}

func (p *PriceLevel) Total() decimal.Decimal {
	total := decimal.Zero

	for _, order := range p.Orders {
		total = total.Add(order.UnfilledQuantity())
	}

	return total
}

func (p *PriceLevel) Remove(order *Order) {
	p.Lock()
	defer p.Unlock()
	for index, o := range p.Orders {
		if o.ID == order.ID {
			p.Orders = append(p.Orders[:index], p.Orders[index+1:]...)
		}
	}

	p.onChange(p.Side, p.Price, p.Total())
}
