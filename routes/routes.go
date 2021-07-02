package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/zsmartex/go-finex/controllers"
	"github.com/zsmartex/go-finex/controllers/market_controllers"
	"github.com/zsmartex/go-finex/routes/middlewares"
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
		api_v2_market.Get("/orders/:id", market_controllers.GetOrderByID)
		api_v2_market.Post("/orders/cancel/:id", market_controllers.CancelOrderByID)
		api_v2_market.Post("/orders/cancel", market_controllers.CancelAllOrders)
		api_v2_market.Get("/trades", market_controllers.GetTrades)
	}

	return app
}
