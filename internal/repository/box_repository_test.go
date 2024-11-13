package repository

import (
	"Boxed/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBWithBox() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	err := db.AutoMigrate(&models.Box{})
	if err != nil {
		return nil
	}
	return db
}

func TestBoxRepository_Create(t *testing.T) {
	db := setupTestDBWithBox()
	boxRepo := NewGenericRepository[models.Box](db)

	box := &models.Box{Name: "Test Box", Path: "/test/box/path"}
	err := boxRepo.Create(box)

	assert.NoError(t, err)
	assert.NotZero(t, box.ID)
}

func TestBoxRepository_FindByID(t *testing.T) {
	db := setupTestDBWithBox()
	boxRepo := NewGenericRepository[models.Box](db)

	box := &models.Box{Name: "FindByID Box", Path: "/find/id/box"}
	boxRepo.Create(box)

	foundBox, err := boxRepo.FindByID(box.ID)

	assert.NoError(t, err)
	assert.Equal(t, box.ID, foundBox.ID)
	assert.Equal(t, "FindByID Box", foundBox.Name)
}

func TestBoxRepository_FindAll(t *testing.T) {
	db := setupTestDBWithBox()
	boxRepo := NewGenericRepository[models.Box](db)

	err := boxRepo.Create(&models.Box{Name: "Box 1", Path: "/path/box1"})
	assert.NoError(t, err)
	err = boxRepo.Create(&models.Box{Name: "Box 2", Path: "/path/box2"})
	assert.NoError(t, err)

	boxes, err := boxRepo.FindAll()

	assert.NoError(t, err)
	assert.Len(t, boxes, 2)
	assert.Equal(t, "Box 1", boxes[0].Name)
	assert.Equal(t, "Box 2", boxes[1].Name)
}

func TestBoxRepository_Update(t *testing.T) {
	db := setupTestDBWithBox()
	boxRepo := NewGenericRepository[models.Box](db)

	box := &models.Box{Name: "Original Box", Path: "/original/box"}
	err := boxRepo.Create(box)
	assert.NoError(t, err)

	box.Name = "Updated Box"
	err = boxRepo.Update(box)
	assert.NoError(t, err)

	updatedBox, err := boxRepo.FindByID(box.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Box", updatedBox.Name)
}

func TestBoxRepository_Delete(t *testing.T) {
	db := setupTestDBWithBox()
	boxRepo := NewGenericRepository[models.Box](db)

	box := &models.Box{Name: "To Delete", Path: "/delete/box"}
	err := boxRepo.Create(box)
	assert.NoError(t, err)

	err = boxRepo.Delete(box.ID)
	assert.NoError(t, err)

	deletedBox, err := boxRepo.FindByID(box.ID)
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
	assert.NotEqual(t, box.ID, deletedBox.ID)
}
