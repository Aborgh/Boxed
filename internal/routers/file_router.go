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
	app.Patch("/update/:box/*", fileHandler.UpdateItem)
	app.Get("/download/:box/*", fileHandler.DownloadFile)
	app.Get("/list/:box/*", fileHandler.ListFileOrFolder)
	app.Delete("/delete/:box/*", fileHandler.DeleteFile)
	app.Post("/copy/:box/*", fileHandler.CopyOrMoveItem)
}
