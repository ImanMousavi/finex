package controllers

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/controllers/helpers"
	"github.com/zsmartex/go-finex/types"
)

func GetTimestamp(c *fiber.Ctx) error {

	c.Status(200).JSON(time.Now())

	return nil
}

func GetDepth(c *fiber.Ctx) error {
	market := c.Params("market")
	depth := types.Depth{
		Asks:     [][]decimal.Decimal{},
		Bids:     [][]decimal.Decimal{},
		Sequence: 0,
	}

	var err error
	msg, err := config.Nats.Request("depth:"+market, []byte(market), 10*time.Millisecond)

	if err != nil {
		return c.Status(200).JSON(depth)
	}

	err = json.Unmarshal(msg.Data, &depth)

	if err != nil {
		return c.Status(200).JSON(depth)
	}

	return c.Status(200).JSON(depth)
}

func GetGlobalPrice(c *fiber.Ctx) error {
	var global_price types.GlobalPrice

	if err := config.Redis.GetKey("finex:h24:global_price", &global_price); err != nil {
		log.Fatalln(err)
		c.Status(422).JSON(helpers.Errors{
			Errors: []string{"public.global_price.failed"},
		})

		return err
	}

	return c.Status(200).JSON(global_price)
}
