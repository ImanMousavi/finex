package depth_service

import (
	"github.com/shopspring/decimal"
	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/models"
	"github.com/zsmartex/go-finex/types"
)

type DepthService = types.Depth

func NewDepthService(market string) *DepthService {
	return &DepthService{
		Asks:     make([][]decimal.Decimal, 0),
		Bids:     make([][]decimal.Decimal, 0),
		Sequence: 0,
	}
}

func Fetch(market string) *DepthService {
	depthService := NewDepthService(market)

	var sequence uint64

	config.Redis.GetKey("finex:"+market+":depth:sequence", &sequence)

	asks_depth := make([][]decimal.Decimal, 0)
	bids_depth := make([][]decimal.Decimal, 0)
	for _, ord := range models.GetDepth(models.SideSell, market) {
		asks_depth = append(asks_depth, []decimal.Decimal{ord[0], ord[1]})
	}
	for _, ord := range models.GetDepth(models.SideBuy, market) {
		bids_depth = append(bids_depth, []decimal.Decimal{ord[0], ord[1]})
	}

	depthService.Asks = asks_depth
	depthService.Bids = bids_depth
	depthService.Sequence = sequence

	return depthService
}
