package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/zsmartex/finex/controllers"
	"github.com/zsmartex/finex/controllers/admin_controllers"
	"github.com/zsmartex/finex/controllers/ieo_controllers"
	"github.com/zsmartex/finex/controllers/market_controllers"
	"github.com/zsmartex/finex/controllers/referral_controllers"
	"github.com/zsmartex/finex/routes/middlewares"
)

func SetupRouter() *fiber.App {
	app := fiber.New()
	app.Use(logger.New())

	api_v2_public := app.Group("/api/v2/public")
	{
		api_v2_public.Get("/timestamp", controllers.GetTimestamp)
		api_v2_public.Get("/global_price", controllers.GetGlobalPrice)
		api_v2_public.Get("/ieo/list", controllers.GetIEOList)
		api_v2_public.Get("/ieo/:id", controllers.GetIEO)
		api_v2_public.Get("/markets/:market/depth", controllers.GetDepth)
	}

	api_v2_admin := app.Group("/api/v2/admin", middlewares.Authenticate, middlewares.AdminVaildator)
	{
		api_v2_admin.Get("/trades", admin_controllers.GetTrades)
		api_v2_admin.Get("/ieo/list", admin_controllers.GetIEOList)
		api_v2_admin.Get("/ieo/:id", admin_controllers.GetIEO)
		api_v2_admin.Post("/ieo", admin_controllers.CreateIEO)
		api_v2_admin.Put("/ieo", admin_controllers.UpdateIEO)
		api_v2_admin.Delete("/ieo", admin_controllers.DeleteIEO)
		api_v2_admin.Post("/ieo/currencies", admin_controllers.AddIEOCurrencies)
		api_v2_admin.Delete("/ieo/currencies", admin_controllers.RemoveIEOCurrencies)

		api_v2_admin.Post("/orders/:uuid/cancel", admin_controllers.CancelOrder)
		api_v2_admin.Post("/orders/cancel", admin_controllers.CancelAllOrders)
	}

	api_v2_market := app.Group("/api/v2/market", middlewares.Authenticate)
	{
		api_v2_market.Post("/orders", market_controllers.CreateOrder)
		api_v2_market.Get("/orders", market_controllers.GetOrders)
		api_v2_market.Get("/orders/:uuid", market_controllers.GetOrderByUUID)
		api_v2_market.Post("/orders/:uuid/cancel", market_controllers.CancelOrderByUUID)
		api_v2_market.Post("/orders/cancel", market_controllers.CancelAllOrders)
		api_v2_market.Get("/trades", market_controllers.GetTrades)
	}

	api_v2_ieo := app.Group("/api/v2/ieo", middlewares.Authenticate)
	{
		api_v2_ieo.Post("/", ieo_controllers.CreateIEOOrder)
		api_v2_ieo.Get("/:id", ieo_controllers.GetIEO)
	}

	api_v2_referral := app.Group("/api/v2/referral", middlewares.Authenticate)
	{
		api_v2_referral.Get("/", referral_controllers.GetReleaseCommission)
		api_v2_referral.Get("/commissions", referral_controllers.GetCommissions)
	}

	return app
}
