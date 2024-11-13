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

type MockBoxService struct {
	mock.Mock
}

func (m *MockBoxService) CreateBox(name, path string, properties map[string]interface{}) (*models.Box, error) {
	args := m.Called(name, path, properties)
	return args.Get(0).(*models.Box), args.Error(1)
}

func (m *MockBoxService) GetBoxByID(id uint) (*models.Box, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Box), args.Error(1)
}

func (m *MockBoxService) UpdateBox(id uint, name, path string, properties map[string]interface{}) (*models.Box, error) {
	args := m.Called(id, name, path, properties)
	return args.Get(0).(*models.Box), args.Error(1)
}

func (m *MockBoxService) DeleteBox(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockBoxService) GetBoxes() ([]models.Box, error) {
	args := m.Called()
	return args.Get(0).([]models.Box), args.Error(1)
}

func TestBoxHandler_CreateBox(t *testing.T) {
	app := fiber.New()
	mockService := new(MockBoxService)
	handler := NewBoxHandler(mockService)

	app.Post("/boxes", handler.CreateBox)

	reqBody := map[string]interface{}{
		"name":       "New Box",
		"path":       "/path/to/box",
		"properties": map[string]interface{}{"key": "value"},
	}
	reqBodyJSON, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	box := &models.Box{BaseModel: models.BaseModel{ID: 1}, Name: "New Box", Path: "/path/to/box"}
	mockService.On("CreateBox", "New Box", "/path/to/box", reqBody["properties"]).Return(box, nil)

	req := httptest.NewRequest(http.MethodPost, "/boxes", bytes.NewReader(reqBodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestBoxHandler_GetBoxByID(t *testing.T) {
	app := fiber.New()
	mockService := new(MockBoxService)
	handler := NewBoxHandler(mockService)

	app.Get("/boxes/:id", handler.GetBoxByID)

	box := &models.Box{BaseModel: models.BaseModel{ID: 1}, Name: "Box 1", Path: "/path/box1"}
	mockService.On("GetBoxByID", uint(1)).Return(box, nil)

	req := httptest.NewRequest(http.MethodGet, "/boxes/1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestBoxHandler_ListBoxes(t *testing.T) {
	app := fiber.New()
	mockService := new(MockBoxService)
	handler := NewBoxHandler(mockService)

	app.Get("/boxes", handler.ListBoxes)

	boxes := []models.Box{
		{BaseModel: models.BaseModel{ID: 1}, Name: "Box 1"},
		{BaseModel: models.BaseModel{ID: 2}, Name: "Box 2"},
	}
	mockService.On("GetBoxes").Return(boxes, nil)

	req := httptest.NewRequest(http.MethodGet, "/boxes", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockService.AssertExpectations(t)
}
