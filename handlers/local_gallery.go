package handlers

import (
	"github.com/gofiber/fiber/v2"
	"go_/database"
)

func getAllGalleries(ctx *fiber.Ctx) error {
	rows, err := database.GetAllGalleries()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON()
	}
}
