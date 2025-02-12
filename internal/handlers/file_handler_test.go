package handlers

import (
	"Boxed/internal/dto"
	"Boxed/internal/models"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFileService implements the FileService interface for testing
type MockFileService struct {
	mock.Mock
}

func (m *MockFileService) CreateFileStructure(box *models.Box, filePath string, fileHeader *multipart.FileHeader, flat bool, properties string) (*dto.ItemGetDTO, error) {
	args := m.Called(box, filePath, fileHeader, flat, properties)
	if dto, ok := args.Get(0).(*dto.ItemGetDTO); ok {
		return dto, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockFileService) FindBoxByPath(boxPath string) (*models.Box, error) {
	args := m.Called(boxPath)
	if box, ok := args.Get(0).(*models.Box); ok {
		return box, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockFileService) ListFileOrFolder(boxName string, itemPath string) (*models.Item, error) {
	args := m.Called(boxName, itemPath)
	if item, ok := args.Get(0).(*models.Item); ok {
		return item, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockFileService) GetFileItem(box *models.Box, filePath string) (*models.Item, error) {
	args := m.Called(box, filePath)
	if item, ok := args.Get(0).(*models.Item); ok {
		return item, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockFileService) GetStoragePath() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockFileService) DeleteItemOnDisk(item models.Item, box *models.Box) error {
	args := m.Called(item, box)
	return args.Error(0)
}

func (m *MockFileService) GetItemProperties(itemPath string, boxId uint) (json.RawMessage, error) {
	args := m.Called(itemPath, boxId)
	if raw, ok := args.Get(0).(json.RawMessage); ok {
		return raw, args.Error(1)
	}
	return nil, args.Error(1)
}

// Helper function to create multipart form data for testing
func createMultipartFormData(t *testing.T, fieldName, fileName, content string, properties string) (*bytes.Buffer, string) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add file
	part, err := writer.CreateFormFile(fieldName, fileName)
	assert.NoError(t, err)

	_, err = io.Copy(part, strings.NewReader(content))
	assert.NoError(t, err)

	// Add properties as form field if provided
	if properties != "" {
		err = writer.WriteField("properties", properties)
		assert.NoError(t, err)
	}

	err = writer.Close()
	assert.NoError(t, err)

	return &b, writer.FormDataContentType()
}

func TestUploadFile_Success(t *testing.T) {
	app := fiber.New()
	mockService := new(MockFileService)
	handler := NewFileHandler(mockService)

	app.Post("/upload/:box/*", handler.UploadFile)

	tests := []struct {
		name         string
		boxName      string
		filePath     string
		fileContent  string
		properties   string
		setupMock    func(*models.Box)
		expectedCode int
	}{
		{
			name:        "Upload file successfully",
			boxName:     "testbox",
			filePath:    "folder/test.txt",
			fileContent: "test content",
			properties:  "key=value",
			setupMock: func(box *models.Box) {
				mockService.On("FindBoxByPath", "testbox").Return(box, nil).Once()
				mockService.On("CreateFileStructure",
					box,
					"folder/test.txt",
					mock.AnythingOfType("*multipart.FileHeader"),
					false,
					"key=value",
				).Return(&dto.ItemGetDTO{
					ID:   1,
					Name: "test.txt",
					Path: "folder/test.txt",
				}, nil).Once()
			},
			expectedCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			box := &models.Box{
				BaseModel: models.BaseModel{ID: 1},
				Name:      tt.boxName,
			}
			tt.setupMock(box)

			body, contentType := createMultipartFormData(t, "file", "test.txt", tt.fileContent, tt.properties)

			req := httptest.NewRequest(http.MethodPost, "/upload/"+tt.boxName+"/"+tt.filePath, body)
			req.Header.Set("Content-Type", contentType)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			mockService.AssertExpectations(t)
		})
	}
}

func TestUploadFile_Errors(t *testing.T) {
	app := fiber.New()
	mockService := new(MockFileService)
	handler := NewFileHandler(mockService)

	app.Post("/upload/:box/*", handler.UploadFile)

	tests := []struct {
		name         string
		boxName      string
		filePath     string
		setupMock    func()
		expectedCode int
	}{
		{
			name:     "Box not found",
			boxName:  "nonexistent",
			filePath: "test.txt",
			setupMock: func() {
				mockService.On("FindBoxByPath", "nonexistent").
					Return(nil, errors.New("box not found")).Once()
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:     "Invalid file",
			boxName:  "testbox",
			filePath: "test.txt",
			setupMock: func() {
				mockService.On("FindBoxByPath", "testbox").
					Return(&models.Box{BaseModel: models.BaseModel{ID: 1}}, nil).Once()
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodPost, "/upload/"+tt.boxName+"/"+tt.filePath, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			mockService.AssertExpectations(t)
		})
	}
}

func TestListFileOrFolder(t *testing.T) {
	app := fiber.New()
	mockService := new(MockFileService)
	handler := NewFileHandler(mockService)

	app.Get("/:box/*", handler.ListFileOrFolder)

	tests := []struct {
		name          string
		boxName       string
		path          string
		setupMock     func()
		expectedCode  int
		checkResponse func(*testing.T, *http.Response)
	}{
		{
			name:    "List root folder",
			boxName: "testbox",
			path:    "/",
			setupMock: func() {
				mockService.On("ListFileOrFolder", "testbox", "").
					Return(&models.Item{
						Name: "testbox",
						Type: "folder",
						Children: []models.Item{
							{Name: "file1.txt", Type: "file"},
							{Name: "folder1", Type: "folder"},
						},
					}, nil).Once()
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result models.Item
				body, _ := io.ReadAll(resp.Body)
				err := json.Unmarshal(body, &result)
				assert.NoError(t, err)
				assert.Equal(t, "testbox", result.Name)
				assert.Equal(t, "folder", result.Type)
				assert.Len(t, result.Children, 2)
			},
		},
		{
			name:         "Invalid path format",
			boxName:      "testbox",
			path:         "/invalid_path",
			setupMock:    func() {},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodGet, "/"+tt.boxName+tt.path, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestDownloadFile(t *testing.T) {
	// Create a temp dir for test files
	tmpDir := t.TempDir()

	app := fiber.New()
	mockService := new(MockFileService)
	handler := NewFileHandler(mockService)

	app.Get("/download/:box/*", handler.DownloadFile)

	tests := []struct {
		name         string
		boxName      string
		filePath     string
		fileContent  string
		setupMock    func(string)
		expectedCode int
	}{
		{
			name:        "Download file success",
			boxName:     "testbox",
			filePath:    "test.txt",
			fileContent: "test content",
			setupMock: func(storagePath string) {
				box := &models.Box{BaseModel: models.BaseModel{ID: 1}, Name: "testbox"}
				mockService.On("FindBoxByPath", "testbox").Return(box, nil).Once()
				mockService.On("GetFileItem", box, "test.txt").
					Return(&models.Item{
						Name: "test.txt",
						Type: "file",
						Path: "test.txt",
					}, nil).Once()
				mockService.On("GetStoragePath").Return(storagePath).Once()
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Invalid path",
			boxName:      "testbox",
			filePath:     "../test.txt",
			setupMock:    func(storagePath string) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:     "Box not found",
			boxName:  "nonexistent",
			filePath: "test.txt",
			setupMock: func(storagePath string) {
				mockService.On("FindBoxByPath", "nonexistent").
					Return(nil, errors.New("box not found")).Once()
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:     "File not found",
			boxName:  "testbox",
			filePath: "nonexistent.txt",
			setupMock: func(storagePath string) {
				box := &models.Box{BaseModel: models.BaseModel{ID: 1}, Name: "testbox"}
				mockService.On("FindBoxByPath", "testbox").Return(box, nil).Once()
				mockService.On("GetFileItem", box, "nonexistent.txt").
					Return(nil, errors.New("file not found")).Once()
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:     "Try to download folder",
			boxName:  "testbox",
			filePath: "folder1",
			setupMock: func(storagePath string) {
				box := &models.Box{BaseModel: models.BaseModel{ID: 1}, Name: "testbox"}
				mockService.On("FindBoxByPath", "testbox").Return(box, nil).Once()
				mockService.On("GetFileItem", box, "folder1").
					Return(&models.Item{
						Name: "folder1",
						Type: "folder",
						Path: "folder1",
					}, nil).Once()
			},
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test file if needed
			if tt.fileContent != "" {
				boxDir := filepath.Join(tmpDir, tt.boxName)
				err := os.MkdirAll(boxDir, 0755)
				assert.NoError(t, err)

				err = os.WriteFile(filepath.Join(boxDir, tt.filePath), []byte(tt.fileContent), 0644)
				assert.NoError(t, err)
			}

			tt.setupMock(tmpDir)

			req := httptest.NewRequest(http.MethodGet, "/download/"+tt.boxName+"/"+tt.filePath, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			if tt.expectedCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				assert.Equal(t, tt.fileContent, string(body))
				assert.Equal(t, "attachment; filename=\"test.txt\"", resp.Header.Get("Content-Disposition"))
			}

			mockService.AssertExpectations(t)
		})
	}
}
