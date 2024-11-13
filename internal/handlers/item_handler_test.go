package handlers

import (
	"Boxed/internal/models"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockItemService struct {
	mock.Mock
}

func (m *MockItemService) CreateItem(name, path, itemType string, size int64, boxID uint, properties map[string]interface{}) (*models.Item, error) {
	args := m.Called(name, path, itemType, size, boxID, properties)
	return args.Get(0).(*models.Item), args.Error(1)
}

func (m *MockItemService) GetItemByID(id uint) (*models.Item, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Item), args.Error(1)
}

func (m *MockItemService) UpdateItem(id uint, name, path string, properties map[string]interface{}) (*models.Item, error) {
	args := m.Called(id, name, path, properties)
	return args.Get(0).(*models.Item), args.Error(1)
}

func (m *MockItemService) DeleteItem(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockItemService) GetItems() ([]models.Item, error) {
	args := m.Called()
	return args.Get(0).([]models.Item), args.Error(1)
}

func TestItemHandler_CreateItem(t *testing.T) {
	app := fiber.New()
	mockService := new(MockItemService)
	handler := NewItemHandler(mockService)

	app.Post("/items", handler.CreateItem)

	reqBody := map[string]interface{}{
		"name":       "New Item",
		"path":       "/path/to/item",
		"type":       "file",
		"size":       1024,
		"box_id":     1,
		"properties": map[string]interface{}{"key": "value"},
	}
	reqBodyJSON, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	item := &models.Item{BaseModel: models.BaseModel{ID: 1}, Name: "New Item", Path: "/path/to/item", BoxID: 1}
	mockService.On("CreateItem", "New Item", "/path/to/item", "file", int64(1024), uint(1), reqBody["properties"]).Return(item, nil)

	req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewReader(reqBodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestItemHandler_GetItemByID(t *testing.T) {
	app := fiber.New()
	mockService := new(MockItemService)
	handler := NewItemHandler(mockService)

	app.Get("/items/:id", handler.GetItemByID)

	item := &models.Item{BaseModel: models.BaseModel{ID: 1}, Name: "Item 1", Path: "/path/item1"}
	mockService.On("GetItemByID", uint(1)).Return(item, nil)

	req := httptest.NewRequest(http.MethodGet, "/items/1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestItemHandler_ListItems(t *testing.T) {
	app := fiber.New()
	mockService := new(MockItemService)
	handler := NewItemHandler(mockService)

	app.Get("/items", handler.ListItems)

	items := []models.Item{
		{BaseModel: models.BaseModel{ID: 1}, Name: "Item 1"},
		{BaseModel: models.BaseModel{ID: 2}, Name: "Item 2"},
	}
	mockService.On("GetItems").Return(items, nil)

	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockService.AssertExpectations(t)
}
