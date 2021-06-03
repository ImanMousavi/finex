package depth_service

import (
	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/models"
	"github.com/zsmartex/go-finex/types"
)

type Depth struct {
	Asks     map[float64]float64
	Bids     map[float64]float64
	Sequence uint64
}

type DepthService struct {
	Market   string
	Asks     map[float64]float64
	Bids     map[float64]float64
	Sequence uint64
}

func NewDepthService(market string) *DepthService {
	return &DepthService{
		Market:   market,
		Asks:     make(map[float64]float64),
		Bids:     make(map[float64]float64),
		Sequence: 0,
	}
}

func Fetch(market string) *DepthService {
	depthService := NewDepthService(market)

	var sequence uint64

	config.Redis.GetKey("finex:"+market+":depth:sequence", &sequence)

	asks_depth := make(map[float64]float64)
	bids_depth := make(map[float64]float64)
	for _, ord := range models.GetDepth(models.SideSell, market) {
		asks_depth[ord[0]] = ord[1]
	}
	for _, ord := range models.GetDepth(models.SideBuy, market) {
		bids_depth[ord[0]] = ord[1]
	}

	depthService.Asks = asks_depth
	depthService.Bids = bids_depth
	depthService.Sequence = sequence

	return depthService
}

func (d *DepthService) ToJSON() types.Depth {
	var ask_depth [][]float64
	var bid_depth [][]float64

	for price, amount := range d.Asks {
		ask_depth = append(ask_depth, []float64{price, amount})
	}
	for price, amount := range d.Bids {
		bid_depth = append(bid_depth, []float64{price, amount})
	}

	return types.Depth{
		Asks:     ask_depth,
		Bids:     bid_depth,
		Sequence: d.Sequence,
	}
}
