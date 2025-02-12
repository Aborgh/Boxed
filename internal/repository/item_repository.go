package repository

import (
	"Boxed/internal/helpers"
	"Boxed/internal/models"
	"errors"
	"gorm.io/gorm"
	"math"
)

type ItemRepository interface {
	GenericRepository[models.Item]
	FindFolderByNameAndParent(name string, parentID *uint, boxID uint) (*models.Item, error)
	FindByPathAndBoxId(path string, boxID uint) (*models.Item, error)
	FindItemsByParentID(parentID *uint, boxID uint) ([]models.Item, error)
	FindDeleted() ([]models.Item, error)
	HardDelete(item *models.Item) error
	GetAllDescendants(parentID uint, maxLevel int) ([]models.Item, error)
	ItemsSearch(
		whereClause string,
		args []interface{},
		order string,
		limit int,
		offset int,
	) ([]models.Item, error)
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
	ltreePath := helpers.PathToLtree(path)

	result := r.db.Where("path = ? AND box_id = ?", ltreePath, boxID).First(&item)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	item.Path = helpers.LtreeToUserPath(&item)
	return &item, nil
}

func (r *ItemRepositoryImpl[T]) Create(item *models.Item) error {
	// Convert the path to ltree format before saving
	item.Path = helpers.PathToLtree(item.Path)

	err := r.db.Create(item).Error
	if err != nil {
		return err
	}

	// Convert back to user-friendly format after saving
	item.Path = helpers.LtreeToUserPath(item)
	return nil
}

func (r *ItemRepositoryImpl[T]) Update(item *models.Item) error {
	// Convert to ltree format for storage
	storagePath := helpers.UserPathToLtree(item.Path)
	itemToUpdate := *item
	itemToUpdate.Path = storagePath

	err := r.db.Save(&itemToUpdate).Error
	if err != nil {
		return err
	}

	// Keep the original item's path in user-friendly format
	return nil
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

	// Convert paths for all items
	for i := range items {
		items[i].Path = helpers.LtreeToUserPath(&items[i])
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

func (r *ItemRepositoryImpl[T]) HardDelete(item *models.Item) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if item.Type == "folder" {
			// Delete the item and all its descendants
			err := r.deleteItemAndDescendants(tx, item.ID)
			if err != nil {
				return err
			}
		} else {
			// Delete the single item
			err := tx.Unscoped().Delete(item).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ItemRepositoryImpl[T]) deleteItemAndDescendants(tx *gorm.DB, parentID uint) error {
	var parentItem models.Item
	if err := tx.Unscoped().First(&parentItem, parentID).Error; err != nil {
		return err
	}
	query := "DELETE FROM items where path <@ ?"
	result := tx.Exec(query, parentItem.Path)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		// TODO Logging
	}

	return nil
}

func (r *ItemRepositoryImpl[T]) GetAllDescendants(parentID uint, maxLevel int) ([]models.Item, error) {
	parentItem, err := r.FindByID(parentID)
	if err != nil {
		return nil, err
	}
	if parentItem == nil {
		return nil, errors.New("parent item not found")
	}
	if maxLevel < 0 {
		maxLevel = math.MaxInt32
	}

	var items []models.Item
	query := r.db.Where("path <@ ?", parentItem.Path)
	if maxLevel > 0 {
		query = query.Where("nlevel(path) <= ?", maxLevel)
	}
	if err = query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil

}

func (r *ItemRepositoryImpl[T]) ItemsSearch(
	whereClause string,
	args []interface{},
	order string,
	limit int,
	offset int,
) ([]models.Item, error) {
	var items []models.Item
	query := r.db.Where(whereClause, args...).
		Order(order).
		Limit(limit).
		Offset(offset)
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ItemRepositoryImpl[T]) UpdatePath(oldPath, newPath string) error {
	return r.db.Exec(`
		UPDATE items
		SET path = regexp_replace(path::text, ?, ?, 'g')::ltree
		WHERE path <@ ?`, oldPath, newPath, oldPath).Error
}
