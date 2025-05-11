package services

import (
	"Boxed/internal/config"
	"Boxed/internal/dto"
	"Boxed/internal/helpers"
	"Boxed/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MoverService interface {
	CopyItem(sourcePath string, destinationPath string, properties json.RawMessage) (*dto.ItemGetDTO, error)
	MoveItem(sourcePath string, destinationPath string, properties json.RawMessage) (*dto.ItemGetDTO, error)
}

type MoverServiceImpl struct {
	itemService   ItemService
	boxService    BoxService
	fileService   FileService
	configuration *config.Configuration
	logService    LogService
}

func NewMoverService(
	itemService ItemService,
	boxService BoxService,
	fileService FileService,
	configuration *config.Configuration,
	logService LogService,
) MoverService {
	return &MoverServiceImpl{
		itemService:   itemService,
		boxService:    boxService,
		fileService:   fileService,
		configuration: configuration,
		logService:    logService,
	}
}

func (m *MoverServiceImpl) CopyItem(sourcePath string, destinationPath string, properties json.RawMessage) (*dto.ItemGetDTO, error) {
	sourceBox, sourceItem, err := m.getItemAndBox(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("error with source path: %w", err)
	}

	destBox, destParentPath, destItemName, err := m.parseDestinationPath(destinationPath)
	if err != nil {
		return nil, fmt.Errorf("error with destination path: %w", err)
	}

	destItemPath := destItemName
	if destParentPath != "" {
		destItemPath = filepath.Join(destParentPath, destItemName)
	}

	existingItem, err := m.itemService.FindByPathAndBoxId(destItemPath, destBox.ID)
	if err == nil && existingItem != nil {
		return nil, fmt.Errorf("destination item already exists: %s", destItemPath)
	}

	if sourceItem.Type == "file" {
		var parentID *uint
		if destParentPath != "" {
			parentItem, err := m.ensureFolderPathExists(destBox, destParentPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create parent folders: %w", err)
			}
			parentID = &parentItem.ID
		}

		newItem := &models.Item{
			Name:       destItemName,
			Type:       "file",
			Extension:  sourceItem.Extension,
			BoxID:      destBox.ID,
			ParentID:   parentID,
			Path:       destItemPath,
			Size:       sourceItem.Size,
			SHA256:     sourceItem.SHA256,
			SHA512:     sourceItem.SHA512,
			Properties: properties,
		}

		err = m.itemService.Create(newItem)
		if err != nil {
			return nil, fmt.Errorf("failed to create new item record: %w", err)
		}

		err = m.ensureFileExistsInDestination(sourceItem, sourceBox, destBox)
		if err != nil {
			deleteErr := m.itemService.DeleteItem(newItem.ID, true)
			if deleteErr != nil {
				return nil, fmt.Errorf("failed to delete new item record: %w", err)
			}
			return nil, fmt.Errorf("failed to copy file content: %w", err)
		}

		itemDTO, err := m.itemService.GetItemByID(newItem.ID)
		if err != nil {
			return nil, fmt.Errorf("created item but failed to retrieve DTO: %w", err)
		}
		return itemDTO, nil
	}

	if sourceItem.Type == "folder" {
		destFolder, err := m.createFolderAtDestination(destBox, destParentPath, destItemName)
		if err != nil {
			return nil, fmt.Errorf("failed to create destination folder: %w", err)
		}

		children, err := m.itemService.FindItemsByParentID(&sourceItem.ID, sourceBox.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get folder contents: %w", err)
		}

		for _, child := range children {
			childSourcePath := filepath.Join(sourcePath, child.Name)
			childDestPath := filepath.Join(destinationPath, child.Name)
			_, err := m.CopyItem(childSourcePath, childDestPath, properties)
			if err != nil {
				m.logService.Log.Warnf("Failed to copy child item %s: %v", child.Name, err)
			}
		}

		folderDTO, err := m.itemService.GetItemByID(destFolder.ID)
		if err != nil {
			return nil, fmt.Errorf("created folder but failed to retrieve DTO: %w", err)
		}
		return folderDTO, nil
	}

	return nil, fmt.Errorf("unsupported item type: %s", sourceItem.Type)
}

