package handlers

import (
	"Boxed/internal/dto"
	"Boxed/internal/models"
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

// Implementing all required methods from services.ItemService interface
func (m *MockItemService) GetItemByID(id uint) (*dto.ItemGetDTO, error) {
	args := m.Called(id)
	if dto, ok := args.Get(0).(*dto.ItemGetDTO); ok {
		return dto, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockItemService) UpdateItemPartial(id uint, name, path string, properties map[string]interface{}) (*models.Item, error) {
	args := m.Called(id, name, path, properties)
	if item, ok := args.Get(0).(*models.Item); ok {
		return item, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockItemService) DeleteItem(id uint, force bool) error {
	args := m.Called(id, force)
	return args.Error(0)
}

func (m *MockItemService) GetItems() ([]dto.ItemGetDTO, error) {
	args := m.Called()
	return args.Get(0).([]dto.ItemGetDTO), args.Error(1)
}

func (m *MockItemService) FindDeleted() ([]models.Item, error) {
	args := m.Called()
	return args.Get(0).([]models.Item), args.Error(1)
}

func (m *MockItemService) FindByPathAndBoxId(path string, boxID uint) (*models.Item, error) {
	args := m.Called(path, boxID)
	if item, ok := args.Get(0).(*models.Item); ok {
		return item, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockItemService) FindItemsByParentID(parentID *uint, boxID uint) ([]models.Item, error) {
	args := m.Called(parentID, boxID)
	return args.Get(0).([]models.Item), args.Error(1)
}

func (m *MockItemService) FindFolderByNameAndParent(name string, parentID *uint, boxID uint) (*models.Item, error) {
	args := m.Called(name, parentID, boxID)
	if item, ok := args.Get(0).(*models.Item); ok {
		return item, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockItemService) GetAllDescendants(parentID uint, maxLevel int) ([]models.Item, error) {
	args := m.Called(parentID, maxLevel)
	return args.Get(0).([]models.Item), args.Error(1)
}

func (m *MockItemService) HardDelete(item *models.Item) error {
	args := m.Called(item)
	return args.Error(0)
}

func (m *MockItemService) Create(item *models.Item) error {
	args := m.Called(item)
	return args.Error(0)
}

func (m *MockItemService) UpdateItem(item *models.Item) error {
	args := m.Called(item)
	return args.Error(0)
}

func (m *MockItemService) ItemsSearch(filter string, order string, limit int, offset int) ([]models.Item, error) {
	args := m.Called(filter, order, limit, offset)
	return args.Get(0).([]models.Item), args.Error(1)
}

func TestCreateItem_Success(t *testing.T) {
	app := fiber.New()
	mockService := new(MockItemService)
	handler := NewItemHandler(mockService)

	app.Post("/items", handler.CreateItem)

	tests := []struct {
		name         string
		input        map[string]interface{}
		expectedCode int
		setupMock    func(*models.Item)
	}{
		{
			name: "Create file item",
			input: map[string]interface{}{
				"name":       "test.txt",
				"path":       "/test/path",
				"type":       "file",
				"size":       1024,
				"box_id":     1,
				"properties": map[string]interface{}{"key": "value"},
			},
			expectedCode: http.StatusCreated,
			setupMock: func(expectedItem *models.Item) {
				mockService.On("Create", mock.MatchedBy(func(item *models.Item) bool {
					return item.Name == expectedItem.Name &&
						item.Path == expectedItem.Path &&
						item.Type == expectedItem.Type
				})).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedItem := &models.Item{
				Name:  tt.input["name"].(string),
				Path:  tt.input["path"].(string),
				Type:  tt.input["type"].(string),
				Size:  int64(tt.input["size"].(int)),
				BoxID: uint(tt.input["box_id"].(int)),
			}

			tt.setupMock(expectedItem)

			reqBody, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			mockService.AssertExpectations(t)
		})
	}
}

func TestGetItemByID_Scenarios(t *testing.T) {
	app := fiber.New()
	mockService := new(MockItemService)
	handler := NewItemHandler(mockService)

	app.Get("/items/:id", handler.GetItemByID)

	tests := []struct {
		name          string
		itemID        string
		setupMock     func()
		expectedCode  int
		checkResponse func(*testing.T, *http.Response)
	}{
		{
			name:   "Successfully get item",
			itemID: "1",
			setupMock: func() {
				mockService.On("GetItemByID", uint(1)).Return(&dto.ItemGetDTO{
					ID:   1,
					Name: "Test Item",
					Path: "/test/path",
				}, nil).Once()
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result dto.ItemGetDTO
				body, _ := io.ReadAll(resp.Body)
				err := json.Unmarshal(body, &result)
				assert.NoError(t, err)
				assert.Equal(t, uint(1), result.ID)
				assert.Equal(t, "Test Item", result.Name)
			},
		},
		{
			name:   "Item not found",
			itemID: "999",
			setupMock: func() {
				mockService.On("GetItemByID", uint(999)).Return(nil, errors.New("not found")).Once()
			},
			expectedCode:  http.StatusNotFound,
			checkResponse: func(t *testing.T, resp *http.Response) {},
		},
		{
			name:          "Invalid ID format",
			itemID:        "invalid",
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp *http.Response) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodGet, "/items/"+tt.itemID, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			tt.checkResponse(t, resp)
		})
	}
}

func TestListItems_Scenarios(t *testing.T) {
	app := fiber.New()
	mockService := new(MockItemService)
	handler := NewItemHandler(mockService)

	app.Get("/items", handler.ListItems)

	tests := []struct {
		name          string
		setupMock     func()
		expectedCode  int
		checkResponse func(*testing.T, *http.Response)
	}{
		{
			name: "Successfully list items",
			setupMock: func() {
				mockService.On("GetItems").Return([]dto.ItemGetDTO{
					{ID: 1, Name: "Item 1"},
					{ID: 2, Name: "Item 2"},
				}, nil).Once()
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var items []dto.ItemGetDTO
				body, _ := io.ReadAll(resp.Body)
				err := json.Unmarshal(body, &items)
				assert.NoError(t, err)
				assert.Len(t, items, 2)
			},
		},
		{
			name: "Service error",
			setupMock: func() {
				mockService.On("GetItems").Return([]dto.ItemGetDTO{}, errors.New("service error")).Once()
			},
			expectedCode:  http.StatusInternalServerError,
			checkResponse: func(t *testing.T, resp *http.Response) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodGet, "/items", nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			tt.checkResponse(t, resp)
		})
	}
}

func TestDeleteItem_Scenarios(t *testing.T) {
	app := fiber.New()
	mockService := new(MockItemService)
	handler := NewItemHandler(mockService)

	app.Delete("/items/:id", handler.DeleteItem)

	tests := []struct {
		name         string
		itemID       string
		forceDelete  bool
		setupMock    func()
		expectedCode int
	}{
		{
			name:        "Successfully delete item",
			itemID:      "1",
			forceDelete: true,
			setupMock: func() {
				mockService.On("DeleteItem", uint(1), true).Return(nil).Once()
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:         "Invalid ID",
			itemID:       "invalid",
			forceDelete:  false,
			setupMock:    func() {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "Delete error",
			itemID:      "1",
			forceDelete: true,
			setupMock: func() {
				mockService.On("DeleteItem", uint(1), true).Return(errors.New("delete error")).Once()
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			url := "/items/" + tt.itemID
			if tt.forceDelete {
				url += "?force=true"
			}

			req := httptest.NewRequest(http.MethodDelete, url, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}
