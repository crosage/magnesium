package handlers

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go_/database"
)

var ErrResponseBodyEmpty = errors.New("response body is empty or not a map")

type SearchRequest struct {
	Tags             []string `json:"tags"`
	Page             int      `json:"page"`
	PageSize         int      `json:"size"`
	SortBy           string   `json:"sort_by"`
	SortOrder        string   `json:"sort_order"`
	Author           string   `json:"author"`
	MinBookmarkCount *int     `json:"min_bookmark_count,omitempty"`
	MaxBookmarkCount *int     `json:"max_bookmark_count,omitempty"`
	IsBookmarked     *bool    `json:"is_bookmarked,omitempty"`
}

func searchImages(ctx *fiber.Ctx) error {
	var req SearchRequest

	if err := ctx.BodyParser(&req); err != nil {
		rawBody := ctx.Body() // 读取原始 Body
		log.Printf("Received Raw Body: %s", string(rawBody))
		log.Error().Err(err).Msg("解析报文出现错误")
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

	if req.SortBy == "pid" || req.SortBy == "" {
		req.SortBy = "i.pid"
	}

	if req.SortOrder == "" {
		req.SortOrder = "DESC"
	}

	var count int
	images, count, err := database.SearchImages(req.Tags, req.Page, req.PageSize, req.Author, req.SortBy, req.SortOrder, req.MinBookmarkCount, req.MaxBookmarkCount, req.IsBookmarked)
	if err != nil {
		log.Error().Err(err)
		return sendCommonResponse(ctx, 500, "查询图片出现错误", nil)
	}
	return sendCommonResponse(ctx, 200, "成功", map[string]interface{}{
		"images": images,
		"total":  count,
	})
}
