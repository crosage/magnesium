package main

import (
	"github.com/gofiber/fiber/v2"
	"go_/database"
	"go_/handlers"
)

func main() {
	app := fiber.New()
	database.InitDatabase()
	handlers.InitHandlers(app)
	app.Listen(":23333")
}
