package repository

import (
	"Boxed/internal/models"
	"errors"
	"gorm.io/gorm"
)

type ItemRepository interface {
	GenericRepository[models.Item]
	FindFolderByNameAndParent(name string, parentID *uint, boxID uint) (*models.Item, error)
	FindByPathAndBoxId(path string, boxID uint) (*models.Item, error)
	FindItemsByParentID(parentID *uint, boxID uint) ([]models.Item, error)
	FindDeleted() ([]models.Item, error)
}

type ItemRepositoryImpl[T models.Item] struct {
	GenericRepository[models.Item]
	db *gorm.DB
}

func NewItemRepository(db *gorm.DB) ItemRepository {
	return &ItemRepositoryImpl[models.Item]{
		GenericRepository: NewGenericRepository[models.Item](db),
		db:                db,
	}
}

func (r *ItemRepositoryImpl[T]) FindFolderByNameAndParent(name string, parentID *uint, boxID uint) (*models.Item, error) {
	var folder models.Item
	query := r.db.Where("name = ? AND box_id = ? AND type = ?", name, boxID, "folder")
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	err := query.First(&folder).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &folder, nil
}

func (r *ItemRepositoryImpl[T]) FindByPathAndBoxId(path string, boxID uint) (*models.Item, error) {
	var item models.Item
	err := r.db.Where("path = ? AND box_id = ?", path, boxID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *ItemRepositoryImpl[T]) FindItemsByParentID(parentID *uint, boxID uint) ([]models.Item, error) {
	var items []models.Item
	var err error
	if parentID != nil {
		err = r.db.Where("parent_id = ? AND box_id = ?", *parentID, boxID).Find(&items).Error
	} else {
		err = r.db.Where("parent_id IS NULL AND box_id = ?", boxID).Find(&items).Error
	}
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ItemRepositoryImpl[T]) FindDeleted() ([]models.Item, error) {
	var items []models.Item
	var err error
	err = r.db.Unscoped().Where("deleted_at IS NOT NULL").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}
