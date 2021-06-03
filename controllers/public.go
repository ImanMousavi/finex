package controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"gitlab.com/zsmartex/finex/models"
	"gitlab.com/zsmartex/finex/types"
)

func GetTimestamp(c *fiber.Ctx) error {

	c.Status(200).JSON(time.Now())

	return nil
}

func GetDepth(c *fiber.Ctx) error {
	market := c.Params("market")
	depth := types.Depth{
		Asks:     models.GetDepth(models.SideSell, market),
		Bids:     models.GetDepth(models.SideBuy, market),
		Sequence: 0,
	}

	c.Status(200).JSON(depth)

	return nil
}
