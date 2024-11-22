package routers

import (
	"Boxed/internal/cmd/janitor"
	"Boxed/internal/config"
	"Boxed/internal/services"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(
	app *fiber.App,
	itemService services.ItemService,
	boxService services.BoxService,
	logService services.LogService,
	janitor *janitor.Janitor,
	cfg *config.Configuration,
) {
	SetupItemRouter(app, itemService)
	SetupBoxRouter(app, boxService)
	SetupUploadRouter(app, itemService, boxService, logService, cfg)
	SetupJanitorRouter(app, janitor)
}
