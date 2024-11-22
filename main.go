package main

import (
	"Boxed/database"
	"Boxed/internal/cmd/janitor"
	"Boxed/internal/config"
	"Boxed/internal/repository"
	"Boxed/internal/routers"
	"Boxed/internal/services"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/gorm"
	"log"
)

func main() {

	cfg, itemService, boxService, logService, db, janitorCleaner, err := bootstrap()
	defer database.CloseDatabase(db)
	app := fiber.New(fiber.Config{
		BodyLimit:   cfg.Server.RequestConfig.SizeLimit * 1024 * 1024,
		Concurrency: cfg.Server.Concurrency * 1024,
	})

	app.Use(logger.New())
	routers.SetupRoutes(app, itemService, boxService, logService, janitorCleaner, cfg)

	err = app.Listen(fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func bootstrap() (
	*config.Configuration,
	services.ItemService,
	services.BoxService,
	services.LogService,
	*gorm.DB,
	*janitor.Janitor,
	error,
) {
	cfg, err := config.LoadConfiguration("boxed.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	db, err := database.SetupDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	itemRepository := repository.NewItemRepository(db)
	itemService := services.NewItemService(itemRepository)

	boxRepository := repository.NewBoxRepository(db)
	boxService := services.NewBoxService(boxRepository)
	logService := services.NewLogService(cfg)
	fileService := services.NewFileService(itemService, boxService, logService, cfg)
	cleaner := janitor.NewJanitor(itemService, boxService, fileService, logService, cfg)
	cleaner.StartCleanCycle()
	return cfg, itemService, boxService, logService, db, cleaner, err
}
