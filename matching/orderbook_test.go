package matching

import (
	"math/rand"
	"os"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"io/ioutil"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"github.com/zsmartex/go-finex/config"
)

type suiteOrderBookTester struct {
	suite.Suite
}

type OrderBookEntry struct {
	Name   string   `yaml:"name"`
	Orders []string `yaml:"orders"`
	Trades []string `yaml:"trades"`
}

func (ode *OrderBookEntry) Test(s *suiteOrderBookTester) {
	s.T().Run(ode.Name, func(t *testing.T) {
		orderBook := NewOrderBook("market")

		var trades []*Trade
		for _, o := range ode.Orders {
			rawResult := strings.Split(o, ",")
			var result []string
			for _, r := range rawResult {
				result = append(result, strings.TrimSpace(r))
			}

			var side Side
			switch result[1] {
			case "ASK":
				side = SideSell
			case "BID":
				side = SideBuy
			}
			id, _ := strconv.Atoi(result[0])
			price, _ := decimal.NewFromString(result[2])
			quantity, _ := decimal.NewFromString(result[3])
			stopPrice := decimal.Zero

			if len(result) >= 5 {
				val, _ := decimal.NewFromString(result[4])
				stopPrice = val
			}

			newOrder := &Order{
				ID:        uint64(id),
				Side:      side,
				Price:     price,
				Quantity:  quantity,
				StopPrice: stopPrice,
			}

			newTrades := orderBook.InsertOrder(newOrder)
			if len(newTrades) > 0 {
				trades = append(trades, newTrades...)
			}
		}

		var expectedTrades []*Trade
		for _, t := range ode.Trades {
			rawResult := strings.Split(t, ",")
			var result []string
			for _, r := range rawResult {
				result = append(result, strings.TrimSpace(r))
			}

			price, _ := decimal.NewFromString(result[0])
			quantity, _ := decimal.NewFromString(result[1])
			total := price.Mul(quantity)
			makeID, _ := strconv.Atoi(result[2])
			takerID, _ := strconv.Atoi(result[3])
			expectedTrades = append(expectedTrades, &Trade{
				Price:        price,
				Quantity:     quantity,
				Total:        total,
				MakerOrderID: uint64(makeID),
				TakerOrderID: uint64(takerID),
			})
		}

		s.EqualValues(expectedTrades, trades)
	})
}

func (s *suiteOrderBookTester) TestInsertOrder() {
	os.Setenv("LOG_LEVEL", "DEBUG")
	config.NewLoggerService()

	orderbookFile, err := ioutil.ReadFile("./fixtures/orderbook.yaml")

	s.NoError(err)

	var entries []OrderBookEntry
	err = yaml.Unmarshal(orderbookFile, &entries)
	if err != nil {
		panic(err)
	}

	for _, entry := range entries {
		entry.Test(s)
	}
}

func (s *suiteOrderBookTester) TestInsertLimitOrder() {
	orderBook := NewOrderBook("market")

	limitOrder := &Order{
		ID:       2,
		Side:     SideBuy,
		Price:    decimal.NewFromFloat(10.0),
		Quantity: decimal.NewFromFloat(30.0),
	}

	s.EqualValues([]*Trade{}, orderBook.InsertOrder(limitOrder))
	s.EqualValues(limitOrder, orderBook.Bids.Right().Value.(*Order))
	s.EqualValues(1, orderBook.Bids.Size())
}

func (s *suiteOrderBookTester) TestInsertImmediateOrCancelOrder() {
	orderBook := NewOrderBook("market")

	iocOrder := &Order{
		ID:                2,
		Side:              SideBuy,
		Price:             decimal.NewFromFloat(10.0),
		Quantity:          decimal.NewFromFloat(30.0),
		ImmediateOrCancel: true,
	}

	s.EqualValues([]*Trade{}, orderBook.InsertOrder(iocOrder))
	s.True(orderBook.Bids.Empty())
	s.True(orderBook.Asks.Empty())
}

func (s *suiteOrderBookTester) TestCancelOrder() {
	orderBook := NewOrderBook("market")

	bidOrder := &Order{
		ID:       1,
		Side:     SideBuy,
		Price:    decimal.NewFromFloat(10.0),
		Quantity: decimal.NewFromFloat(30.0),
	}

	askOrder := &Order{
		ID:       2,
		Side:     SideSell,
		Price:    decimal.NewFromFloat(10.0),
		Quantity: decimal.NewFromFloat(30.0),
	}

	orderBook.InsertOrder(bidOrder)
	orderBook.InsertOrder(askOrder)

	orderBook.CancelOrder(bidOrder)
	s.Nil(orderBook.Bids.Right())
	s.EqualValues(0, orderBook.Bids.Size())

	orderBook.CancelOrder(askOrder)
	s.Nil(orderBook.Asks.Right())
	s.EqualValues(0, orderBook.Asks.Size())

	orderBook.InsertOrder(bidOrder)
	orderBook.CancelOrder(&Order{
		ID: 1,
	})
	s.Nil(orderBook.Bids.Right())
	s.EqualValues(0, orderBook.Bids.Size())
}

func TestOrderBook(t *testing.T) {
	tester := new(suiteOrderBookTester)
	suite.Run(t, tester)
}

func BenchmarkInsertOrder(b *testing.B) {
	orderBook := NewOrderBook("market")

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
			Price:    decimal.NewFromFloat(float64(price)),
			Quantity: decimal.NewFromFloat(float64(quantity)),
		}
	}

	for n := 0; n < b.N; n++ {
		orderBook.InsertOrder(orders[n])
	}
}
