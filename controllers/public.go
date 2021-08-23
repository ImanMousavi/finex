package controllers

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/controllers/queries"
	"github.com/zsmartex/finex/types"
	"github.com/zsmartex/pkg"
)

func GetTimestamp(c *fiber.Ctx) error {

	c.Status(200).JSON(time.Now())

	return nil
}

func GetDepth(c *fiber.Ctx) error {
	var errors = new(helpers.Errors)

	market := c.Params("market")
	params := new(queries.DepthQuery)
	if err := c.QueryParser(params); err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_query"},
		})
	}

	helpers.Vaildate(params, errors)

	if errors.Size() > 0 {
		return c.Status(422).JSON(errors)
	}

	if params.Limit == 0 {
		params.Limit = 100
	}

	depth := pkg.DepthJSON{
		Asks:     [][]decimal.Decimal{},
		Bids:     [][]decimal.Decimal{},
		Sequence: 0,
	}

	var err error
	payload, _ := json.Marshal(pkg.GetDepthPayload{
		Market: market,
		Limit:  params.Limit,
	})
	msg, err := config.Nats.Request("finex:depth:"+market, payload, 5*time.Second)

	if err != nil {
		return c.Status(200).JSON(depth)
	}

	json.Unmarshal(msg.Data, &depth)

	return c.Status(200).JSON(depth)
}

func GetGlobalPrice(c *fiber.Ctx) error {
	var global_price types.GlobalPrice

	if err := config.Redis.GetKey("finex:h24:global_price", &global_price); err != nil {
		config.Logger.Errorf("Error %v", err.Error())
		c.Status(422).JSON(helpers.Errors{
			Errors: []string{"public.global_price.failed"},
		})

		return err
	}

	return c.Status(200).JSON(global_price)
}
