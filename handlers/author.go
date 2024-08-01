package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go_/database"
)

func getAuthorsWithCount(ctx *fiber.Ctx) error {
	authors, err := database.GetAuthorImageCounts()
	if err != nil {
		log.Error().Msg(err.Error())
		return sendCommonResponse(ctx, 500, "错误", nil)
	}
	return sendCommonResponse(ctx, 200, "成功", map[string]interface{}{
		"tags":  authors,
		"total": len(authors),
	})
}
