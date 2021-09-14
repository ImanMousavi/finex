package market_controllers

import (
	"strconv"
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

	tx := config.DataBase.Order("id "+params.OrderBy).Where("maker_id = ? OR taker_id = ?", CurrentUser.ID, CurrentUser.ID)

	if len(params.Market) > 0 {
		tx = tx.Where("market_id = ?", params.Market)
	}

	if len(params.Type) > 0 {
		var opposite_type_param types.TakerType

		if params.Type == types.TypeBuy {
			opposite_type_param = types.TypeSell
		} else {
			opposite_type_param = types.TypeBuy
		}

		tx = tx.Where("(taker_id = ? AND taker_type = ?) OR (maker_id = ? AND taker_type = ?)", CurrentUser.ID, params.Type, CurrentUser.ID, opposite_type_param)
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

	var trades_json []entities.TradeEntity
	for _, trade := range trades {
		trades_json = append(trades_json, trade.ForUser(CurrentUser))
	}

	c.Response().Header.Add("page", strconv.FormatInt(int64(params.Page), 10))
	c.Response().Header.Add("per-page", strconv.FormatInt(int64(len(trades)), 10))

	return c.Status(200).JSON(trades_json)
}
