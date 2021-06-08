package matching

import (
	"math/rand"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type OrderQueueTestSuite struct {
	suite.Suite
}

func (s *OrderQueueTestSuite) TestOrderQueue() {
	orderQueue := NewOrderQueue(5)

	s.Nil(orderQueue.First())
	s.Nil(orderQueue.Pop())

	for i := uint64(0); i < 10; i++ {
		orderQueue.Push(&Order{
			ID: i,
		})

		s.Equal(int64(i+1), orderQueue.Size())
	}

	for i := uint64(0); i < 10; i++ {
		s.Equal(&Order{ID: i}, orderQueue.First())
		s.Equal(&Order{ID: i}, orderQueue.Pop())
		s.Equal(int64(9-i), orderQueue.Size())
	}
}

func TestOrderQueue(t *testing.T) {
	suite.Run(t, new(OrderQueueTestSuite))
}

func BenchmarkAddQueue(b *testing.B) {
	orderQueue := NewOrderQueue(0)

	orders := make([]*Order, b.N)
	for n := 0; n < b.N; n++ {
		var side Side
		switch rand.Intn(2) {
		case 0:
			side = SideSell
		case 1:
			side = SideBuy
		}

		price := rand.Intn(10)
		quantity := rand.Intn(10) + 1

		orders[n] = &Order{
			ID:       uint64(n),
			Side:     side,
			Price:    decimal.NullDecimal{Decimal: decimal.NewFromFloat(float64(price)), Valid: true},
			Quantity: decimal.NewFromFloat(float64(quantity)),
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		orderQueue.Push(orders[n])
	}
}
