package services

import (
	"Boxed/internal/dto"
	"Boxed/internal/helpers"
	"Boxed/internal/mapper"
	"Boxed/internal/models"
	"Boxed/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
)

type ItemService interface {
	GetItemByID(id uint) (*dto.ItemGetDTO, error)
	UpdateItemPartial(id uint, name, path string, properties map[string]interface{}) (*models.Item, error)
	DeleteItem(id uint, force bool) error
	GetItems() ([]dto.ItemGetDTO, error)
	FindDeleted() ([]models.Item, error)
	FindByPathAndBoxId(path string, boxID uint) (*models.Item, error)
	FindItemsByParentID(parentID *uint, boxID uint) ([]models.Item, error)
	FindFolderByNameAndParent(name string, parentID *uint, boxID uint) (*models.Item, error)
	GetAllDescendants(parentID uint, maxLevel int) ([]models.Item, error)
	HardDelete(item *models.Item) error
	Create(item *models.Item) error
	UpdateItem(item *models.Item) error
	ItemsSearch(
		filter string,
		order string,
		limit int,
		offset int,
	) ([]models.Item, error)
}

type itemServiceImpl struct {
	itemRepo repository.ItemRepository
}

func NewItemService(itemRepository repository.ItemRepository) ItemService {
	return &itemServiceImpl{itemRepo: itemRepository}
}

func (s *itemServiceImpl) Create(item *models.Item) error {
	var parentPath string
	if item.ParentID != nil {
		parentItem, err := s.itemRepo.FindByID(*item.ParentID)
		if err != nil {
			return err
		}
		if parentItem == nil {
			return errors.New("parent item not found")
		}
		parentPath = parentItem.Path
	}

	// Bygg sökvägen korrekt
	if parentPath != "" {
		item.Path = fmt.Sprintf("%s.%s", parentPath, helpers.SanitizeLtreeIdentifier(item.Name))
	} else {
		item.Path = helpers.SanitizeLtreeIdentifier(item.Name)
	}

	return s.itemRepo.Create(item)
}

func (s *itemServiceImpl) GetItemByID(id uint) (*dto.ItemGetDTO, error) {
	item, err := s.itemRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	itemGetDto, err := mapper.ToItemGetDTO(item)
	if err != nil {
		return nil, err
	}
	return itemGetDto, nil
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

func (s *itemServiceImpl) GetItems() ([]dto.ItemGetDTO, error) {
	items, err := s.itemRepo.FindAll()
	if err != nil {
		return nil, err
	}
	itemsGetDTOs, err := mapper.ToItemsGetDTOs(items)
	if err != nil {
		return nil, err
	}
	return itemsGetDTOs, nil
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

func (s *itemServiceImpl) ItemsSearch(
	filter string,
	order string,
	limit int,
	offset int,
) ([]models.Item, error) {
	whereClause := "1=1"
	var args []interface{}
	if filter != "" {
		parsedFilter, params := ParseFilter(filter)
		whereClause = parsedFilter
		args = append(args, params...)
	}
	return s.itemRepo.ItemsSearch(whereClause, args, order, limit, offset)
}

func (s *itemServiceImpl) GetItemProperties(path string, boxId uint) (json.RawMessage, error) {
	item, err := s.itemRepo.FindByPathAndBoxId(path, boxId)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}

	return item.Properties, nil
}
