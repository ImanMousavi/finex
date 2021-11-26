package matching

import (
	"math/rand"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/pkg/order"
)

func BenchmarkInsertOrder(b *testing.B) {
	orderBook := NewOrderBook("market", decimal.Zero)

	orders := make([]*order.Order, b.N)
	for n := 0; n < b.N; n++ {
		var side order.OrderSide
		switch rand.Intn(2) {
		case 0:
			side = order.SideSell
		case 1:
			side = order.SideBuy
		}

		price := rand.Intn(10)
		quantity := rand.Intn(10) + 1

		orders[n] = &order.Order{
			ID:        int64(n),
			Side:      side,
			Price:     decimal.NewFromFloat(float64(price)),
			Quantity:  decimal.NewFromFloat(float64(quantity)),
			CreatedAt: time.Now(),
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		orderBook.Add(orders[n])
	}
	b.StopTimer()
}
