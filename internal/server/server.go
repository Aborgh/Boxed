package server

import (
	"Boxed/cmd"
	"Boxed/internal/config"
	"Boxed/internal/helpers/janitor"
	"Boxed/internal/routers"
	"Boxed/internal/services"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func NewApp(server *cmd.Server, itemService services.ItemService, boxService services.BoxService, logService services.LogService, janitor *janitor.Janitor, cfg *config.Configuration) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit:   cfg.Server.RequestConfig.SizeLimit * 1024 * 1024,
		Concurrency: cfg.Server.Concurrency * 1024,
	})

	app.Use(logger.New())

	routers.SetupItemRouter(app, server)
	routers.SetupBoxRouter(app, server)
	routers.SetupUploadRouter(app, itemService, boxService, logService, cfg)
	routers.SetupJanitorRouter(app, janitor)

	err := app.Listen(fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	return app
}
