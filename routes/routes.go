package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/zsmartex/finex/controllers"
	"github.com/zsmartex/finex/controllers/market_controllers"
	"github.com/zsmartex/finex/routes/middlewares"
)

func SetupRouter() *fiber.App {
	app := fiber.New()
	app.Use(logger.New())

	api_v2_public := app.Group("/api/v2/public")
	{
		api_v2_public.Get("/timestamp", controllers.GetTimestamp)
		api_v2_public.Get("/global_price", controllers.GetGlobalPrice)
		api_v2_public.Get("/markets/:market/depth", controllers.GetDepth)
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

	return app
}
