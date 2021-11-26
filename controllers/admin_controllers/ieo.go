package admin_controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/admin_controllers/entities"
	"github.com/zsmartex/finex/controllers/admin_controllers/queries"
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/models"
	"github.com/zsmartex/finex/types"
)

func IEOToEntity(ieo *models.IEO) *entities.IEO {
	return &entities.IEO{
		ID:                  ieo.ID,
		CurrencyID:          ieo.CurrencyID,
		MainPaymentCurrency: ieo.MainPaymentCurrency,
		Price:               ieo.Price,
		OriginQuantity:      ieo.OriginQuantity,
		ExecutedQuantity:    ieo.ExecutedQuantity,
		PaymentCurrencies:   ieo.PaymentCurrencies(),
		MinAmount:           ieo.MinAmount,
		State:               ieo.State,
		StartTime:           ieo.StartTime.Unix(),
		EndTime:             ieo.EndTime.Unix(),
		Data:                ieo.Data,
		BannerUrl:           ieo.BannerUrl,
		LimitPerUser:        ieo.LimitPerUser,
		CreatedAt:           ieo.CreatedAt,
		UpdatedAt:           ieo.UpdatedAt,
	}
}

func ValidateIEOPayload(payload *queries.IEOPayload) *helpers.Errors {
	e := new(helpers.Errors)

	if !payload.Price.IsPositive() {
		e.Errors = append(e.Errors, "Price must be positive")
	}

	if !payload.MinAmount.IsPositive() {
		e.Errors = append(e.Errors, "Min Amount must be positive")
	}

	if payload.State != types.MarketStateDisabled && payload.State != types.MarketStateEndabled {
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

	ieo_entities := make([]*entities.IEO, 0)

	for _, ieo := range lst_ieo {
		ieo_entities = append(ieo_entities, IEOToEntity(ieo))
	}

	return c.Status(200).JSON(ieo_entities)
}

func GetIEO(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_query"},
		})
	}

	var ieo *models.IEO
	if result := config.DataBase.Find(&ieo, id); result.Error != nil {
		return c.Status(404).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	return c.Status(200).JSON(IEOToEntity(ieo))
}

func CreateIEO(c *fiber.Ctx) error {
	var payload *queries.IEOPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_body"},
		})
	}

	if errors := ValidateIEOPayload(payload); errors != nil {
		return c.Status(422).JSON(errors)
	}

	ieo := &models.IEO{
		CurrencyID:          payload.CurrencyID,
		MainPaymentCurrency: payload.MainPaymentCurrency,
		Price:               payload.Price,
		OriginQuantity:      payload.OriginQuantity,
		ExecutedQuantity:    decimal.Decimal{},
		LimitPerUser:        payload.LimitPerUser,
		MinAmount:           payload.MinAmount,
		State:               payload.State,
		StartTime:           time.Unix(payload.StartTime, 0),
		EndTime:             time.Unix(payload.EndTime, 0),
		Data:                payload.Data,
		BannerUrl:           payload.BannerUrl,
	}

	if result := config.DataBase.Create(&ieo); result.Error == nil {
		ieo_currency := &models.IEOPaymentCurrency{
			CurrencyID: payload.MainPaymentCurrency,
			IEOID:      ieo.ID,
		}

		config.DataBase.Create(&ieo_currency)
	}

	return c.Status(200).JSON(IEOToEntity(ieo))
}

func UpdateIEO(c *fiber.Ctx) error {
	var payload *queries.IEOPayload
	if err := c.BodyParser(&payload); err != nil {
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
	ieo.OriginQuantity = payload.OriginQuantity
	ieo.LimitPerUser = payload.LimitPerUser
	ieo.MinAmount = payload.MinAmount
	ieo.State = payload.State
	ieo.StartTime = time.Unix(payload.StartTime, 0)
	ieo.EndTime = time.Unix(payload.EndTime, 0)
	ieo.Data = payload.Data
	ieo.BannerUrl = payload.BannerUrl

	config.DataBase.Save(&ieo)

	return c.Status(200).JSON(IEOToEntity(ieo))
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

type PayloadIEOCurrency struct {
	ID         int64    `json:"id"`
	Currencies []string `json:"currencies"`
}

func AddIEOCurrencies(c *fiber.Ctx) error {
	var payload *PayloadIEOCurrency

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_body"},
		})
	}

	var ieo *models.IEO
	if result := config.DataBase.First(&ieo, payload.ID); result.Error != nil {
		return c.Status(404).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	for _, currency_id := range payload.Currencies {
		if ieo.CurrencyID == currency_id {
			return c.Status(404).JSON(helpers.Errors{
				Errors: []string{"Payment currency can not same as target currency"},
			})
		}
	}

	for _, currency_id := range payload.Currencies {
		ieo_currency := &models.IEOPaymentCurrency{
			CurrencyID: currency_id,
			IEOID:      ieo.ID,
		}

		config.DataBase.Create(&ieo_currency)
	}

	return c.Status(200).JSON(200)
}

func RemoveIEOCurrencies(c *fiber.Ctx) error {
	var payload *PayloadIEOCurrency

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_message_body"},
		})
	}

	var ieo *models.IEO
	if result := config.DataBase.First(&ieo, payload.ID); result.Error != nil {
		return c.Status(404).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	for _, currency_id := range payload.Currencies {
		config.DataBase.Where("ieo_id = ? AND currency_id = ?", ieo.ID, currency_id).Delete(&models.IEOPaymentCurrency{})
	}

	return c.Status(200).JSON(200)
}
