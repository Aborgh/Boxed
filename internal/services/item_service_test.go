package services

import (
	"Boxed/internal/models"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockItemRepository struct {
	mock.Mock
}

func (m *MockItemRepository) Create(item *models.Item) error {
	args := m.Called(item)
	return args.Error(0)
}

func (m *MockItemRepository) FindByID(id uint) (*models.Item, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Item), args.Error(1)
}

func (m *MockItemRepository) Update(item *models.Item) error {
	args := m.Called(item)
	return args.Error(0)
}

func (m *MockItemRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockItemRepository) FindAllByBoxID(boxID uint) ([]models.Item, error) {
	args := m.Called(boxID)
	return args.Get(0).([]models.Item), args.Error(1)
}

func (m *MockItemRepository) FindAll() ([]models.Item, error) {
	args := m.Called()
	return args.Get(0).([]models.Item), args.Error(1)
}

func TestItemService_GetItems(t *testing.T) {
	mockRepo := new(MockItemRepository)
	service := NewItemService(mockRepo)

	items := []models.Item{
		{BaseModel: models.BaseModel{ID: 1}, Name: "Item 1", Path: "/path/item1"},
		{BaseModel: models.BaseModel{ID: 1}, Name: "Item 2", Path: "/path/item2"},
	}

	mockRepo.On("FindAll").Return(items, nil)

	allItems, err := service.GetItems()

	assert.NoError(t, err)
	assert.Len(t, allItems, 2)
	assert.Equal(t, "Item 1", allItems[0].Name)
	assert.Equal(t, "Item 2", allItems[1].Name)
	mockRepo.AssertExpectations(t)
}

func TestItemService_CreateItem(t *testing.T) {
	mockRepo := new(MockItemRepository)
	service := NewItemService(mockRepo)

	properties := map[string]interface{}{"key": "value"}
	propertiesJSON, _ := json.Marshal(properties)
	item := &models.Item{Name: "Test Item", Path: "/path/to/item", Type: "file", BoxID: 1, Properties: propertiesJSON}

	mockRepo.On("Create", item).Return(nil)

	createdItem, err := service.CreateItem("Test Item", "/path/to/item", "file", 0, 1, properties)

	assert.NoError(t, err)
	assert.Equal(t, "Test Item", createdItem.Name)
	assert.Equal(t, "file", createdItem.Type)
	mockRepo.AssertExpectations(t)
}

func TestItemService_GetItemByID(t *testing.T) {
	mockRepo := new(MockItemRepository)
	service := NewItemService(mockRepo)

	item := &models.Item{BaseModel: models.BaseModel{ID: 1}, Name: "Test Item", Path: "/path/to/item"}
	mockRepo.On("FindByID", uint(1)).Return(item, nil)

	foundItem, err := service.GetItemByID(1)

	assert.NoError(t, err)
	assert.Equal(t, uint(1), foundItem.ID)
	assert.Equal(t, "Test Item", foundItem.Name)
	mockRepo.AssertExpectations(t)
}

func TestItemService_UpdateItem(t *testing.T) {
	mockRepo := new(MockItemRepository)
	service := NewItemService(mockRepo)

	item := &models.Item{BaseModel: models.BaseModel{ID: 1}, Name: "Original Item", Path: "/original/path"}
	updatedProperties := map[string]interface{}{"newKey": "newValue"}
	updatedPropertiesJSON, _ := json.Marshal(updatedProperties)

	mockRepo.On("FindByID", uint(1)).Return(item, nil)
	mockRepo.On("Update", item).Return(nil)

	updatedItem, err := service.UpdateItem(1, "Updated Item", "/updated/path", updatedProperties)

	assert.NoError(t, err)
	assert.Equal(t, "Updated Item", updatedItem.Name)
	assert.EqualValues(t, updatedPropertiesJSON, updatedItem.Properties)
	mockRepo.AssertExpectations(t)
}

func TestItemService_DeleteItem(t *testing.T) {
	mockRepo := new(MockItemRepository)
	service := NewItemService(mockRepo)

	mockRepo.On("Delete", uint(1)).Return(nil)

	err := service.DeleteItem(1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
