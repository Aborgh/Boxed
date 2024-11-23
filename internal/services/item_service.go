package services

import (
	"Boxed/internal/models"
	"Boxed/internal/repository"
	"encoding/json"
	"errors"
)

type ItemService interface {
	CreateItem(name, path, itemType string, size int64, boxID uint, properties map[string]interface{}) (*models.Item, error)
	GetItemByID(id uint) (*models.Item, error)
	UpdateItemPartial(id uint, name, path string, properties map[string]interface{}) (*models.Item, error)
	DeleteItem(id uint, force bool) error
	GetItems() ([]models.Item, error)
	FindDeleted() ([]models.Item, error)
	FindByPathAndBoxId(path string, boxID uint) (*models.Item, error)
	FindItemsByParentID(parentID *uint, boxID uint) ([]models.Item, error)
	FindFolderByNameAndParent(name string, parentID *uint, boxID uint) (*models.Item, error)
	GetAllDescendants(parentID uint, maxLevel int) ([]models.Item, error)
	HardDelete(item *models.Item) error
	InsertItem(item *models.Item) error
	UpdateItem(item *models.Item) error
}

type itemServiceImpl struct {
	itemRepo repository.ItemRepository
}

func NewItemService(itemRepository repository.ItemRepository) ItemService {
	return &itemServiceImpl{itemRepo: itemRepository}
}

func (s *itemServiceImpl) CreateItem(name, path, itemType string, size int64, boxID uint, properties map[string]interface{}) (*models.Item, error) {
	propertiesJSON, _ := json.Marshal(properties)
	item := &models.Item{Name: name, Path: path, Type: itemType, Size: size, BoxID: boxID, Properties: propertiesJSON}
	if err := s.itemRepo.Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *itemServiceImpl) InsertItem(item *models.Item) error {
	return s.itemRepo.Create(item)
}

func (s *itemServiceImpl) GetItemByID(id uint) (*models.Item, error) {
	return s.itemRepo.FindByID(id)
}

func (s *itemServiceImpl) UpdateItem(item *models.Item) error {
	return s.itemRepo.Update(item)
}

func (s *itemServiceImpl) UpdateItemPartial(id uint, name, path string, properties map[string]interface{}) (*models.Item, error) {
	item, err := s.itemRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	item.Name = name
	item.Path = path
	item.Properties, _ = json.Marshal(properties)
	if err := s.itemRepo.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *itemServiceImpl) DeleteItem(id uint, force bool) error {
	item, err := s.itemRepo.FindByID(id)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}
	if item.Type == "folder" && !force {
		return errors.New("to delete folder the 'force' option is required")
	}
	return s.itemRepo.Delete(id)
}

func (s *itemServiceImpl) GetItems() ([]models.Item, error) {
	return s.itemRepo.FindAll()
}

func (s *itemServiceImpl) FindDeleted() ([]models.Item, error) {
	return s.itemRepo.FindDeleted()
}

func (s *itemServiceImpl) FindByPathAndBoxId(path string, boxID uint) (*models.Item, error) {
	return s.itemRepo.FindByPathAndBoxId(path, boxID)
}

func (s *itemServiceImpl) FindItemsByParentID(parentID *uint, boxID uint) ([]models.Item, error) {
	return s.itemRepo.FindItemsByParentID(parentID, boxID)
}

func (s *itemServiceImpl) HardDelete(item *models.Item) error {
	return s.itemRepo.HardDelete(item)
}

func (s *itemServiceImpl) FindFolderByNameAndParent(name string, parentID *uint, boxID uint) (*models.Item, error) {
	return s.itemRepo.FindFolderByNameAndParent(name, parentID, boxID)

}

func (s *itemServiceImpl) GetAllDescendants(parentID uint, maxLevel int) ([]models.Item, error) {
	return s.itemRepo.GetAllDescendants(parentID, maxLevel)
}
