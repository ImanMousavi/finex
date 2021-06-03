package matching

import (
	"time"
)

type Book struct {
	Asks [][]float64
	Bids [][]float64
}

type Notification struct {
	Instrument string // instrument name
	Sequence   uint64
	Book       *Book
}

func NewNotification(instrument string) *Notification {
	return &Notification{
		Instrument: instrument,
		Sequence:   0,
		Book: &Book{
			Asks: [][]float64{},
			Bids: [][]float64{},
		},
	}
}

func (n Notification) Start() {
	go n.StartLoop()
}

func (n Notification) StartLoop() {
	for {
		time.Sleep(100 * time.Millisecond)

		if len(n.Book.Asks) == 0 && len(n.Book.Bids) == 0 {
			continue
		}

		// n.Sequence = n.Sequence + 1

		// payload := types.Depth{
		// 	Asks:     n.Book.Asks,
		// 	Bids:     n.Book.Bids,
		// 	Sequence: n.Sequence,
		// }

		// payload_message, _ := json.Marshal(payload)

		// mq_client.EnqueueEvent("public", n.Instrument, "depth", payload_message)

		n.Book.Asks = [][]float64{}
		n.Book.Bids = [][]float64{}
	}
}

func (n Notification) Publish(side OrderSide, price float64, amount float64) {
	if side {
		n.Book.Bids = append(n.Book.Bids, []float64{price, amount})
	} else {
		n.Book.Asks = append(n.Book.Bids, []float64{price, amount})
	}
}
