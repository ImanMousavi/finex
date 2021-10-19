package referral_controllers

import (
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/entities"
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

	release_commission_entities := make([]*entities.ReleaseCommissionEntity, 0)
	for _, release_commission := range release_commissions {
		release_commission_entities = append(release_commission_entities, &entities.ReleaseCommissionEntity{
			ID:          release_commission.ID,
			AccountType: release_commission.AccountType,
			MemberID:    release_commission.MemberID,
			EarnedBTC:   release_commission.EarnedBTC,
			FriendTrade: release_commission.FriendTrade,
			Friend:      release_commission.Friend,
			CreatedAt:   release_commission.CreatedAt,
			UpdatedAt:   release_commission.UpdatedAt,
		})
	}

	return c.Status(200).JSON(release_commission_entities)
}

func GetCommissions(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)
	errors := new(helpers.Errors)
	params := new(queries.CommissionQueries)
	if err := c.QueryParser(params); err != nil {
		log.Println(err)

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

	if params.Page == 0 {
		params.Page = 1
	}

	var commissions []*models.Commission

	config.DataBase.Order("id desc").Offset(params.Page*params.Limit-params.Limit).Limit(params.Limit).Find(&commissions, "member_id = ?", CurrentUser.ID)

	commission_entities := make([]*entities.CommissionEntity, 0)

	for _, commission := range commissions {
		commission_entities = append(commission_entities, &entities.CommissionEntity{
			ID:              commission.ID,
			AccountType:     commission.AccountType,
			MemberID:        commission.MemberID,
			FriendUID:       commission.FriendUID,
			EarnAmount:      commission.EarnAmount,
			CurrencyID:      commission.CurrencyID,
			ParentID:        commission.ParentID,
			ParentCreatedAt: commission.ParentCreatedAt,
			CreatedAt:       commission.CreatedAt,
			UpdatedAt:       commission.UpdatedAt,
		})
	}

	c.Response().Header.Add("page", strconv.FormatInt(int64(params.Page), 10))
	c.Response().Header.Add("per-page", strconv.FormatInt(int64(len(commissions)), 10))

	return c.Status(200).JSON(commission_entities)
}
