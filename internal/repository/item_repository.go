package repository

import (
	"Boxed/internal/models"
	"errors"
	"gorm.io/gorm"
	"math"
	"strings"
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

	//sanitizedPath := cmd.SanitizeLtreeIdentifier(path)
	sanitizedPath := strings.Replace(path, "/", ".", -1)
	err := r.db.Where("path = ? AND box_id = ?", sanitizedPath, boxID).First(&item).Error
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
	query := `
        WITH RECURSIVE descendants AS (
            SELECT id
            FROM items
            WHERE id = ?

            UNION ALL

            SELECT i.id
            FROM items i
            INNER JOIN descendants d ON i.parent_id = d.id
        )
        DELETE FROM items
        WHERE id IN (SELECT id FROM descendants);
    `
	return tx.Exec(query, parentID).Error
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
