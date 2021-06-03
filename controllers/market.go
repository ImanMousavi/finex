package controllers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/controllers/auth"
	"github.com/zsmartex/go-finex/controllers/helpers"
	"github.com/zsmartex/go-finex/models"
)

func CreateOrder(c *fiber.Ctx) error {
	CurrentUser := auth.GetCurrentUser(c)

	if CurrentUser == nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"jwt.decode_and_verify"},
		})
	}

	errors := new(helpers.Errors)
	payload := new(helpers.CreateOrderPayload)

	if err := c.BodyParser(payload); err != nil {
		c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_body"},
		})

		return err
	}

	helpers.Vaildate(payload, errors)
	payload.CreateOrder(CurrentUser, errors)

	if errors.Size() > 0 {
		return c.Status(422).JSON(errors)
	}

	return c.Status(200).JSON(200)
}

func GetOrderByID(c *fiber.Ctx) error {
	CurrentUser := auth.GetCurrentUser(c)

	if CurrentUser == nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"jwt.decode_and_verify"},
		})
	}

	params := new(FindByID)
	c.QueryParser(params)

	var order models.Order

	config.DataBase.Where("id = ? AND member_id = ?", params.ID, CurrentUser.ID).Find(&order)

	return c.Status(200).JSON(order.ToJSON())
}

func CancelOrderByID(c *fiber.Ctx) error {
	CurrentUser := auth.GetCurrentUser(c)

	if CurrentUser == nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"jwt.decode_and_verify"},
		})
	}

	params := new(FindByID)
	c.QueryParser(params)

	var order models.Order

	config.DataBase.Where("id = ? AND member_id = ?", params.ID, CurrentUser.ID).Find(&order)

	// Doing cancel

	return c.Status(200).JSON(order.ToJSON())
}
