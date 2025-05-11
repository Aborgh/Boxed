package services

import (
	"Boxed/internal/config"
	"Boxed/internal/helpers"
	"Boxed/internal/models"
	"fmt"
	"path/filepath"
	"strings"
)

type MoverService interface {
	CopyItem(sourcePath string, destinationPath string) error
	MoveItem(sourcePath string, destinationPath string) error
}

type MoverServiceImpl struct {
	itemService   ItemService
	boxService    BoxService
	configuration *config.Configuration
	logService    LogService
}

func NewMoverService(
	itemService ItemService,
	boxService BoxService,
	configuration *config.Configuration,
	logService LogService,
) MoverService {
	return &MoverServiceImpl{
		itemService:   itemService,
		boxService:    boxService,
		configuration: configuration,
		logService:    logService,
	}
}

func (m *MoverServiceImpl) CopyItem(sourcePath string, destinationPath string) error {
	sourceItem, sourceBox, err := m.getItemAndBox(sourcePath)
	if err != nil {
		return err
	}
	destinationItem, destinationBox, err := m.getItemAndBox(destinationPath)
	if err != nil {
		return err
	}

	if destinationItem != nil {
		return fmt.Errorf("destination item already exists")
	}
	if destinationBox == nil {
		return fmt.Errorf("destination box not found")
	}
	firstHashPrefix := sourceItem.SHA256[:2]
	secondHashPrefix := sourceItem.SHA256[2:4]
	err = helpers.CopyFile(filepath.Join(sourceBox.Path, firstHashPrefix, secondHashPrefix, sourceItem.SHA256), filepath.Join(destinationBox.Path, firstHashPrefix, secondHashPrefix, sourceItem.SHA256))
	if err != nil {
		return err
	}
	newFile := &models.Item{
		Name:       sourceItem.Name,
		Type:       "file",
		Extension:  sourceItem.Extension,
		BoxID:      destinationBox.ID,
		ParentID:   sourceItem.ParentID,
		Path:       destinationPath,
		Size:       sourceItem.Size,
		SHA256:     sourceItem.SHA256,
		SHA512:     sourceItem.SHA512,
		Properties: sourceItem.Properties,
	}
	return m.itemService.Create(newFile)
}

func (m *MoverServiceImpl) MoveItem(sourcePath string, destinationPath string) error {
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

func (m *MoverServiceImpl) getItemAndBox(sourcePath string) (*models.Item, *models.Box, error) {
	cleanSource := filepath.Clean(sourcePath)
	boxAndItemPath := strings.SplitN(cleanSource, string(filepath.Separator), 2)
	//if boxName := boxAndItemPath[0]; boxName != "" {
	//	return nil, nil, errors.New("invalid path: top-level directory (boxName) is missing")
	//}
	//if itemPath := boxAndItemPath[1]; itemPath != "" {
	//	return nil, nil, errors.New("invalid path: path to item is missing")
	//}
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
