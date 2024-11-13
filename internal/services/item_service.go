package services

import (
	"Boxed/internal/models"
	"Boxed/internal/repository"
	"encoding/json"
)

type ItemService interface {
	CreateItem(name, path, itemType string, size int64, boxID uint, properties map[string]interface{}) (*models.Item, error)
	GetItemByID(id uint) (*models.Item, error)
	UpdateItem(id uint, name, path string, properties map[string]interface{}) (*models.Item, error)
	DeleteItem(id uint) error
	GetItems() ([]models.Item, error)
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

func (s *itemServiceImpl) GetItemByID(id uint) (*models.Item, error) {
	return s.itemRepo.FindByID(id)
}

func (s *itemServiceImpl) UpdateItem(id uint, name, path string, properties map[string]interface{}) (*models.Item, error) {
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

func (s *itemServiceImpl) DeleteItem(id uint) error {
	return s.itemRepo.Delete(id)
}

func (s *itemServiceImpl) GetItems() ([]models.Item, error) {
	return s.itemRepo.FindAll()
}
