package controllers

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/zsmartex/finex/config"
	"gitlab.com/zsmartex/finex/controllers/auth"
	"gitlab.com/zsmartex/finex/controllers/helpers"
	"gitlab.com/zsmartex/finex/models"
)

func GetAccounts(c *fiber.Ctx) error {
	CurrentUser := auth.GetCurrentUser(c)

	if CurrentUser == nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"jwt.decode_and_verify"},
		})
	}

	var accounts []models.Account

	config.DataBase.Where(&models.Account{MemberID: CurrentUser.ID}).Find(&accounts)

	c.Status(200).JSON(accounts)

	return nil
}
