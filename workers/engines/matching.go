package engines

import (
	"encoding/json"
	"log"

	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/matching"
	"github.com/zsmartex/go-finex/models"
)

type MatchingPayloadMessage struct {
	Action matching.PayloadAction `json:"action"`
	Order  *matching.Order        `json:"order"`
	Market string                 `json:"market"`
}

type MatchingWorker struct {
	Engines map[string]*matching.Engine
}

func NewMatchingWorker() *MatchingWorker {
	worker := &MatchingWorker{
		Engines: make(map[string]*matching.Engine),
	}

	worker.Reload("all")

	return worker
}

func (w MatchingWorker) Process(payload []byte) error {
	var matching_payload MatchingPayloadMessage
	err := json.Unmarshal(payload, &matching_payload)
	if err != nil {
		log.Print(err)
	}

	switch matching_payload.Action {
	case matching.ActionSubmit:
		order := matching_payload.Order
		return w.SubmitOrder(order)
	case matching.ActionCancel:
		order := matching_payload.Order
		return w.CancelOrder(order)
	case matching.ActionReload:
		w.Reload(matching_payload.Market)
	default:
		log.Fatalf("Unknown action: %s", matching_payload.Action)
	}

	return nil
}

func (w MatchingWorker) SubmitOrder(order *matching.Order) error {
	return w.Engines[order.Symbol].Submit(order)
}

func (w MatchingWorker) CancelOrder(order *matching.Order) error {
	return w.Engines[order.Symbol].Cancel(order)
}

func (w MatchingWorker) AddNewEngine(market string) *matching.Engine {
	w.Engines[market] = matching.NewEngine(market)
	return w.Engines[market]
}

func (w MatchingWorker) GetEngineByMarket(market string) *matching.Engine {
	engine, found := w.Engines[market]

	if found {
		return engine
	}

	return nil
}

func (w MatchingWorker) Reload(market string) {
	if market == "all" {
		var markets []models.Market
		config.DataBase.Where("state = ?", "enabled").Find(&markets)
		for _, market := range markets {
			w.InitializeEngine(market.ID)
		}
		log.Println("All engines reloaded.")
	} else {
		w.InitializeEngine(market)
	}
}

func (w MatchingWorker) InitializeEngine(market string) {
	w.AddNewEngine(market)
	w.LoadOrders(market)
	log.Printf("%v engine reloaded.\n", market)
}

func (w MatchingWorker) BuildOrder(order map[string]interface{}) *matching.Order {
	mapOrderInterfaceJSON, _ := json.Marshal(order)

	var mOrder *matching.Order
	json.Unmarshal(mapOrderInterfaceJSON, &mOrder)

	return mOrder
}

func (w MatchingWorker) LoadOrders(market string) {
	var orders []models.Order
	config.DataBase.Where("market_id = ? AND state = ?", market, models.StateWait).Order("id asc").Find(&orders)

	for _, order := range orders {
		mOrder := w.BuildOrder(order.ToMatchingAttributes())
		w.SubmitOrder(mOrder)
	}
}
