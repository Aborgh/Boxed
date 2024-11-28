package services

import (
	"Boxed/internal/cmd"
	"Boxed/internal/config"
	"Boxed/internal/models"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

type FileService interface {
	CreateFileStructure(box *models.Box, filePath string, fileHeader *multipart.FileHeader, flat bool, properties string) (*models.Item, error)
	FindBoxByPath(boxPath string) (*models.Box, error)
	ListFileOrFolder(boxName string, itemPath string) (*models.Item, error)
	GetFileItem(box *models.Box, filePath string) (*models.Item, error)
	GetStoragePath() string
	DeleteItemOnDisk(item models.Item, box *models.Box) error
}

type FileServiceImpl struct {
	itemService   ItemService
	boxService    BoxService
	logService    LogService
	configuration config.Configuration
}

func NewFileService(
	itemService ItemService,
	boxService BoxService,
	logService LogService,
	configuration *config.Configuration,
) FileService {

	return &FileServiceImpl{
		itemService:   itemService,
		boxService:    boxService,
		logService:    logService,
		configuration: *configuration,
	}
}

func (s *FileServiceImpl) CreateFileStructure(
	box *models.Box,
	filePath string,
	fileHeader *multipart.FileHeader,
	flat bool,
	properties string,
) (*models.Item, error) {
	pathParts := strings.Split(filePath, "/")

	// Parse properties
	propertiesMap := make(map[string][]string)
	if properties != "" {
		keyValueProperties := strings.Split(properties, ";")
		for _, keyValueProperty := range keyValueProperties {
			keyAndValue := strings.SplitN(keyValueProperty, "=", 2)
			if len(keyAndValue) != 2 {
				continue // Skip invalid key-value pairs
			}
			key := strings.TrimSpace(keyAndValue[0])
			value := strings.TrimSpace(keyAndValue[1])
			propertiesMap[key] = append(propertiesMap[key], value)
		}
	}

	jsonProperties, err := json.Marshal(propertiesMap)
	if err != nil {
		return nil, err
	}

	var parentItem *models.Item

	if !flat {
		for _, part := range pathParts[:len(pathParts)-1] {
			folderItem, err := s.createOrGetFolderItem(part, parentItem, box)
			if err != nil {
				return nil, err
			}
			parentItem = folderItem
		}
	}

	name := pathParts[len(pathParts)-1]

	if fileHeader == nil {
		// No file provided; create a folder
		item, err := s.createOrGetFolderItem(name, parentItem, box)
		if err != nil {
			return nil, err
		}
		return item, nil
	} else {
		// File provided; create a file
		fileType := cmd.GetFileType(name)
		item, err := s.createFolderOrFileItem(name, fileType, parentItem, box, fileHeader, jsonProperties)
		if err != nil {
			return nil, err
		}
		return item, nil
	}
}

func (s *FileServiceImpl) createOrGetFolderItem(name string, parentItem *models.Item, box *models.Box) (*models.Item, error) {
	var parentID *uint
	var path string

	if parentItem != nil {
		parentID = &parentItem.ID
		path = filepath.Join(parentItem.Path, name)
	} else {
		// For top-level folders, path is just name
		path = name
	}

	// Check if the folder already exists
	existingFolder, err := s.itemService.FindFolderByNameAndParent(name, parentID, box.ID)
	if err == nil && existingFolder != nil {
		return existingFolder, nil
	}

	// Create the directory on disk
	if err := os.MkdirAll(path, 0750); err != nil {
		return nil, err
	}

	// Create the folder item
	newFolder := &models.Item{
		Name:     name,
		Type:     "folder",
		BoxID:    box.ID,
		ParentID: parentID,
		Path:     path,
	}
	if err := s.itemService.InsertItem(newFolder); err != nil {
		return nil, err
	}

	return newFolder, nil
}
func (s *FileServiceImpl) createFolderOrFileItem(
	name, fileType string,
	parentItem *models.Item,
	box *models.Box,
	fileHeader *multipart.FileHeader,
	properties []byte,
) (*models.Item, error) {
	var parentID *uint
	var dirPath, itemPath string

	// Determine the item's path and directory path
	if parentItem != nil {
		parentID = &parentItem.ID
		itemPath = filepath.Join(parentItem.Path, name)
		dirPath = filepath.Join(s.configuration.Storage.Path, box.Name, parentItem.Path)
	} else {
		itemPath = name
		dirPath = filepath.Join(s.configuration.Storage.Path, box.Name)
	}

	fullFilePath := filepath.Join(dirPath, name)

	// Ensure the directory exists on disk
	if err := os.MkdirAll(dirPath, 0750); err != nil {
		return nil, err
	}

	// Save the file and compute checksums
	sha256sum, sha512sum, err := cmd.SaveFileAndComputeChecksums(fileHeader, fullFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to save file and compute checksums: %w", err)
	}

	// Check if an item with the same path already exists
	existingItem, err := s.itemService.FindByPathAndBoxId(itemPath, box.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing item: %w", err)
	}

	if existingItem != nil {
		existingItem.Size = fileHeader.Size
		existingItem.SHA256 = sha256sum
		existingItem.SHA512 = sha512sum
		existingItem.Properties = properties

		if err := s.itemService.UpdateItem(existingItem); err != nil {
			return nil, fmt.Errorf("failed to update existing item: %w", err)
		}

		return existingItem, nil
	} else {
		newFile := &models.Item{
			Name:       name,
			Type:       "file",
			Extension:  fileType,
			BoxID:      box.ID,
			ParentID:   parentID,
			Path:       itemPath, // Store the relative path
			Size:       fileHeader.Size,
			SHA256:     sha256sum,
			SHA512:     sha512sum,
			Properties: properties,
		}

		if err := s.itemService.InsertItem(newFile); err != nil {
			return nil, fmt.Errorf("failed to create item record: %w", err)
		}

		return newFile, nil
	}
}

func (s *FileServiceImpl) FindBoxByPath(boxPath string) (*models.Box, error) {
	return s.boxService.GetBoxByPath(boxPath)
}

func (s *FileServiceImpl) ListFileOrFolder(boxName string, itemPath string) (*models.Item, error) {
	box, err := s.FindBoxByPath(boxName)
	if err != nil {
		return nil, err
	}
	if box == nil {
		return nil, fmt.Errorf("box not found")
	}

	var item *models.Item
	if itemPath == "" || itemPath == "/" {
		// Root folder
		item = &models.Item{
			Name:  box.Name,
			Type:  "folder",
			BoxID: box.ID,
			Path:  "",
		}
	} else {
		item, err = s.itemService.FindByPathAndBoxId(itemPath, box.ID)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return nil, fmt.Errorf("item not found")
		}
	}

	// Get children if the item is a folder
	if item.Type == "folder" {
		children, err := s.itemService.FindItemsByParentID(&item.ID, box.ID)
		if err != nil {
			return nil, err
		}
		item.Children = children
	}

	return item, nil
}

