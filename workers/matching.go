package workers

import (
	"encoding/json"
	"log"

	"github.com/zsmartex/go-finex/matching"
	"github.com/zsmartex/go-finex/types"
)

type MatchingWorker struct {
	Engines map[string]*matching.Engine
}

func NewMatchingWorker() *MatchingWorker {
	return &MatchingWorker{
		Engines: make(map[string]*matching.Engine),
	}
}

func (w MatchingWorker) Process(payload []byte) {
	var matching_payload types.MatchingPayloadMessage
	err := json.Unmarshal(payload, &matching_payload)
	if err != nil {
		log.Print(err)
	}

	switch matching_payload.Action {
	case types.ActionSubmit:
		w.SubmitOrder(matching_payload.Order)
	case types.ActionCancel:
		w.CancelOrder(matching_payload.Order)
	default:
		log.Fatalf("Unknown action: %s", matching_payload.Action)
	}
}

func (w MatchingWorker) SubmitOrder(order matching.Order) {
	w.Engines[order.Instrument].Submit(order, 1)
}

func (w MatchingWorker) CancelOrder(order matching.Order) {
	w.Engines[order.Instrument].Cancel(order)
}

func (w MatchingWorker) AddNewEngine(market string) {
	w.Engines[market] = matching.NewEngine(market)
}

func (w MatchingWorker) GetEngineByMarket(market string) *matching.Engine {
	engine, found := w.Engines[market]

	if found {
		return engine
	}

	return nil
}
