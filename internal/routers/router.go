package routers

import (
	"Boxed/cmd"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(
	app *fiber.App,
	server *cmd.Server,
) {
	SetupItemRouter(app, server)
	SetupBoxRouter(app, server)
	SetupUploadRouter(app, server)
	SetupJanitorRouter(app, server)
}
