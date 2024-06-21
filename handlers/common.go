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
	app.Get("/api/image", getImages)
	app.Get("/api/image/:pid", getImageByPid)

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
