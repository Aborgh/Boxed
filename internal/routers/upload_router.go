package routers

import (
	"Boxed/cmd"
	"github.com/gofiber/fiber/v2"
)

func SetupUploadRouter(
	app *fiber.App,
	server *cmd.Server,
) {
	fileHandler := server.FileHandler
	app.Post("/upload/:box/*", fileHandler.UploadFile)
	app.Get("/download/:box/*", fileHandler.DownloadFile)
	app.Get("/:box/*", fileHandler.ListFileOrFolder)
}
