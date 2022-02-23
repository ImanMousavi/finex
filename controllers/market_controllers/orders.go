package market_controllers

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/entities"
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/controllers/queries"
	"github.com/zsmartex/finex/models"
	"github.com/zsmartex/finex/types"

	"github.com/zsmartex/pkg"
)

func CreateOrder(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

	errors := new(helpers.Errors)
	payload := new(helpers.CreateOrderParams)

	if err := c.BodyParser(payload); err != nil {
		c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_body"},
		})

		return err
	}

	helpers.Vaildate(payload, errors)
	order := payload.CreateOrder(CurrentUser, errors)

	if errors.Size() > 0 {
		return c.Status(422).JSON(errors)
	}

	return c.Status(201).JSON(order.ToJSON())
}

func GetOrders(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

	var orders []*models.Order
	orders_json := make([]entities.OrderEntity, 0)

	params := new(queries.OrderFilters)
	if err := c.QueryParser(params); err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_query"},
		})
	}

	if len(params.OrderBy) == 0 {
		params.OrderBy = types.OrderByDesc
	}

	tx := config.DataBase.Order("updated_at "+params.OrderBy).Where("member_id = ?", CurrentUser.ID)

	if len(params.Market) > 0 {
		tx = tx.Where("market_id = ?", params.Market)
	}

	if len(params.BaseUnit) > 0 {
		tx = tx.Where("ask = ?", params.BaseUnit)
	}

	if len(params.BaseUnit) > 0 {
		tx = tx.Where("bid = ?", params.QuoteUnit)
	}

	if len(params.Type) > 0 {
		if params.Type == types.SideBuy {
			tx = tx.Where("type = ?", models.SideBuy)
		} else {
			tx = tx.Where("type = ?", models.SideSell)
		}
	}

	if len(params.State) > 0 {
		state := models.StateWait

		switch params.State {
		case "pending":
			state = models.StatePending
		case "wait":
			state = models.StateWait
		case "cancel":
			state = models.StateCancel
		case "done":
			state = models.StateDone
		case "reject":
			state = models.StateReject
		default:
			return c.Status(422).JSON(helpers.Errors{
				Errors: []string{"market.order.invalid_state"},
			})
		}

		tx = tx.Where("state = ?", state)
	}

	if params.TimeFrom > 0 {
		time_from := time.Unix(params.TimeFrom, 0)
		tx = tx.Where("created_at >= ?", time_from)
	}

	if params.TimeTo > 0 {
		time_to := time.Unix(params.TimeTo, 0)
		tx = tx.Where("created_at < ?", time_to)
	}

	if params.Limit == 0 {
		params.Limit = 100
	}

	if params.Page == 0 {
		params.Page = 1
	}

	tx = tx.Offset(params.Page*params.Limit - params.Limit).Limit(params.Limit)

	tx.Find(&orders)

	for _, order := range orders {
		orders_json = append(orders_json, order.ToJSON())
	}

	c.Response().Header.Add("page", strconv.FormatInt(int64(params.Page), 10))
	c.Response().Header.Add("per-page", strconv.FormatInt(int64(len(orders)), 10))

	return c.Status(200).JSON(orders_json)
}

func GetOrderByUUID(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

	uuid, err := uuid.Parse(c.Params("uuid"))
	if err != nil {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.order.invaild_uuid"},
		})
	}

	var order *models.Order

	result := config.DataBase.Where("uuid = ? AND member_id = ?", uuid, CurrentUser.ID).First(&order)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	return c.Status(200).JSON(order.ToJSON())
}

func CancelOrderByUUID(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

	uuid, err := uuid.Parse(c.Params("uuid"))
	if err != nil {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.order.invaild_uuid"},
		})
	}

	var order *models.Order

	result := config.DataBase.Where("uuid = ? AND member_id = ?", uuid, CurrentUser.ID).First(&order)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	// Doing cancel
	config.KafkaProducer.Produce("matching", map[string]interface{}{
		"action": pkg.ActionCancel,
		"order":  order.ToMatchingAttributes(),
	})

	return c.Status(200).JSON(order.ToJSON())
}

func CancelAllOrders(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

	var orders []*models.Order
	params := new(queries.CancelOrderParams)

	if err := c.BodyParser(params); err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_body"},
		})
	}

	tx := config.DataBase.Where("member_id = ? AND state = ?", CurrentUser.ID, models.StateWait)

	if len(params.Market) > 0 {
		tx = tx.Where("market_id = ?", params.Market)
	}

	if len(params.Side) > 0 {
		var nSide models.OrderSide

		if params.Side == types.TypeBuy {
			nSide = models.SideBuy
		} else if params.Side == types.TypeSell {
			nSide = models.SideSell
		} else {
			return c.Status(422).JSON(helpers.Errors{
				Errors: []string{"market.orders.invalid_side"},
			})
		}

		tx = tx.Where("type = ?", nSide)
	}

	tx.Find(&orders)

	for _, order := range orders {
		// Doing cancel
		config.KafkaProducer.Produce("matching", map[string]interface{}{
			"action": pkg.ActionCancel,
			"order":  order.ToMatchingAttributes(),
		})
	}

	var ordersJSON []entities.OrderEntity

	for _, order := range orders {
		ordersJSON = append(ordersJSON, order.ToJSON())
	}

	return c.Status(201).JSON(ordersJSON)
}
