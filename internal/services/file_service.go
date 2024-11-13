package services

import (
	"Boxed/internal/cmd"
	"Boxed/internal/config"
	"Boxed/internal/models"
	"Boxed/internal/repository"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

type FileService interface {
	CreateFileStructure(box *models.Box, filePath string, fileHeader *multipart.FileHeader, flat bool) (*models.Item, error)
	FindBoxByPath(boxPath string) (*models.Box, error)
}

type FileServiceImpl struct {
	itemRepository repository.ItemRepository
	boxRepository  repository.BoxRepository
	configuration  config.Configuration
}

func NewFileService(itemRepository repository.ItemRepository, boxRepository repository.BoxRepository, configuration *config.Configuration) FileService {
	return &FileServiceImpl{
		itemRepository: itemRepository,
		boxRepository:  boxRepository,
		configuration:  *configuration,
	}
}

func (s *FileServiceImpl) CreateFileStructure(box *models.Box, filePath string, fileHeader *multipart.FileHeader, flat bool) (*models.Item, error) {
	pathParts := strings.Split(filePath, "/")

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

	fileName := pathParts[len(pathParts)-1]
	fileType := cmd.GetFileType(fileName)

	fileItem, err := s.createFileItem(fileName, fileType, parentItem, box, fileHeader)
	if err != nil {
		return nil, err
	}

	return fileItem, nil
}

func (s *FileServiceImpl) createOrGetFolderItem(name string, parentItem *models.Item, box *models.Box) (*models.Item, error) {
	var parentID *uint
	var path string

	if parentItem != nil {
		parentID = &parentItem.ID
		path = filepath.Join(parentItem.Path, name)
	} else {
		// For top-level folders, path is box.Path/name
		path = filepath.Join(s.configuration.Storage.Path, name)
	}

	// Check if the folder already exists
	existingFolder, err := s.itemRepository.FindFolderByNameAndParent(name, parentID, box.ID)
	if err == nil && existingFolder != nil {
		return existingFolder, nil
	}

	// Create the directory on disk
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
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
	if err := s.itemRepository.Create(newFolder); err != nil {
		return nil, err
	}

	return newFolder, nil
}
func (s *FileServiceImpl) createFileItem(name, fileType string, parentItem *models.Item, box *models.Box, fileHeader *multipart.FileHeader) (*models.Item, error) {
	var parentID *uint
	var dirPath string

	if parentItem != nil {
		parentID = &parentItem.ID
		dirPath = parentItem.Path
		if dirPath == "" {
			return nil, fmt.Errorf("parentItem.Path is empty")
		}
	} else {
		dirPath = filepath.Join(s.configuration.Storage.Path, box.Name)
		if dirPath == "" {
			return nil, fmt.Errorf("box.Path is empty")
		}
	}

	filePath := filepath.Join(dirPath, name)

	// Ensure the directory exists on disk
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return nil, err
	}

	sha256sum, sha512sum, err := cmd.SaveFileAndComputeChecksums(fileHeader, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to save file and compute checksums: %w", err)
	}

	newFile := &models.Item{
		Name:     name,
		Type:     fileType,
		BoxID:    box.ID,
		ParentID: parentID,
		Path:     filePath,
		Size:     fileHeader.Size,
		SHA256:   sha256sum,
		SHA512:   sha512sum,
	}

	if err := s.itemRepository.Create(newFile); err != nil {
		return nil, fmt.Errorf("failed to create item record: %w", err)
	}

	return newFile, nil
}
func (s *FileServiceImpl) FindBoxByPath(boxPath string) (*models.Box, error) {
	return s.boxRepository.FindByName(boxPath)
}
