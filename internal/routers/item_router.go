package routers

import (
	"Boxed/internal/handlers"
	"Boxed/internal/services"
	"github.com/gofiber/fiber/v2"
)

func SetupItemRouter(app *fiber.App, itemService services.ItemService) {
	itemHandler := handlers.NewItemHandler(itemService)
	app.Get("/items", itemHandler.ListItems)
	app.Post("/items", itemHandler.CreateItem)
	app.Get("/items/deleted", itemHandler.ListDeletedItems)
	app.Get("/items/search", itemHandler.ItemsSearch)
	app.Get("/items/:id", itemHandler.GetItemByID)
	app.Put("/items/:id", itemHandler.UpdateItem)
	app.Delete("/items/:id", itemHandler.DeleteItem)
	app.Get("/items/:id/tree", itemHandler.GetItemTree)
	app.Post("/items/copy", itemHandler.ItemCopy)
	app.Post("/items/move", itemHandler.ItemMove)
}
