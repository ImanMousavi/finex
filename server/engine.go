package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/emirpasic/gods/trees/redblacktree"
	GrpcEngine "github.com/zsmartex/pkg/Grpc/engine"
	GrpcOrder "github.com/zsmartex/pkg/Grpc/order"
	GrpcUtils "github.com/zsmartex/pkg/Grpc/utils"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/matching"
	"github.com/zsmartex/finex/models"
	"github.com/zsmartex/pkg"
	"github.com/zsmartex/pkg/order"
)

type EngineServer struct {
	Engines map[string]*matching.Engine
}

func NewEngineServer() *EngineServer {
	worker := &EngineServer{
		Engines: make(map[string]*matching.Engine),
	}

	worker.Reload("all")

	return worker
}

func (w *EngineServer) Process(payload []byte) error {
	var matching_payload pkg.MatchingPayloadMessage
	if err := json.Unmarshal(payload, &matching_payload); err != nil {
		return err
	}

	switch matching_payload.Action {
	case pkg.ActionSubmit:
		order := matching_payload.Order
		return w.SubmitOrder(order)
	case pkg.ActionCancel:
		order := matching_payload.Order
		return w.CancelOrder(order)
	case pkg.ActionCancelWithKey:
		key := matching_payload.Key
		return w.CancelOrderWithKey(key)
	case pkg.ActionNew:
		w.InitializeEngine(matching_payload.Market)
	case pkg.ActionReload:
		w.Reload(matching_payload.Market)
	default:
		config.Logger.Fatalf("Unknown action: %s", matching_payload.Action)
	}

	return nil
}

func (s *EngineServer) SubmitOrder(order *order.Order) error {
	engine := s.Engines[order.Symbol]

	if engine == nil {
		return errors.New("engine not found")
	}

	if !engine.Initialized {
		return errors.New("engine is not ready")
	}

	if order.Price.IsNegative() || order.StopPrice.IsNegative() {
		return errors.New("price is negative")
	}

	engine.Submit(order)
	return nil
}

func (s *EngineServer) CancelOrderWithKey(key *order.OrderKey) error {
	engine := s.Engines[key.Symbol]

	if engine == nil {
		return errors.New("engine not found")
	}

	if !engine.Initialized {
		return errors.New("engine is not ready")
	}

	engine.CancelWithKey(key)
	return nil
}

func (s *EngineServer) CancelOrder(order *order.Order) error {
	engine := s.Engines[order.Symbol]

	if engine == nil {
		return errors.New("engine not found")
	}

	if !engine.Initialized {
		return errors.New("engine is not ready")
	}

	engine.Cancel(order)
	return nil
}

func (s EngineServer) GetEngineByMarket(market string) *matching.Engine {
	engine, found := s.Engines[market]

	if found {
		return engine
	}

	return nil
}

func (s *EngineServer) FetchOrder(ctx context.Context, req *GrpcEngine.FetchOrderRequest) (*GrpcEngine.FetchOrderResponse, error) {
	key := req.OrderKey.ToOrderKey()
	engine := s.GetEngineByMarket(key.Symbol)
	pl := matching.NewPriceLevel(key.Side, key.Price)

	var price_levels *redblacktree.Tree
	if key.Side == order.SideSell {
		price_levels = engine.OrderBook.Depth.Asks
	} else {
		price_levels = engine.OrderBook.Depth.Bids
	}

	value, found := price_levels.Get(pl.Key())
	if !found {
		return nil, errors.New(fmt.Sprint("Can't find order with uuid: %s in orderbook", key.UUID.String()))
	}

	price_level := value.(*matching.PriceLevel)
	order := price_level.Get(key)

	return &GrpcEngine.FetchOrderResponse{
		Order: &GrpcOrder.Order{
			Id:       order.ID,
			Uuid:     order.UUID[:],
			MemberId: order.MemberID,
			Symbol:   order.Symbol,
			Side:     string(order.Side),
			Type:     string(order.Type),
			Price: &GrpcUtils.Decimal{
				Val: order.Price.CoefficientInt64(),
				Exp: order.Price.Exponent(),
			},
			StopPrice: &GrpcUtils.Decimal{
				Val: order.StopPrice.CoefficientInt64(),
				Exp: order.StopPrice.Exponent(),
			},
			Quantity: &GrpcUtils.Decimal{
				Val: order.Quantity.CoefficientInt64(),
				Exp: order.Quantity.Exponent(),
			},
			FilledQuantity: &GrpcUtils.Decimal{
				Val: order.FilledQuantity.CoefficientInt64(),
				Exp: order.FilledQuantity.Exponent(),
			},
			Fake:      order.Fake,
			Cancelled: order.Cancelled,
			CreatedAt: timestamppb.New(order.CreatedAt),
		},
	}, nil
}

func (s *EngineServer) FetchMarketPrice(ctx context.Context, req *GrpcEngine.FetchMarketPriceRequest) (*GrpcEngine.FetchMarketPriceResponse, error) {
	engine := s.GetEngineByMarket(req.Symbol)
	price := engine.OrderBook.MarketPrice

	return &GrpcEngine.FetchMarketPriceResponse{
		Price: &GrpcUtils.Decimal{
			Val: price.CoefficientInt64(),
			Exp: price.Exponent(),
		},
	}, nil
}

func (s *EngineServer) FetchOrderBook(ctx context.Context, req *GrpcEngine.FetchOrderBookRequest) (*GrpcEngine.FetchOrderBookResponse, error) {
	engine := s.GetEngineByMarket(req.Symbol)
	response := engine.OrderBook.Depth.FetchOrderBook(req.Limit)

	return response, nil
}

func (s *EngineServer) CalcMarketOrder(ctx context.Context, req *GrpcEngine.CalcMarketOrderRequest) (*GrpcEngine.CalcMarketOrderResponse, error) {
	engine := s.GetEngineByMarket(req.Symbol)
	response := engine.OrderBook.CalcMarketOrder(order.OrderSide(req.Side), req.Quantity.ToNullDecimal(), req.Volume.ToNullDecimal())

	return response, nil
}

func (s *EngineServer) Reload(market string) {
	if market == "all" {
		var markets []models.Market
		config.DataBase.Where("state = ?", "enabled").Find(&markets)
		for _, market := range markets {
			s.InitializeEngine(market.Symbol)
		}
		config.Logger.Info("All engines reloaded.")
	} else {
		s.InitializeEngine(market)
	}
}

func (s *EngineServer) InitializeEngine(market string) {
	lastPrice := decimal.Zero
	trade := models.GetLastTradeFromInflux(market)
	if trade != nil {
		lastPrice = trade.Price
	}

	engine := matching.NewEngine(market, lastPrice)
	s.Engines[market] = engine
	s.LoadOrders(engine)
	engine.Initialized = true
	config.Logger.Infof("%v engine reloaded.", market)
}

func (s *EngineServer) LoadOrders(engine *matching.Engine) {
	var orders []models.Order
	config.DataBase.Where("market_id = ? AND state = ?", engine.Market, models.StateWait).Order("id asc").Find(&orders)
	for _, order := range orders {
		engine.Submit(order.ToMatchingAttributes())
	}
}
