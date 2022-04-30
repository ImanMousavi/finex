package controllers

import (
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/zsmartex/pkg"
	clientEngine "github.com/zsmartex/pkg/client/engine"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/entities"
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/controllers/queries"
	"github.com/zsmartex/finex/models"
	"github.com/zsmartex/finex/types"
	engineGrpc "github.com/zsmartex/pkg/Grpc/engine"
	GrpcSymbol "github.com/zsmartex/pkg/Grpc/symbol"
)

func IEOToEntity(ieo *models.IEO) *entities.IEO {
	return &entities.IEO{
		ID:                  ieo.ID,
		CurrencyID:          ieo.CurrencyID,
		Price:               ieo.Price,
		MainPaymentCurrency: ieo.MainPaymentCurrency,
		PaymentCurrencies:   ieo.PaymentCurrencies(),
		LimitPerUser:        ieo.LimitPerUser,
		MinAmount:           ieo.MinAmount,
		ExecutedQuantity:    ieo.ExecutedQuantity,
		OriginQuantity:      ieo.OriginQuantity,
		StartTime:           ieo.StartTime.Unix(),
		EndTime:             ieo.EndTime.Unix(),
		Ended:               ieo.IsEnded(),
		BannerUrl:           ieo.BannerUrl,
		Data:                ieo.Data,
		Distributors:        ieo.Distributors(),
		CreatedAt:           ieo.CreatedAt,
		UpdatedAt:           ieo.UpdatedAt,
	}
}

func GetTimestamp(c *fiber.Ctx) error {

	c.Status(200).JSON(time.Now())

	return nil
}

func GetIEOList(c *fiber.Ctx) error {
	var lst_ieo []*models.IEO

	config.DataBase.Find(&lst_ieo, "state = ?", types.MarketStateEndabled)

	ieo_entities := make([]*entities.IEO, 0)

	for _, ieo := range lst_ieo {
		ieo_entities = append(ieo_entities, IEOToEntity(ieo))
	}

	return c.Status(200).JSON(ieo_entities)
}

func GetIEO(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_query"},
		})
	}

	var ieo *models.IEO
	if result := config.DataBase.Find(&ieo, id); result.Error != nil {
		return c.Status(404).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	return c.Status(200).JSON(IEOToEntity(ieo))
}

func GetDepth(c *fiber.Ctx) error {
	var errs = new(helpers.Errors)

	marketID := c.Params("market")
	params := new(queries.DepthQuery)
	if err := c.QueryParser(params); err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_query"},
		})
	}

	helpers.Vaildate(params, errs)

	if errs.Size() > 0 {
		return c.Status(422).JSON(errs)
	}

	var market *models.Market
	if result := config.DataBase.First(&market, "symbol = ?", marketID); errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"public.market.doesnt_exist"},
		})
	}

	matching_client := clientEngine.NewMatchingClient()
	defer matching_client.Close()

	if params.Limit == 0 {
		params.Limit = 100
	}

	depth := pkg.DepthJSON{
		Asks:     [][]decimal.Decimal{},
		Bids:     [][]decimal.Decimal{},
		Sequence: 0,
	}
	symbol := market.GetSymbol()
	fetch_orderbook_response, err := matching_client.FetchOrderBook(&engineGrpc.FetchOrderBookRequest{
		Symbol: &GrpcSymbol.Symbol{BaseCurrency: symbol.BaseCurrency, QuoteCurrency: symbol.QuoteCurrency},
		Limit:  params.Limit,
	})
	if err != nil {
		log.Println(err)
		config.Logger.Errorf("Failed to fetch %s depth, Error: %v", symbol.String(), err)

		return c.Status(200).JSON(depth)
	}

	for _, bookOrder := range fetch_orderbook_response.Asks {
		price := bookOrder.PriceQuantity[0].ToDecimal()
		amount := bookOrder.PriceQuantity[1].ToDecimal()

		depth.Asks = append(depth.Asks, []decimal.Decimal{price, amount})
	}

	for _, bookOrder := range fetch_orderbook_response.Bids {
		price := bookOrder.PriceQuantity[0].ToDecimal()
		amount := bookOrder.PriceQuantity[1].ToDecimal()

		depth.Bids = append(depth.Bids, []decimal.Decimal{price, amount})
	}

	depth.Sequence = fetch_orderbook_response.Sequence

	return c.Status(200).JSON(depth)
}

func GetGlobalPrice(c *fiber.Ctx) error {
	var global_price types.GlobalPrice

	if err := config.Redis.GetKey("finex:h24:global_price", global_price); err != nil {
		config.Logger.Errorf("Failed to fetch global price %v", err)

		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"public.global_price.failed"},
		})
	}

	return c.Status(200).JSON(global_price)
}
