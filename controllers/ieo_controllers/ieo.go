package ieo_controllers

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/controllers/helpers"
	"github.com/zsmartex/finex/models"
	"github.com/zsmartex/finex/models/datatypes"
	"gorm.io/gorm"
)

type CreateIEOOrderPayload struct {
	IEOID         uint64          `json:"ieo_id" form:"ieo_id"`
	QuoteCurrency string          `json:"quote_currency" form:"quote_currency"`
	Quantity      decimal.Decimal `json:"quantity" form:"quantity"`
}

func CreateIEOOrder(c *fiber.Ctx) error {
	CurrentUser := c.Locals("CurrentUser").(*models.Member)

	var payload *CreateIEOOrderPayload
	if err := c.QueryParser(&payload); err != nil {
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

	if !ieo.IsStarted() {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.not_started"},
		})
	}

	if payload.QuoteCurrency == strings.ToLower(payload.QuoteCurrency) {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.invalid_quote_currency"},
		})
	}

	if !payload.Quantity.IsPositive() {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.non_positive_quantity"},
		})
	}

	var payment_currency *datatypes.IEOPaymentCurrency

	found_payment_currency := false
	for _, pu := range ieo.PaymentCurrencies {
		if pu.Currency == payload.QuoteCurrency {
			payment_currency = &pu
			found_payment_currency = true
		}
	}

	if !found_payment_currency {
		return c.Status(422).JSON(helpers.Errors{
			Errors: []string{"market.ieo.invalid_quote_currency"},
		})
	}

	ieo_order := &models.IEOOrder{
		IEOID:    payload.IEOID,
		MemberID: CurrentUser.ID,
		Ask:      ieo.CurrencyID,
		Bid:      payload.QuoteCurrency,
		Price:    ieo.GetPriceByParent(payment_currency.Currency),
		Quantity: payload.Quantity,
		Bouns:    payment_currency.Bouns,
		State:    models.StatePending,
	}

	config.DataBase.Create(&ieo_order)

	member_balance := ieo_order.MemberBalance()

	if member_balance.LessThan(ieo_order.Total()) {
		return errors.New("market.account.insufficient_balance")
	}

	payload_ieo_order_processor_attrs, _ := json.Marshal(ieo_order)
	config.Nats.Publish("ieo_order_processor", payload_ieo_order_processor_attrs)

	return c.Status(201).JSON(ieo_order.ToJSON())
}
