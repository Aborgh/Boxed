package repository

import (
	"Boxed/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func setupTestDBWithItems() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	err := db.AutoMigrate(&models.Box{}, &models.Item{})
	if err != nil {
		panic(err)
	}
	return db
}

func TestItemRepository_Create(t *testing.T) {
	db := setupTestDBWithItems()
	itemRepo := NewGenericRepository[models.Item](db)

	item := &models.Item{Name: "Test Item", Path: "/test/item/path", Type: "file"}
	err := itemRepo.Create(item)

	assert.NoError(t, err)
	assert.NotZero(t, item.ID)
}

func TestItemRepository_FindByID(t *testing.T) {
	db := setupTestDBWithItems()
	itemRepo := NewGenericRepository[models.Item](db)

	item := &models.Item{Name: "FindByID Item", Path: "/find/id/item", Type: "file"}
	err := itemRepo.Create(item)
	assert.NoError(t, err)

	foundItem, err := itemRepo.FindByID(item.ID)

	assert.NoError(t, err)
	assert.Equal(t, item.ID, foundItem.ID)
	assert.Equal(t, "FindByID Item", foundItem.Name)
}

func TestItemRepository_FindAll(t *testing.T) {
	db := setupTestDBWithItems()
	itemRepo := NewGenericRepository[models.Item](db)

	err := itemRepo.Create(&models.Item{Name: "Item 1", Path: "/path/item1", Type: "file"})
	assert.NoError(t, err)
	err = itemRepo.Create(&models.Item{Name: "Item 2", Path: "/path/item2", Type: "folder"})
	assert.NoError(t, err)
	items, err := itemRepo.FindAll()

	assert.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "Item 1", items[0].Name)
	assert.Equal(t, "Item 2", items[1].Name)
}

func TestItemRepository_Update(t *testing.T) {
	db := setupTestDBWithItems()
	itemRepo := NewGenericRepository[models.Item](db)

	item := &models.Item{Name: "Original Item", Path: "/original/item", Type: "file"}
	err := itemRepo.Create(item)
	assert.NoError(t, err)

	item.Name = "Updated Item"
	err = itemRepo.Update(item)
	assert.NoError(t, err)

	updatedItem, err := itemRepo.FindByID(item.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Item", updatedItem.Name)
}

func TestItemRepository_Delete(t *testing.T) {
	db := setupTestDBWithItems()
	itemRepo := NewGenericRepository[models.Item](db)

	item := &models.Item{Name: "To Delete", Path: "/delete/item", Type: "file"}
	err := itemRepo.Create(item)
	assert.NoError(t, err)

	err = itemRepo.Delete(item.ID)
	assert.NoError(t, err)

	deletedItem, err := itemRepo.FindByID(item.ID)
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
	assert.NotEqual(t, item.ID, deletedItem.ID)
}
