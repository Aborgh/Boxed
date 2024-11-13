package routers

import (
	"Boxed/internal/handlers"
	"Boxed/internal/repository"
	"Boxed/internal/services"
	"github.com/gofiber/fiber/v2"
)

func SetupBoxRouter(app *fiber.App, boxRepository repository.BoxRepository) {
	boxService := services.NewBoxService(boxRepository)
	boxHandler := handlers.NewBoxHandler(boxService)

	app.Get("/boxes", boxHandler.ListBoxes)
	app.Post("/boxes", boxHandler.CreateBox)
	app.Get("/boxes/:id", boxHandler.GetBoxByID)
	app.Put("/boxes/:id", boxHandler.UpdateBox)
	app.Delete("/boxes/:id", boxHandler.DeleteBox)
}
