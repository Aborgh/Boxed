package repository

import (
	"Boxed/internal/models"
	"errors"
	"gorm.io/gorm"
)

type ItemRepository interface {
	GenericRepository[models.Item]
	FindFolderByNameAndParent(name string, parentID *uint, boxID uint) (*models.Item, error)
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
