//go:build wireinject
// +build wireinject

package main

import (
	"Boxed/cmd"
	"Boxed/database"
	"Boxed/internal/config"
	"Boxed/internal/handlers"
	"Boxed/internal/repository"
	"Boxed/internal/services"
	"github.com/google/wire"
)

func Provider() (*config.Configuration, error) {
	return config.LoadConfiguration("boxed.yaml")
}

func InitializeServer() (*cmd.Server, error) {
	wire.Build(
		cmd.NewServer,
		services.NewBoxService,
		handlers.NewBoxHandler,
		repository.NewBoxRepository,
		services.NewItemService,
		handlers.NewItemHandler,
		repository.NewItemRepository,
		database.SetupDatabase,
		services.NewFileService,
		handlers.NewFileHandler,
		services.NewLogService,
		services.NewJanitorService,
		services.NewMoverService,
		Provider,
	)
	return nil, nil
}
