package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/emirpasic/gods/trees/redblacktree"
	GrpcEngine "github.com/zsmartex/pkg/Grpc/engine"
	GrpcOrder "github.com/zsmartex/pkg/Grpc/order"
	GrpcSymbol "github.com/zsmartex/pkg/Grpc/symbol"
	GrpcUtils "github.com/zsmartex/pkg/Grpc/utils"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/matching"
	"github.com/zsmartex/finex/models"
	"github.com/zsmartex/pkg"
)

type EngineServer struct {
	Engines map[pkg.Symbol]*matching.Engine
}

func NewEngineServer() *EngineServer {
	worker := &EngineServer{
		Engines: make(map[pkg.Symbol]*matching.Engine),
	}

	worker.Reload(pkg.Symbol{BaseCurrency: "ALL", QuoteCurrency: "ALL"})

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
		w.InitializeEngine(matching_payload.Symbol)
	case pkg.ActionReload:
		w.Reload(matching_payload.Symbol)
	default:
		config.Logger.Fatalf("Unknown action: %s", matching_payload.Action)
	}

	return nil
}

func (s *EngineServer) SubmitOrder(order *pkg.Order) error {
	engine := s.Engines[order.Symbol]

	if engine == nil {
		return errors.New("engine not found")
	}

	if !engine.Initialized {
		return errors.New("engine is not ready")
	}

	if order.Price.IsNegative() || order.StopPrice.IsNegative() {
		config.Logger.Error("price is negative")
		return nil
	}

	engine.Submit(order)
	return nil
}

func (s *EngineServer) CancelOrderWithKey(key *pkg.OrderKey) error {
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

func (s *EngineServer) CancelOrder(order *pkg.Order) error {
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

func (s EngineServer) GetEngineBySymbol(symbol pkg.Symbol) *matching.Engine {
	engine, found := s.Engines[symbol]

	if found {
		return engine
	}

	return nil
}

func (s *EngineServer) FetchOrder(ctx context.Context, req *GrpcEngine.FetchOrderRequest) (*GrpcEngine.FetchOrderResponse, error) {
	key := req.OrderKey.ToOrderKey()
	engine := s.GetEngineBySymbol(key.Symbol)
	pl := matching.NewPriceLevel(key.Side, key.Price)

	var price_levels *redblacktree.Tree
	if key.Side == pkg.SideSell {
		price_levels = engine.OrderBook.Depth.Asks
	} else {
		price_levels = engine.OrderBook.Depth.Bids
	}

	value, found := price_levels.Get(pl.Key())
	if !found {
		return nil, fmt.Errorf("can't find order with uuid: %s in orderbook", key.UUID.String())
	}

	price_level := value.(*matching.PriceLevel)
	order := price_level.Get(key)

	return &GrpcEngine.FetchOrderResponse{
		Order: &GrpcOrder.Order{
			Id:       order.ID,
			Uuid:     order.UUID[:],
			MemberId: order.MemberID,
			Symbol:   &GrpcSymbol.Symbol{BaseCurrency: order.Symbol.BaseCurrency, QuoteCurrency: order.Symbol.QuoteCurrency},
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
	engine := s.GetEngineBySymbol(req.Symbol.ToSymbol())
	price := engine.OrderBook.MarketPrice

	return &GrpcEngine.FetchMarketPriceResponse{
		Price: &GrpcUtils.Decimal{
			Val: price.CoefficientInt64(),
			Exp: price.Exponent(),
		},
	}, nil
}

func (s *EngineServer) FetchOrderBook(ctx context.Context, req *GrpcEngine.FetchOrderBookRequest) (*GrpcEngine.FetchOrderBookResponse, error) {
	engine := s.GetEngineBySymbol(req.Symbol.ToSymbol())
	response := engine.OrderBook.Depth.FetchOrderBook(req.Limit)

	return response, nil
}

func (s *EngineServer) CalcMarketOrder(ctx context.Context, req *GrpcEngine.CalcMarketOrderRequest) (*GrpcEngine.CalcMarketOrderResponse, error) {
	engine := s.GetEngineBySymbol(req.Symbol.ToSymbol())
	response := engine.OrderBook.CalcMarketOrder(pkg.OrderSide(req.Side), req.Quantity.ToNullDecimal(), req.Volume.ToNullDecimal())

	return response, nil
}

func (s *EngineServer) Reload(symbol pkg.Symbol) {
	if symbol.BaseCurrency == "ALL" && symbol.QuoteCurrency == "ALL" {
		var markets []models.Market
		config.DataBase.Where("state = ?", "enabled").Find(&markets)
		for _, market := range markets {
			s.InitializeEngine(market.GetSymbol())
		}
		config.Logger.Info("All engines reloaded.")
	} else {
		s.InitializeEngine(symbol)
	}
}

func (s *EngineServer) InitializeEngine(symbol pkg.Symbol) {
	lastPrice := decimal.Zero
	trade := models.GetLastTradeFromInflux(strings.ToLower(symbol.ToSymbol("")))
	if trade != nil {
		lastPrice = trade.Price
	}

	engine := matching.NewEngine(symbol, lastPrice)
	s.Engines[symbol] = engine
	s.LoadOrders(engine)
	engine.Initialized = true
	config.Logger.Infof("%v engine reloaded.", symbol.String())
}

func (s *EngineServer) LoadOrders(engine *matching.Engine) {
	var orders []models.Order
	config.DataBase.Where("market_id = ? AND state = ?", strings.ToLower(engine.Symbol.ToSymbol("")), models.StateWait).Order("id asc").Find(&orders)
	for _, order := range orders {
		engine.Submit(order.ToMatchingAttributes())
	}
}
