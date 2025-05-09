package routers

import (
	"Boxed/cmd"
	"github.com/gofiber/fiber/v2"
)

func SetupItemRouter(app *fiber.App, server *cmd.Server) {
	itemHandler := server.ItemHandler
	app.Get("/items", itemHandler.ListItems)
	app.Get("/items/deleted", itemHandler.ListDeletedItems)
	app.Get("/items/search", itemHandler.ItemsSearch)

	// TODO: Not implemented yet, implement ASAP
	//app.Post("/items/copy", itemHandler.ItemCopy)
	//app.Post("/items/move", itemHandler.ItemMove)
}