func (m *MoverServiceImpl) MoveItem(sourcePath string, destinationPath string, properties json.RawMessage) (*dto.ItemGetDTO, error) {
	// TODO: The ability to move entire folders. And delete source folders on move
	sourceBox, sourceItem, err := m.getItemAndBox(sourcePath)
	if sourceItem == nil {
		return nil, fmt.Errorf("source item not found: %s", sourcePath)
	}
	if err != nil {
		return nil, fmt.Errorf("error with source path: %w", err)
	}

	destBox, destParentPath, destItemName, err := m.parseDestinationPath(destinationPath)
	if err != nil {
		return nil, fmt.Errorf("error with destination path: %w", err)
	}

	destItemPath := destItemName
	if destParentPath != "" {
		destItemPath = filepath.Join(destParentPath, destItemName)
	}

	existingItem, err := m.itemService.FindByPathAndBoxId(destItemPath, destBox.ID)
	if err == nil && existingItem != nil {
		return nil, fmt.Errorf("destination item already exists: %s", destItemPath)
	}

	if sourceBox.ID == destBox.ID {
		if sourceItem.Type == "file" {
			var parentID *uint
			if destParentPath != "" {
				parentItem, err := m.ensureFolderPathExists(destBox, destParentPath)
				if err != nil {
					return nil, fmt.Errorf("failed to create parent folders: %w", err)
				}
				parentID = &parentItem.ID
			}

			sourceItem.Name = destItemName
			sourceItem.Path = destItemPath
			sourceItem.ParentID = parentID

			err = m.itemService.UpdateItem(sourceItem)
			if err != nil {
				return nil, fmt.Errorf("failed to update item path: %w", err)
			}

			return m.itemService.GetItemByID(sourceItem.ID)
		}

		if sourceItem.Type == "folder" {
			var parentID *uint
			if destParentPath != "" {
				parentItem, err := m.ensureFolderPathExists(destBox, destParentPath)
				if err != nil {
					return nil, fmt.Errorf("failed to create parent folders: %w", err)
				}
				parentID = &parentItem.ID
			}

			sourceItem.Name = destItemName
			sourceItem.Path = destItemPath
			sourceItem.ParentID = parentID

			err = m.itemService.UpdateItem(sourceItem)
			if err != nil {
				return nil, fmt.Errorf("failed to update folder path: %w", err)
			}

			// TODO: Update paths of all children
			// This would require implementing a method to update all child paths
			// For now, we'll use copy and delete for folders

			return m.itemService.GetItemByID(sourceItem.ID)
		}
	}
	// If properties are not set, clone the source properties
	if properties == nil {
		properties = sourceItem.Properties
	}
	itemDTO, err := m.CopyItem(sourcePath, destinationPath, properties)
	if err != nil {
		return nil, fmt.Errorf("failed to copy during move operation: %w", err)
	}

	// 2. Delete the original
	err = m.fileService.DeleteItemOnDisk(*sourceItem, sourceBox)
	if err != nil {
		m.logService.Log.Warnf("Moved item but failed to delete source: %v", err)
		// We still return success since the item was copied successfully
	}

	return itemDTO, nil
}

func (m *MoverServiceImpl) getItemAndBox(sourcePath string) (*models.Box, *models.Item, error) {
	cleanSource := filepath.Clean(sourcePath)
	boxAndItemPath := strings.SplitN(cleanSource, string(filepath.Separator), 2)

	if len(boxAndItemPath) < 2 {
		return nil, nil, errors.New("invalid path format: must be 'boxname/path/to/item'")
	}

	boxName := boxAndItemPath[0]
	itemPath := boxAndItemPath[1]

	box, err := m.boxService.GetBoxByPath(boxName)
	if err != nil {
		return nil, nil, fmt.Errorf("box not found: %s", boxName)
	}

	item, err := m.itemService.FindByPathAndBoxId(itemPath, box.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("item not found: %s", itemPath)
	}

	return box, item, nil
}

