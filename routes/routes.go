package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zsmartex/go-finex/controllers"
)

func SetupRouter() *fiber.App {
	app := fiber.New()

	app.Get("/api/v2/public/timestamp", controllers.GetTimestamp)
	app.Get("/api/v2/public/markets/:market/depth", controllers.GetDepth)

	app.Post("/api/v2/market/orders", controllers.CreateOrder)

	return app
}
