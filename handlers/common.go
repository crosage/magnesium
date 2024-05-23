package handlers

import (
	"github.com/gofiber/fiber/v2"
	"go_/structs"
)

func InitHandlers(app *fiber.App) {

}
func sendCommonResponse(ctx *fiber.Ctx, code int, message string, data map[string]interface{}) error {
	response := structs.HoshinoResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
	json, err := jsoniter.Marshal(response)
	if err != nil {
		// THIS SHOULD NOT HAPPEN
		// If this happens, just stop the server and wait for further investigation
		utils.HoshinoLogger.Fatal().Err(err).Send()
	}
	return ctx.Status(code).Send(json)
}
