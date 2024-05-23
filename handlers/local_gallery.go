package handlers

import (
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"go_/database"
	"go_/structs"
	"strconv"
)

func getAllGalleries(ctx *fiber.Ctx) error {
	var galleries []structs.LocalGallery
	galleries, err := database.GetAllGalleries()
	if err != nil {
		return sendCommonResponse(ctx, 500, "", nil)
	}
	return sendCommonResponse(ctx, 200, "成功", map[string]interface{}{
		"total":     len(galleries),
		"galleries": galleries,
	})
}

func createGallery(ctx *fiber.Ctx) error {
	var gallery structs.LocalGallery
	err := jsoniter.Unmarshal(ctx.Body(), &gallery)
	if err != nil {
		return sendCommonResponse(ctx, 500, "", nil)
	}
	err = database.CreateLocalGalleryPath(gallery.Path)
	if err != nil {
		return sendCommonResponse(ctx, 500, "", nil)
	}
	return sendCommonResponse(ctx, 200, "成功", nil)
}

func deleteGallery(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	id, _ := strconv.Atoi(idParam)
	err := database.DeleteLocalGalleryByID(id)
	if err != nil {
		return sendCommonResponse(ctx, 500, "删除失败", nil)
	}
	return sendCommonResponse(ctx, 200, "删除成功", nil)
}
