package admin_controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/admin_controllers/queries"
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/models"
	"github.com/zsmartex/finex/types"
)

func ValidateIEOPayload(payload *queries.IEOPayload) *helpers.Errors {
	e := new(helpers.Errors)

	if !payload.Price.IsPositive() {
		e.Errors = append(e.Errors, "Price must be positive")
	}

	validated_main_currency := false
	for _, v := range payload.PaymentCurrencies {
		if v.Currency == payload.MainPaymentCurrency {
			validated_main_currency = true
		}
	}

	if !validated_main_currency {
		e.Errors = append(e.Errors, "Main Currency must be includes in Payment Currencies")
	}

	if !payload.MinAmount.IsPositive() {
		e.Errors = append(e.Errors, "Min Amount must be positive")
	}

	if payload.State != types.MarketStateDisabled || payload.State != types.MarketStateEndabled {
		e.Errors = append(e.Errors, "Unknow State")
	}

	if payload.EndTime <= payload.StartTime {
		e.Errors = append(e.Errors, "Start time must be before End time")
	}

	if len(e.Errors) > 0 {
		return e
	} else {
		return nil
	}
}

func GetIEOList(c *fiber.Ctx) error {
	var lst_ieo []*models.IEO

	config.DataBase.Find(&lst_ieo)

	return c.Status(200).JSON(lst_ieo)
}

func CreateIEO(c *fiber.Ctx) error {
	var payload *queries.IEOPayload
	if err := c.QueryParser(&payload); err != nil {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_body"},
		})
	}

	if errors := ValidateIEOPayload(payload); errors != nil {
		return c.Status(422).JSON(errors)
	}

	ieo := &models.IEO{
		ID:                  payload.ID,
		CurrencyID:          payload.CurrencyID,
		MainPaymentCurrency: payload.MainPaymentCurrency,
		Price:               payload.Price,
		PaymentCurrencies:   payload.PaymentCurrencies,
		MinAmount:           payload.MinAmount,
		State:               payload.State,
		StartTime:           time.Unix(payload.StartTime, 0),
		EndTime:             time.Unix(payload.EndTime, 0),
	}

	config.DataBase.Create(&ieo)

	return c.Status(200).JSON(ieo)
}

func UpdateIEO(c *fiber.Ctx) error {
	var payload *queries.IEOPayload
	if err := c.QueryParser(&payload); err != nil {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_body"},
		})
	}

	if errors := ValidateIEOPayload(payload); errors != nil {
		return c.Status(422).JSON(errors)
	}

	var ieo *models.IEO
	if result := config.DataBase.First(&ieo, payload.ID); result.Error != nil {
		return c.Status(404).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	ieo.MainPaymentCurrency = payload.MainPaymentCurrency
	ieo.Price = payload.Price
	ieo.PaymentCurrencies = payload.PaymentCurrencies
	ieo.MinAmount = payload.MinAmount
	ieo.State = payload.State
	ieo.StartTime = time.Unix(payload.StartTime, 0)
	ieo.EndTime = time.Unix(payload.EndTime, 0)

	config.DataBase.Save(&ieo)

	return c.Status(200).JSON(ieo)
}

func DeleteIEO(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"Can not find ieo"},
		})
	}

	var ieo *models.IEO
	if result := config.DataBase.First(&ieo, id); result.Error != nil {
		return c.Status(404).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	config.DataBase.Delete(&ieo)

	return c.Status(200).JSON(200)
}
