package workers

import (
	"encoding/json"
	"log"

	"gitlab.com/zsmartex/finex/config"
	"gitlab.com/zsmartex/finex/models"
	"gitlab.com/zsmartex/finex/services/depth_service"
	"gitlab.com/zsmartex/finex/types"
)

type DepthWorker struct {
	Depths map[string]*depth_service.DepthService
}

type DepthCachePayloadMessage struct {
	Market string      `json:"market"`
	Depth  types.Depth `json:"depth"`
}

func NewDeptCachehWorker() *DepthWorker {
	return &DepthWorker{
		Depths: make(map[string]*depth_service.DepthService),
	}
}

func (w *DepthWorker) Process(payload []byte) {
	var depth_m DepthCachePayloadMessage
	err := json.Unmarshal(payload, &depth_m)
	if err != nil {
		log.Print(err)
	}
	depth_payload := depth_m.Depth
	depth := w.Depths[depth_m.Market]

	for _, ord := range depth_payload.Asks {
		price := ord[0]
		amount := ord[1]

		delete(depth.Asks, price)
		if amount > 0 {
			depth.Asks[price] = amount
		}
	}

	for _, ord := range depth_payload.Bids {
		price := ord[0]
		amount := ord[1]

		delete(depth.Asks, price)
		if amount > 0 {
			depth.Bids[price] = amount
		}
	}
	depth.Sequence++

	depth_j := depth.ToJSON()

	config.Redis.SetKey("finex:"+depth_m.Market+":depth:asks", depth_j.Asks, -1)
	config.Redis.SetKey("finex:"+depth_m.Market+":depth:asks", depth_j.Bids, -1)
	config.Redis.SetKey("finex:"+depth_m.Market+":depth:sequence", depth_j.Sequence, -1)
}

func (w *DepthWorker) Reload(market string) {
	switch market {
	case "all":
		var Markets []models.Market

		config.DataBase.Where("state = ?", "enabled").Find(&Markets)
		for _, Market := range Markets {
			w.AddNewDepth(Market.ID)
		}

		log.Printf("All depths reloaded.\n")
	default:
		w.AddNewDepth(market)
		log.Printf("%s depth reloaded.\n", market)
	}
}

func (w *DepthWorker) AddNewDepth(market string) {
	log.Printf("initializing %s\n", market)
	w.Depths[market] = depth_service.Fetch(market)
}
