package admin_controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/entities"
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/controllers/queries"
	"github.com/zsmartex/finex/models"
	"github.com/zsmartex/finex/types"
)

func GetTrades(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

	if CurrentUser.Role != "admin" && CurrentUser.Role != "superadmin" {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"authz.invalid_permission"},
		})
	}

	var errors = new(helpers.Errors)
	var trades []models.Trade

	params := new(queries.TradeFilters)

	if err := c.QueryParser(params); err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_query"},
		})
	}

	helpers.Vaildate(params, errors)

	if errors.Size() > 0 {
		return c.Status(422).JSON(errors)
	}

	if len(params.OrderBy) == 0 {
		params.OrderBy = types.OrderByDesc
	}

	tx := config.DataBase.Order("id " + params.OrderBy)

	if len(params.Market) > 0 {
		tx = tx.Where("market_id = ?", params.Market)
	}

	if len(params.Type) > 0 {
		tx = tx.Where("taker_type = ?", params.Type)
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
	tx.Find(&trades)

	var trades_json []entities.TradeEntities
	for _, trade := range trades {
		trades_json = append(trades_json, trade.ToJSON(CurrentUser))
	}

	return c.Status(200).JSON(trades_json)
}
