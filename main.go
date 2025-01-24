package main

import (
	"Boxed/database"
	"Boxed/internal/config"
	"Boxed/internal/routers"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/gorm"
	"log"
)

func main() {
	server, err := InitializeServer()
	if err != nil {
		log.Fatal(err)
	}
	server.JanitorService.StartCleanCycle()

	cfg, db, err := bootstrap()
	defer database.CloseDatabase(db)
	app := fiber.New(fiber.Config{
		BodyLimit:   cfg.Server.RequestConfig.SizeLimit * 1024 * 1024,
		Concurrency: cfg.Server.Concurrency * 1024,
		AppName:     "Boxed",
	})

	app.Use(logger.New())
	routers.SetupRoutes(app, server)

	err = app.Listen(fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func bootstrap() (
	*config.Configuration,
	*gorm.DB,
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
	return cfg, db, err
}
