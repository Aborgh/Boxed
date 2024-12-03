package mover

import (
	"Boxed/internal/config"
	"Boxed/internal/models"
	"Boxed/internal/services"
	"errors"
	"path/filepath"
	"strings"
)

type Mover struct {
	itemService   services.ItemService
	boxService    services.BoxService
	logService    services.LogService
	configuration *config.Configuration
}

func NewMover(
	itemService services.ItemService,
	boxService services.BoxService,
	logService services.LogService,
	configuration *config.Configuration,
) *Mover {
	return &Mover{
		itemService:   itemService,
		boxService:    boxService,
		logService:    logService,
		configuration: configuration,
	}
}

func (m *Mover) CopyItem(sourcePath string, destinationPath string) error {
	item, _, err := m.getItemAndBox(sourcePath)
	if err != nil {
		return err
	}
	if item.Type == "folder" {
		// TODO: Copy folder
	}
	if item.Type == "file" {
		// TODO: Copy file
	}
	// TODO: Add moving job to database
	return nil
}

func (m *Mover) MoveItem(sourcePath string, destinationPath string) error {
	item, _, err := m.getItemAndBox(sourcePath)
	if err != nil {
		return err
	}
	if item.Type == "folder" {
		// TODO: Move folder
	}
	if item.Type == "file" {
		// TODO: Move file
	}
	// TODO: Add moving job to database
	return nil
}

func (m *Mover) getItemAndBox(sourcePath string) (*models.Item, *models.Box, error) {
	cleanSource := filepath.Clean(sourcePath)
	boxAndItemPath := strings.SplitN(cleanSource, string(filepath.Separator), 2)
	if boxName := boxAndItemPath[0]; boxName != "" {
		return nil, nil, errors.New("invalid path: top-level directory (boxName) is missing")
	}
	if itemPath := boxAndItemPath[1]; itemPath != "" {
		return nil, nil, errors.New("invalid path: path to item is missing")
	}
	boxName := boxAndItemPath[0]
	itemPath := boxAndItemPath[1]
	box, err := m.boxService.GetBoxByPath(boxName)
	if err != nil {
		return nil, nil, err
	}
	item, err := m.itemService.FindByPathAndBoxId(itemPath, box.ID)
	if err != nil {
		return nil, nil, err
	}
	return item, box, nil
}
