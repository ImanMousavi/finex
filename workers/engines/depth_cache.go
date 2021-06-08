package engines

import (
	"encoding/json"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/models"
	"github.com/zsmartex/go-finex/services/depth_service"
	"github.com/zsmartex/go-finex/types"
)

type DepthWorker struct {
	Depths map[string]*depth_service.DepthService
}

type DepthCachePayloadMessage struct {
	Market string      `json:"market"`
	Depth  types.Depth `json:"depth"`
}

func NewDeptCachehWorker() *DepthWorker {
	depth_worker := &DepthWorker{
		Depths: make(map[string]*depth_service.DepthService),
	}

	depth_worker.Reload("all")

	return depth_worker
}

func (w *DepthWorker) Process(payload []byte) {
	var depth_m DepthCachePayloadMessage
	err := json.Unmarshal(payload, &depth_m)
	if err != nil {
		log.Println(err)
	}
	depth_payload := depth_m.Depth

	depth := w.Depths[depth_m.Market]

	for _, ord := range depth_payload.Asks {
		price := ord[0]
		amount := ord[1]

		for i, o := range depth.Asks {
			if price.Equal(o[0]) {
				depth.Asks = append(depth.Asks[:i], depth.Asks[i+1:]...)
			}
		}

		if amount.IsPositive() {
			depth.Asks = append(depth.Asks, []decimal.Decimal{price, amount})
		}
	}

	for _, ord := range depth_payload.Bids {
		price := ord[0]
		amount := ord[1]

		for i, o := range depth.Bids {
			if price.Equal(o[0]) {
				depth.Bids = append(depth.Bids[:i], depth.Bids[i+1:]...)
			}
		}

		if amount.IsPositive() {
			depth.Bids = append(depth.Bids, []decimal.Decimal{price, amount})
		}
	}

	depth.Sequence++

	config.Redis.SetKey("finex:"+depth_m.Market+":depth:asks", depth.Asks, redis.KeepTTL)
	config.Redis.SetKey("finex:"+depth_m.Market+":depth:bids", depth.Bids, redis.KeepTTL)
	config.Redis.SetKey("finex:"+depth_m.Market+":depth:sequence", depth.Sequence, redis.KeepTTL)
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
