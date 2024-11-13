package routers

import (
	"Boxed/internal/config"
	"Boxed/internal/repository"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB, cfg *config.Configuration) {
	itemRepository := repository.NewItemRepository(db)
	boxRepository := repository.NewBoxRepository(db)
	SetupItemRouter(app, itemRepository)
	SetupBoxRouter(app, boxRepository)
	SetupUploadRouter(app, itemRepository, boxRepository, cfg)
}
