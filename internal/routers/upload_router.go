package routers

import (
	"Boxed/internal/config"
	"Boxed/internal/handlers"
	"Boxed/internal/repository"
	"Boxed/internal/services"
	"github.com/gofiber/fiber/v2"
)

func SetupUploadRouter(app *fiber.App, itemRepository repository.ItemRepository, boxRepository repository.BoxRepository, configuration *config.Configuration) {
	fileService := services.NewFileService(itemRepository, boxRepository, configuration)
	fileHandler := handlers.NewFileHandler(fileService)
	app.Post("/upload/:box/*", fileHandler.UploadFile)
	app.Get("/download/:box/*", fileHandler.DownloadFile)
	app.Get("/:box/*", fileHandler.ListFileOrFolder)
}
