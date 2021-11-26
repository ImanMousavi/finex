package ieo_controllers

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers"
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/models"
	"gorm.io/gorm"
)

type CreateIEOOrderPayload struct {
	IEOID           int64           `json:"ieo_id" form:"ieo_id"`
	PaymentCurrency string          `json:"payment_currency" form:"payment_currency"`
	Quantity        decimal.Decimal `json:"quantity" form:"quantity"`
}

func CreateIEOOrder(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

	var payload *CreateIEOOrderPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(500).JSON(helpers.Errors{
			Errors: []string{"server.method.invalid_body"},
		})
	}

	var ieo *models.IEO
	if result := config.DataBase.First(&ieo, payload.IEOID); errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"record.not_found"},
		})
	}

	if !ieo.IsEnabled() {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.not_enabled"},
		})
	}

	if ieo.IsEnded() {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.is_ended"},
		})
	}

	if ieo.IsCompleted() {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.is_completed"},
		})
	}

	if !ieo.IsStarted() {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.not_started"},
		})
	}

	if payload.PaymentCurrency != strings.ToLower(payload.PaymentCurrency) {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.invalid_payment_currency"},
		})
	}

	if !payload.Quantity.IsPositive() {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.non_positive_quantity"},
		})
	}

	if payload.Quantity.LessThan(ieo.MinAmount) {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.low_quantity"},
		})
	}

	found_payment_currency := false
	for _, currency_id := range ieo.PaymentCurrencies() {
		if currency_id == payload.PaymentCurrency {
			found_payment_currency = true
		}
	}

	if !found_payment_currency {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.invalid_payment_currency"},
		})
	}

	if ieo.ExecutedQuantity.Add(payload.Quantity).GreaterThan(ieo.OriginQuantity) {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.out_of_stock"},
		})
	}

	user_bought_quantity := ieo.MemberBoughtQuantity(CurrentUser.ID)

	if user_bought_quantity.Add(payload.Quantity).GreaterThan(ieo.LimitPerUser) {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.reached_limit"},
		})
	}

	if CurrentUser.Level < 3 {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.kyc_required"},
		})
	}

	ieo_order := &models.IEOOrder{
		IEOID:             payload.IEOID,
		UUID:              uuid.New(),
		MemberID:          CurrentUser.ID,
		PaymentCurrencyID: payload.PaymentCurrency,
		Price:             ieo.GetPriceByParent(payload.PaymentCurrency),
		Quantity:          payload.Quantity,
		State:             models.StatePending,
	}

	member_balance := ieo_order.MemberBalance()

	if member_balance.LessThan(ieo_order.Total()) {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.account.insufficient_balance"},
		})
	}

	config.DataBase.Create(&ieo_order)

	payload_ieo_order_processor_attrs, _ := json.Marshal(ieo_order.ToJSON())
	config.Nats.Publish("ieo_order_processor", payload_ieo_order_processor_attrs)

	return c.Status(201).JSON(ieo_order.ToJSON())
}

func GetIEO(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

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

	entity := controllers.IEOToEntity(ieo)
	entity.BoughtQuantity = ieo.MemberBoughtQuantity(CurrentUser.ID)

	return c.Status(200).JSON(entity)
}
