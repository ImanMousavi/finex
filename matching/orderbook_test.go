package matching

import (
	"math/rand"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/pkg"
)

func BenchmarkInsertOrder(b *testing.B) {
	orderBook := NewOrderBook(pkg.Symbol{BaseCurrency: "ABC", QuoteCurrency: "XYZ"}, decimal.Zero)

	orders := make([]*pkg.Order, b.N)
	for n := 0; n < b.N; n++ {
		var side pkg.OrderSide
		switch rand.Intn(2) {
		case 0:
			side = pkg.SideSell
		case 1:
			side = pkg.SideBuy
		}

		price := rand.Intn(10)
		quantity := rand.Intn(10) + 1

		orders[n] = &pkg.Order{
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