func (m *MoverServiceImpl) parseDestinationPath(path string) (*models.Box, string, string, error) {
	cleanPath := filepath.Clean(path)
	boxAndItemPath := strings.SplitN(cleanPath, string(filepath.Separator), 2)

	if len(boxAndItemPath) < 2 {
		return nil, "", "", errors.New("invalid path format: must be 'boxname/path/to/item'")
	}

	boxName := boxAndItemPath[0]
	fullItemPath := boxAndItemPath[1]

	box, err := m.boxService.GetBoxByPath(boxName)
	if err != nil {
		return nil, "", "", fmt.Errorf("destination box not found: %s", boxName)
	}

	lastSlashIndex := strings.LastIndex(fullItemPath, "/")
	var parentPath, itemName string

	if lastSlashIndex == -1 {
		parentPath = ""
		itemName = fullItemPath
	} else {
		parentPath = fullItemPath[:lastSlashIndex]
		itemName = fullItemPath[lastSlashIndex+1:]
	}

	return box, parentPath, itemName, nil
}

func (m *MoverServiceImpl) ensureFolderPathExists(box *models.Box, path string) (*models.Item, error) {
	if path == "" {
		return nil, nil
	}

	parts := strings.Split(path, "/")
	var currentPath string
	var parentID *uint
	var currentFolder *models.Item

	for _, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = filepath.Join(currentPath, part)
		}

		folder, err := m.itemService.FindByPathAndBoxId(currentPath, box.ID)
		if err == nil && folder != nil && folder.Type == "folder" {
			currentFolder = folder
			parentID = &folder.ID
			continue
		}

		newFolder := &models.Item{
			Name:     part,
			Type:     "folder",
			BoxID:    box.ID,
			ParentID: parentID,
			Path:     currentPath,
		}

		if err := m.itemService.Create(newFolder); err != nil {
			return nil, fmt.Errorf("failed to create folder '%s': %w", part, err)
		}

		currentFolder = newFolder
		parentID = &newFolder.ID
	}

	return currentFolder, nil
}

func (m *MoverServiceImpl) createFolderAtDestination(box *models.Box, parentPath, folderName string) (*models.Item, error) {
	var parentID *uint
	var folderPath string

	if parentPath != "" {
		parentFolder, err := m.ensureFolderPathExists(box, parentPath)
		if err != nil {
			return nil, err
		}

		if parentFolder != nil {
			parentID = &parentFolder.ID
		}

		folderPath = filepath.Join(parentPath, folderName)
	} else {
		folderPath = folderName
	}

	existingFolder, err := m.itemService.FindByPathAndBoxId(folderPath, box.ID)
	if err == nil && existingFolder != nil {
		return existingFolder, nil
	}

	newFolder := &models.Item{
		Name:     folderName,
		Type:     "folder",
		BoxID:    box.ID,
		ParentID: parentID,
		Path:     folderPath,
	}

	if err := m.itemService.Create(newFolder); err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	return newFolder, nil
}

func (m *MoverServiceImpl) ensureFileExistsInDestination(sourceItem *models.Item, sourceBox, destBox *models.Box) error {
	hash := sourceItem.SHA256
	if hash == "" {
		return errors.New("source file has no SHA256 hash")
	}

	firstHashPrefix := hash[:2]
	secondHashPrefix := hash[2:4]

	sourcePath := filepath.Join(sourceBox.Path, firstHashPrefix, secondHashPrefix, hash)
	destPath := filepath.Join(destBox.Path, firstHashPrefix, secondHashPrefix, hash)

	if _, err := os.Stat(destPath); err == nil {
		return nil
	}

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return helpers.CopyFile(sourcePath, destPath)
}
