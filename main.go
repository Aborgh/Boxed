package main

import (
	"Boxed/database"
	"Boxed/internal/cmd"
	"Boxed/internal/config"
	"Boxed/internal/repository"
	"Boxed/internal/routers"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"log"
)

func main() {
	db, err := database.SetupDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer database.CloseDatabase(db)
	cfg, err := config.LoadConfiguration("boxed.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	app := fiber.New(fiber.Config{
		BodyLimit:   cfg.Server.RequestConfig.SizeLimit * 1024 * 1024,
		Concurrency: cfg.Server.Concurrency * 1024,
	})
	itemRepository := repository.NewItemRepository(db)

	janitor := cmd.NewJanitor(itemRepository, cfg)
	janitor.StartClean()
	app.Use(logger.New())

	routers.SetupRoutes(app, db, cfg)
	err = app.Listen(fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
