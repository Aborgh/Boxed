package routers

import (
	"Boxed/cmd"
	"github.com/gofiber/fiber/v2"
)

func SetupBoxRouter(app *fiber.App, server *cmd.Server) {
	boxHandler := server.BoxHandler
	app.Get("/boxes", boxHandler.ListBoxes)
	app.Post("/boxes", boxHandler.CreateBox)
	app.Get("/boxes/:id", boxHandler.GetBoxByID)
	app.Put("/boxes/:id", boxHandler.UpdateBox)
	app.Delete("/boxes/:id", boxHandler.DeleteBox)
}
