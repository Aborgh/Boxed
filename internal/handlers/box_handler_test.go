package handlers

import (
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

type MockBoxService struct {
	mock.Mock
}

func (m *MockBoxService) CreateBox(name string, properties map[string]interface{}) (*models.Box, error) {
	args := m.Called(name, properties)
	if box, ok := args.Get(0).(*models.Box); ok {
		return box, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBoxService) GetBoxByID(id uint) (*models.Box, error) {
	args := m.Called(id)
	if box, ok := args.Get(0).(*models.Box); ok {
		return box, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBoxService) UpdateBox(id uint, name string, properties map[string]interface{}) (*models.Box, error) {
	args := m.Called(id, name, properties)
	if box, ok := args.Get(0).(*models.Box); ok {
		return box, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBoxService) DeleteBox(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockBoxService) GetBoxes() ([]models.Box, error) {
	args := m.Called()
	return args.Get(0).([]models.Box), args.Error(1)
}

func (m *MockBoxService) GetBoxByPath(path string) (*models.Box, error) {
	args := m.Called(path)
	if box, ok := args.Get(0).(*models.Box); ok {
		return box, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBoxService) GetDeletedBoxes() ([]models.Box, error) {
	args := m.Called()
	return args.Get(0).([]models.Box), args.Error(1)
}

func TestCreateBox_ValidInput(t *testing.T) {
	app := fiber.New()
	mockService := new(MockBoxService)
	handler := NewBoxHandler(mockService)

	app.Post("/boxes", handler.CreateBox)

	tests := []struct {
		name          string
		input         map[string]interface{}
		expectedBox   *models.Box
		expectedError error
		expectedCode  int
	}{
		{
			name: "Successfully create box",
			input: map[string]interface{}{
				"name": "Test Box",
				"properties": map[string]interface{}{
					"description": "Test Description",
				},
			},
			expectedBox: &models.Box{
				BaseModel: models.BaseModel{ID: 1},
				Name:      "Test Box",
			},
			expectedError: nil,
			expectedCode:  http.StatusCreated,
		},
		{
			name: "Create box with empty properties",
			input: map[string]interface{}{
				"name":       "Test Box",
				"properties": map[string]interface{}{},
			},
			expectedBox: &models.Box{
				BaseModel: models.BaseModel{ID: 2},
				Name:      "Test Box",
			},
			expectedError: nil,
			expectedCode:  http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(tt.input)
			mockService.On("CreateBox", tt.input["name"].(string), tt.input["properties"]).
				Return(tt.expectedBox, tt.expectedError).Once()

			req := httptest.NewRequest(http.MethodPost, "/boxes", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			if tt.expectedCode == http.StatusCreated {
				var result models.Box
				body, _ := io.ReadAll(resp.Body)
				err = json.Unmarshal(body, &result)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBox.ID, result.ID)
				assert.Equal(t, tt.expectedBox.Name, result.Name)
			}
		})
	}
}

func TestCreateBox_InvalidInput(t *testing.T) {
	app := fiber.New()
	mockService := new(MockBoxService)
	handler := NewBoxHandler(mockService)

	app.Post("/boxes", handler.CreateBox)

	tests := []struct {
		name         string
		input        string
		expectedCode int
		setupMock    func()
	}{
		{
			name:         "Invalid JSON",
			input:        `{"name": }`,
			expectedCode: http.StatusBadRequest,
			setupMock:    func() {}, // No mock setup needed as fileService won't be called
		},
		{
			name:         "Empty request body",
			input:        ``,
			expectedCode: http.StatusBadRequest,
			setupMock:    func() {}, // No mock setup needed as fileService won't be called
		},
		{
			name:         "Missing required fields",
			input:        `{"properties": {}}`,
			expectedCode: http.StatusBadRequest,
			setupMock:    func() {}, // No mock setup needed as fileService won't be called
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup any mocks if needed (none for these invalid cases)
			tt.setupMock()

			req := httptest.NewRequest(http.MethodPost, "/boxes", bytes.NewReader([]byte(tt.input)))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			// Verify no unexpected calls were made
			mockService.AssertExpectations(t)
		})
	}
}

func TestUpdateBox_ValidationAndErrors(t *testing.T) {
	app := fiber.New()
	mockService := new(MockBoxService)
	handler := NewBoxHandler(mockService)

	app.Put("/boxes/:id", handler.UpdateBox)

	tests := []struct {
		name         string
		boxID        string
		input        map[string]interface{}
		setupMock    func()
		expectedCode int
	}{
		{
			name:  "Invalid box ID",
			boxID: "invalid",
			input: map[string]interface{}{
				"name":       "Updated Box",
				"properties": map[string]interface{}{},
			},
			setupMock:    func() {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:  "Box not found",
			boxID: "1",
			input: map[string]interface{}{
				"name":       "Updated Box",
				"properties": map[string]interface{}{},
			},
			setupMock: func() {
				mockService.On("UpdateBox", uint(1), "Updated Box", mock.Anything).
					Return(nil, errors.New("box not found")).Once()
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			reqBody, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPut, "/boxes/"+tt.boxID, bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

func TestDeleteBox_Scenarios(t *testing.T) {
	app := fiber.New()
	mockService := new(MockBoxService)
	handler := NewBoxHandler(mockService)

	app.Delete("/boxes/:id", handler.DeleteBox)

	tests := []struct {
		name         string
		boxID        string
		setupMock    func()
		expectedCode int
	}{
		{
			name:  "Successfully delete box",
			boxID: "1",
			setupMock: func() {
				mockService.On("DeleteBox", uint(1)).Return(nil).Once()
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:         "Invalid box ID",
			boxID:        "invalid",
			setupMock:    func() {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:  "Box deletion error",
			boxID: "1",
			setupMock: func() {
				mockService.On("DeleteBox", uint(1)).
					Return(errors.New("deletion error")).Once()
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodDelete, "/boxes/"+tt.boxID, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

func TestListBoxes_Scenarios(t *testing.T) {
	app := fiber.New()
	mockService := new(MockBoxService)
	handler := NewBoxHandler(mockService)

	app.Get("/boxes", handler.ListBoxes)

	tests := []struct {
		name          string
		setupMock     func()
		expectedCode  int
		checkResponse func(*testing.T, *http.Response)
	}{
		{
			name: "Successfully list boxes",
			setupMock: func() {
				mockService.On("GetBoxes").Return([]models.Box{
					{BaseModel: models.BaseModel{ID: 1}, Name: "Box 1"},
					{BaseModel: models.BaseModel{ID: 2}, Name: "Box 2"},
				}, nil).Once()
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var boxes []models.Box
				body, _ := io.ReadAll(resp.Body)
				err := json.Unmarshal(body, &boxes)
				assert.NoError(t, err)
				assert.Len(t, boxes, 2)
				assert.Equal(t, "Box 1", boxes[0].Name)
				assert.Equal(t, "Box 2", boxes[1].Name)
			},
		},
		{
			name: "Empty box list",
			setupMock: func() {
				mockService.On("GetBoxes").Return([]models.Box{}, nil).Once()
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var boxes []models.Box
				body, _ := io.ReadAll(resp.Body)
				err := json.Unmarshal(body, &boxes)
				assert.NoError(t, err)
				assert.Len(t, boxes, 0)
			},
		},
		{
			name: "Service error",
			setupMock: func() {
				mockService.On("GetBoxes").Return([]models.Box{}, errors.New("fileService error")).Once()
			},
			expectedCode:  http.StatusInternalServerError,
			checkResponse: func(t *testing.T, resp *http.Response) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodGet, "/boxes", nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			tt.checkResponse(t, resp)
		})
	}
}
