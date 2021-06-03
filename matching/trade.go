package matching

import (
	"time"

	"github.com/ericlagergren/decimal"
)

// Trade represents two opposed matched orders.
type Trade struct {
	ID           uint64
	Instrument   string
	MakerOrderID uint64
	TakerOrderID uint64
	MakerID      uint64
	TakerID      uint64
	Price        decimal.Big
	Qty          decimal.Big
	Total        decimal.Big
	Timestamp    time.Time
}

func NewTrade() {

}
