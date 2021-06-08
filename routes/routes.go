package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zsmartex/go-finex/controllers"
	"github.com/zsmartex/go-finex/controllers/market_controllers"
)

func SetupRouter() *fiber.App {
	app := fiber.New()

	app.Get("/api/v2/public/timestamp", controllers.GetTimestamp)
	app.Get("/api/v2/public/global_price", controllers.GetGlobalPrice)
	app.Get("/api/v2/public/markets/:market/depth", controllers.GetDepth)

	app.Post("/api/v2/market/orders", market_controllers.CreateOrder)
	app.Get("/api/v2/market/orders", market_controllers.GetOrders)
	app.Get("/api/v2/market/orders/:id", market_controllers.GetOrderByID)
	app.Post("/api/v2/market/orders/cancel/:id", market_controllers.CancelOrderByID)
	app.Post("/api/v2/market/orders/cancel", market_controllers.CancelAllOrders)
	app.Get("/api/v2/market/trades", market_controllers.GetTrades)

	return app
}
