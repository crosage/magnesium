package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"go_/structs"
)

func InitHandlers(app *fiber.App) {
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET, POST, PUT, DELETE",
	}))
	app.Post("/api/gallery", createGallery)
	app.Get("/api/gallery", getAllGalleries)
	app.Delete("/api/gallery", deleteGallery)
	app.Get("/api/gallery/:id/init", initGallery)
	app.Get("/api/pixiv/cookie", getPixivCookie)
	app.Post("/api/pixiv/cookie", updatePixivCookie)
	app.Get("/api/pixiv/image/update", triggerUpdateAllHandler)
	app.Get("/api/pixiv/image/checker", triggerUpdateAllHandlerChecker)
	app.Get("/api/pixiv/image/:pid", getImageByPid)
	app.Post("/api/pixiv/image/following", postFollowLatestIllustsHandler)
	app.Post("/api/pixiv/usr/following", postFollowingUsersHandler)
	app.Post("/api/image", searchImages)
	app.Post("/api/tag", getTagsWithPagination)
	app.Get("/api/tag/tag-statistics", getTagsWithCount)
	app.Get("/api/author/author-statistics", getAuthorsWithCount)

}
func sendCommonResponse(ctx *fiber.Ctx, code int, message string, data map[string]interface{}) error {
	response := structs.Response{
		Code: code,
		Msg:  message,
		Data: data,
	}
	json, err := jsoniter.Marshal(response)
	if err != nil {
		// THIS SHOULD NOT HAPPEN
		// If this happens, just stop the server and wait for further investigation
		log.Error().Err(err).Msg("发送报文出现错误")
	}
	return ctx.Status(code).Send(json)
}
