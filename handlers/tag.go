package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go_/database"
)

func getTagsByPid() {

}
func getTagsWithPagination(ctx *fiber.Ctx) error {
	var req SearchRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot parse JSON",
		})
	}

	if req.Page <= 0 {
		req.Page = 1
	}

	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	tags, err := database.GetTags(req.Page, req.PageSize)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	return sendCommonResponse(ctx, 200, "成功", map[string]interface{}{
		"tags":  tags,
		"total": len(tags),
	})
}

func getTagsWithCount(ctx *fiber.Ctx) error {
	tags, err := database.GetTagCounts()
	if err != nil {
		log.Error().Msg(err.Error())
		return sendCommonResponse(ctx, 500, "错误", nil)
	}
	return sendCommonResponse(ctx, 200, "成功", map[string]interface{}{
		"tags":  tags,
		"total": len(tags),
	})
}
