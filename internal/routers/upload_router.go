package routers

import (
	"Boxed/internal/config"
	"Boxed/internal/handlers"
	"Boxed/internal/services"
	"github.com/gofiber/fiber/v2"
)

func SetupUploadRouter(
	app *fiber.App,
	itemService services.ItemService,
	boxService services.BoxService,
	logService services.LogService,
	configuration *config.Configuration,
) {
	fileService := services.NewFileService(itemService, boxService, logService, configuration)
	fileHandler := handlers.NewFileHandler(fileService)
	app.Post("/upload/:box/*", fileHandler.UploadFile)
	app.Get("/download/:box/*", fileHandler.DownloadFile)
	app.Get("/:box/*", fileHandler.ListFileOrFolder)
}
