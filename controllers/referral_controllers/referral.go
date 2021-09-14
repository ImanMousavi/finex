package referral_controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/controllers/queries"
	"github.com/zsmartex/finex/models"
)

func GetReleaseCommission(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

	var errors = new(helpers.Errors)
	params := new(queries.ReleaseCommissionQueries)

	if err := c.QueryParser(params); err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_query"},
		})
	}

	helpers.Vaildate(params, errors)
	if errors.Size() > 0 {
		return c.Status(422).JSON(errors)
	}

	var release_commissions []*models.ReleaseCommission

	config.DataBase.
		Where(
			"member_id = ? AND created_at >= ? AND created_at <= ?",
			CurrentUser.ID,
			time.Unix(params.TimeFrom, 0),
			time.Unix(params.TimeTo, 0),
		).
		Find(&release_commissions)

	return c.Status(200).JSON(release_commissions)
}

func GetCommissions(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

	var commissions []*models.Commission

	config.DataBase.Find(&commissions, "member_id = ?", CurrentUser.ID)

	return c.Status(200).JSON(commissions)
}
