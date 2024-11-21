package routers

import (
	"Boxed/internal/handlers"
	"Boxed/internal/repository"
	"Boxed/internal/services"
	"github.com/gofiber/fiber/v2"
)

func SetupItemRouter(app *fiber.App, itemRepository repository.ItemRepository) {
	itemService := services.NewItemService(itemRepository)
	itemHandler := handlers.NewItemHandler(itemService)
	app.Get("/items/deleted", itemHandler.ListDeletedItems)
	app.Get("/items", itemHandler.ListItems)
	app.Post("/items", itemHandler.CreateItem)
	app.Get("/items/:id", itemHandler.GetItemByID)
	app.Put("/items/:id", itemHandler.UpdateItem)
	app.Delete("/items/:id", itemHandler.DeleteItem)
}