func (s *FileServiceImpl) GetFileItem(box *models.Box, filePath string) (*models.Item, error) {
	item, err := s.itemService.FindByPathAndBoxId(filePath, box.ID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("file not found")
	}
	return item, nil
}

func (s *FileServiceImpl) GetStoragePath() string {
	return s.configuration.Storage.Path
}

func (s *FileServiceImpl) DeleteItemOnDisk(item models.Item, box *models.Box) error {
	filePath := filepath.Join(s.configuration.Storage.Path, box.Name, item.Path)
	itemLog := s.logService.Log.WithFields(logrus.Fields{
		"name": item.Name,
		"path": item.Path,
		"job":  "clean",
	})

	// Delete item(s) from the database
	itemLog.Debug("Deleting item(s) from the database")
	err := s.itemService.HardDelete(&item)
	if err != nil {
		itemLog.WithError(err).Error("Failed to delete item(s) from the database")
		return err
	}

	// Now delete files from disk
	itemLog.Debug("Deleting item(s) from disk")
	if item.Type == "folder" {
		err = cmd.DeleteFile(filePath, true)
	} else {
		err = cmd.DeleteFile(filePath, false)
	}

	if err != nil {
		itemLog.WithError(err).Error("Failed to delete item(s) from disk")
		return err
	}

	itemLog.Info("Successfully deleted item(s) from disk and database")
	return nil
}
