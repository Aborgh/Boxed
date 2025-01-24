package cmd

import (
	"Boxed/internal/handlers"
	"Boxed/internal/services"
)

type Server struct {
	BoxService     services.BoxService
	BoxHandler     *handlers.BoxHandler
	ItemService    services.ItemService
	ItemHandler    *handlers.ItemHandler
	FileService    services.FileService
	FileHandler    *handlers.FileHandler
	LogService     services.LogService
	JanitorService *services.Janitor
}

func NewServer(
	boxService services.BoxService,
	boxHandler *handlers.BoxHandler,
	itemService services.ItemService,
	itemHandler *handlers.ItemHandler,
	fileService services.FileService,
	fileHandler *handlers.FileHandler,
	logService services.LogService,
	janitorService *services.Janitor,

) *Server {
	return &Server{
		BoxService:     boxService,
		BoxHandler:     boxHandler,
		ItemService:    itemService,
		ItemHandler:    itemHandler,
		FileService:    fileService,
		FileHandler:    fileHandler,
		LogService:     logService,
		JanitorService: janitorService,
	}
}
