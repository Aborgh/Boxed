package services

import (
	"Boxed/internal/config"
	"Boxed/internal/dto"
	"Boxed/internal/helpers"
	"Boxed/internal/mapper"
	"Boxed/internal/models"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type FileService interface {
	CreateFileStructure(box *models.Box, filePath string, fileHeader *multipart.FileHeader, flat bool, properties string) (*dto.ItemGetDTO, error)
	FindBoxByPath(boxPath string) (*models.Box, error)
	ListFileOrFolder(boxName string, itemPath string) (*models.Item, error)
	GetFileItem(box *models.Box, filePath string) (*models.Item, error)
	GetStoragePath() string
	DeleteItemOnDisk(item models.Item, box *models.Box) error
	UpdateItem(item *models.Item) (*dto.ItemGetDTO, error)
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
) (*dto.ItemGetDTO, error) {
	pathParts := strings.Split(filePath, "/")

	// Parse properties
	propertiesMap := make(map[string][]string)
	if properties != "" {
		keyValueProperties := strings.Split(properties, ";")
		for i := range keyValueProperties {
			keyAndValue := strings.SplitN(keyValueProperties[i], "=", 2)
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
			folderItem, err := s.createOrGetFolder(part, parentItem, box)
			if err != nil {
				return nil, err
			}
			parentItem = folderItem
		}
	}

	name := pathParts[len(pathParts)-1]

	if fileHeader == nil {
		// No file provided; create a folder
		item, err := s.createOrGetFolder(name, parentItem, box)
		if err != nil {
			return nil, err
		}
		return s.itemService.GetItemByID(item.ID)
	} else {
		// File provided; create a file using hash-based storage
		fileType := helpers.GetFileType(name)
		item, err := s.createHashBasedFile(name, fileType, parentItem, box, fileHeader, jsonProperties)
		if err != nil {
			return nil, err
		}
		return s.itemService.GetItemByID(item.ID)
	}
}

func (s *FileServiceImpl) createOrGetFolder(name string, parentItem *models.Item, box *models.Box) (*models.Item, error) {
	var parentID *uint
	var path string

	if parentItem != nil {
		parentID = &parentItem.ID
		path = filepath.Join(parentItem.Path, name)
	} else {
		path = name
	}

	existingFolder, err := s.itemService.FindFolderByNameAndParent(name, parentID, box.ID)
	if err == nil && existingFolder != nil {
		return existingFolder, nil
	}

	newFolder := &models.Item{
		Name:     name,
		Type:     "folder",
		BoxID:    box.ID,
		ParentID: parentID,
		Path:     path,
	}
	if err := s.itemService.Create(newFolder); err != nil {
		return nil, err
	}

	return newFolder, nil
}

// createHashBasedFile stores a file using its hash and creates a database entry
func (s *FileServiceImpl) createHashBasedFile(
	name, fileType string,
	parentItem *models.Item,
	box *models.Box,
	fileHeader *multipart.FileHeader,
	properties []byte,
) (*models.Item, error) {
	var parentID *uint
	var itemPath string

	// Determine the item's path in the database
	if parentItem != nil {
		parentID = &parentItem.ID
		itemPath = filepath.Join(parentItem.Path, name)
	} else {
		itemPath = name
	}

	tempFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempFilePath := tempFile.Name()
	var deferErr error
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			deferErr = err
		}
	}(tempFilePath)
	if deferErr != nil {
		return nil, deferErr
	}
	defer func(tempFile *os.File) {
		err := tempFile.Close()
		if err != nil {
			deferErr = err
		}
	}(tempFile)
	if deferErr != nil {
		return nil, deferErr
	}
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			deferErr = err
		}
	}(src)
	if deferErr != nil {
		return nil, deferErr
	}
	sha256sum, sha512sum, err := helpers.SaveFileAndComputeChecksums(fileHeader, tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute checksums: %w", err)
	}

	// Hash-based path: prepare the storage directory using the first few characters of the hash
	firstHashPrefix := sha256sum[:2]
	secondHashPrefix := sha256sum[2:4]
	hashDir := filepath.Join(box.Path, secondHashPrefix, firstHashPrefix)

	if err := os.MkdirAll(hashDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create hash directory: %w", err)
	}

	// Final storage path will be [box_name]/[first_hash_prefix]/[second_hash_prefix]/[full_hash]
	finalStoragePath := filepath.Join(hashDir, sha256sum)

	// Check if the file already exists on box level
	if _, err := os.Stat(finalStoragePath); os.IsNotExist(err) {
		// File doesn't exist yet, copy it from the temp location
		if err := helpers.CopyFile(tempFilePath, finalStoragePath); err != nil {
			return nil, fmt.Errorf("failed to move file to hash storage: %w", err)
		}
	}

	// Check if an item with the same path already exists in the database
	existingItem, err := s.itemService.FindByPathAndBoxId(itemPath, box.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing item: %w", err)
	}

	if existingItem != nil {
		// Update the existing item with new hash and properties
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
			Path:       itemPath,
			Size:       fileHeader.Size,
			SHA256:     sha256sum,
			SHA512:     sha512sum,
			Properties: properties,
		}

		if err = s.itemService.Create(newFile); err != nil {
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

	if item.Type == "folder" {
		children, err := s.itemService.FindItemsByParentID(&item.ID, box.ID)
		if err != nil {
			return nil, err
		}
		item.Children = children
	}

	// Converting path to readable format
	item.Path = helpers.LtreeToUserPath(item)
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

func (s *FileServiceImpl) getHashBasedFilePath(item *models.Item) string {
	hashPrefix := item.SHA256[:2]
	return filepath.Join(s.configuration.Storage.Path, hashPrefix, item.SHA256)
}

func (s *FileServiceImpl) DeleteItemOnDisk(item models.Item, box *models.Box) error {
	itemLog := s.logService.Log.WithFields(logrus.Fields{
		"name": item.Name,
		"path": item.Path,
		"job":  "clean",
	})

	itemLog.Debug("Deleting item(s) from the database")
	err := s.itemService.HardDelete(&item)
	if err != nil {
		itemLog.WithError(err).Error("Failed to delete item(s) from the database")
		return err
	}

	if item.Type == "folder" {
		itemLog.Info("Folder deleted from database")
		return nil
	}

	// For files, we should check if any other items reference the same hash
	// before deleting the file from storage
	itemsWithSameHash, err := s.itemService.ItemsSearch(
		"sha256 eq \""+item.SHA256+"\" and box_id eq \""+strconv.Itoa(int(box.ID))+"\"",
		"id",
		1,
		0,
	)
	if err != nil {
		itemLog.WithError(err).Error("Failed to check for other items with same hash")
		return err
	}

	// If no other items reference this hash, delete the file
	if len(itemsWithSameHash) == 0 {
		hashFilePath := s.getHashBasedFilePath(&item)
		itemLog.Debug("Deleting file from hash storage: " + hashFilePath)

		if err := os.Remove(hashFilePath); err != nil && !os.IsNotExist(err) {
			itemLog.WithError(err).Error("Failed to delete file from hash storage")
			return err
		}

		// Check if the hash directory is empty and remove it if so
		hashDir := filepath.Dir(hashFilePath)
		entries, err := os.ReadDir(hashDir)
		if err != nil {
			itemLog.WithError(err).Error("Failed to read hash directory")
			return err
		}

		if len(entries) == 0 {
			if err := os.Remove(hashDir); err != nil {
				itemLog.WithError(err).Error("Failed to remove empty hash directory")
			}
		}
	} else {
		itemLog.Info("File still referenced by other items, not deleting from storage")
	}

	itemLog.Info("Successfully deleted item from database and storage if needed")
	return nil
}

func (s *FileServiceImpl) UpdateItem(item *models.Item) (*dto.ItemGetDTO, error) {
	itemLog := s.logService.Log.WithFields(logrus.Fields{
		"name": item.Name,
		"path": item.Path,
		"job":  "update",
	})
	itemLog.Debug("Updating item in database")
	err := s.itemService.UpdateItem(item)
	if err != nil {
		itemLog.WithError(err).Error("Failed to update item in database")
		return nil, err
	}
	itemLog.Debug("Successfully updated item from database")
	itemInDB, err := s.itemService.GetItemByID(item.ID)
	if err != nil {
		itemLog.WithError(err).Error("Failed to update item in database")
		return nil, err
	}
	if itemInDB == nil {
		itemLog.Debug("Item not found in database")
		return nil, fmt.Errorf("item not found")
	}
	itemLog.Debug("Converting item to dto")
	itemDTO, err := mapper.ToItemGetDTO(item)
	if err != nil {
		itemLog.WithError(err).Error("Failed to convert item to dto")
		return nil, err
	}
	return itemDTO, nil
}
