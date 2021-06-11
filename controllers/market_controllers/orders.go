package market_controllers

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/controllers/auth"
	"github.com/zsmartex/go-finex/controllers/entities"
	"github.com/zsmartex/go-finex/controllers/helpers"
	"github.com/zsmartex/go-finex/controllers/queries"
	"github.com/zsmartex/go-finex/matching"
	"github.com/zsmartex/go-finex/models"
	"github.com/zsmartex/go-finex/types"
)

func CreateOrder(c *fiber.Ctx) error {
	CurrentUser := auth.GetCurrentUser(c)

	if CurrentUser == nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"jwt.decode_and_verify"},
		})
	}

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
	CurrentUser := auth.GetCurrentUser(c)

	if CurrentUser == nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"jwt.decode_and_verify"},
		})
	}

	var orders []models.Order
	orders_json := make([]entities.OrderEntities, 0)

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

	return c.Status(200).JSON(orders_json)
}

func GetOrderByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.order.invaild_id"},
		})
	}

	CurrentUser := auth.GetCurrentUser(c)

	if CurrentUser == nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"jwt.decode_and_verify"},
		})
	}

	var order *models.Order

	result := config.DataBase.Where("id = ? AND member_id = ?", id, CurrentUser.ID).First(&order)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	return c.Status(200).JSON(order.ToJSON())
}

func CancelOrderByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.order.invaild_id"},
		})
	}
	CurrentUser := auth.GetCurrentUser(c)

	if CurrentUser == nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"jwt.decode_and_verify"},
		})
	}

	var order *models.Order

	result := config.DataBase.Where("id = ? AND member_id = ?", id, CurrentUser.ID).First(&order)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	// Doing cancel
	payload_matching_attrs, _ := json.Marshal(map[string]interface{}{
		"action": matching.ActionCancel,
		"order":  order.ToMatchingAttributes(),
	})
	config.Nats.Publish("matching", payload_matching_attrs)

	return c.Status(200).JSON(order.ToJSON())
}

func CancelAllOrders(c *fiber.Ctx) error {
	var orders []*models.Order
	params := new(queries.CancelOrderParams)

	if err := c.BodyParser(params); err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_body"},
		})
	}

	CurrentUser := auth.GetCurrentUser(c)

	if CurrentUser == nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"jwt.decode_and_verify"},
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
		payload_matching_attrs, _ := json.Marshal(map[string]interface{}{
			"action": matching.ActionCancel,
			"order":  order.ToMatchingAttributes(),
		})
		config.Nats.Publish("matching", payload_matching_attrs)
	}

	var ordersJSON []entities.OrderEntities

	for _, order := range orders {
		ordersJSON = append(ordersJSON, order.ToJSON())
	}

	return c.Status(201).JSON(ordersJSON)
}
