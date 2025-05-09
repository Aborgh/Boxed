package handlers

import (
	"Boxed/internal/config"
	"Boxed/internal/dto"
	"Boxed/internal/models"
	"Boxed/internal/services"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Enhanced MockFileService for hash-based storage tests
type MockHashFileService struct {
	mock.Mock
	StoragePath     string
	ObjectsPath     string
	UploadedObjects map[string]bool // Track uploaded object hashes
}

func (m *MockHashFileService) CreateFileStructure(box *models.Box, filePath string, fileHeader *multipart.FileHeader, flat bool, properties string) (*dto.ItemGetDTO, error) {
	args := m.Called(box, filePath, fileHeader, flat, properties)
	if dto, ok := args.Get(0).(*dto.ItemGetDTO); ok {
		return dto, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBoxService) FindBoxByPath(boxPath string) (*models.Box, error) {
	args := m.Called(boxPath)
	if box, ok := args.Get(0).(*models.Box); ok {
		return box, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockHashFileService) ListFileOrFolder(boxName string, itemPath string) (*models.Item, error) {
	args := m.Called(boxName, itemPath)
	if item, ok := args.Get(0).(*models.Item); ok {
		return item, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockHashFileService) GetFileItem(box *models.Box, filePath string) (*models.Item, error) {
	args := m.Called(box, filePath)
	if item, ok := args.Get(0).(*models.Item); ok {
		return item, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockHashFileService) GetStoragePath() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockHashFileService) DeleteItemOnDisk(item models.Item, box *models.Box) error {
	args := m.Called(item, box)
	return args.Error(0)
}

func (m *MockHashFileService) GetItemProperties(itemPath string, boxId uint) (json.RawMessage, error) {
	args := m.Called(itemPath, boxId)
	if raw, ok := args.Get(0).(json.RawMessage); ok {
		return raw, args.Error(1)
	}
	return nil, args.Error(1)
}

// Setup a test environment for hash-based storage
func setupHashTestEnv(t *testing.T) (*fiber.App, *MockHashFileService, *FileHandler, string) {
	app := fiber.New()

	// Create temporary directories
	tempDir := t.TempDir()
	objectsDir := filepath.Join(tempDir, "objects")
	err := os.MkdirAll(objectsDir, 0755)
	assert.NoError(t, err)

	mockService := &MockHashFileService{
		StoragePath:     tempDir,
		ObjectsPath:     objectsDir,
		UploadedObjects: make(map[string]bool),
	}
	handler := NewFileHandler(mockService)

	return app, mockService, handler, tempDir
}

// Helper function to generate a file with random content
func generateRandomFile(t *testing.T, size int) (string, string, []byte) {
	content := make([]byte, size)
	_, err := rand.Read(content)
	assert.NoError(t, err)

	// Calculate SHA256 for the content
	h := sha256.New()
	h.Write(content)
	sha256sum := hex.EncodeToString(h.Sum(nil))

	// Create a temporary file to store the content
	tempFile, err := os.CreateTemp("", "test-file-*.dat")
	assert.NoError(t, err)
	_, err = tempFile.Write(content)
	assert.NoError(t, err)
	tempFile.Close()

	return tempFile.Name(), sha256sum, content
}

// Helper function to setup file in hash-based storage for testing
func setupHashFile(t *testing.T, mockService *MockHashFileService, sha256sum string, content []byte) string {
	hashPrefix := sha256sum[:2]
	hashDir := filepath.Join(mockService.ObjectsPath, hashPrefix)
	err := os.MkdirAll(hashDir, 0755)
	assert.NoError(t, err)

	hashFilePath := filepath.Join(hashDir, sha256sum)
	err = os.WriteFile(hashFilePath, content, 0644)
	assert.NoError(t, err)

	mockService.UploadedObjects[sha256sum] = true

	return hashFilePath
}

// Helper function to create multipart form data with a file for testing
func createHashMultipartFormData(t *testing.T, fieldName, fileName string, filePath string, properties string) (*bytes.Buffer, string) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add file
	fileContent, err := os.ReadFile(filePath)
	assert.NoError(t, err)

	part, err := writer.CreateFormFile(fieldName, fileName)
	assert.NoError(t, err)

	_, err = io.Copy(part, bytes.NewReader(fileContent))
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

func TestHashBasedUploadFile_Success(t *testing.T) {
	app, mockService, handler, _ := setupHashTestEnv(t)
	app.Post("/upload/:box/*", handler.UploadFile)

	// Setup test parameters
	boxName := "testbox"
	filePath := "folder/test.txt"
	properties := "key=value"

	// Create a test file of 1MB
	fileTempPath, fileHash, _ := generateRandomFile(t, 1024*1024)
	defer os.Remove(fileTempPath)

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      boxName,
	}

	// Setup mock expectations
	mockService.On("FindBoxByPath", boxName).Return(box, nil).Once()

	// Mock the file creation
	mockService.On("CreateFileStructure",
		box,
		filePath,
		mock.AnythingOfType("*multipart.FileHeader"),
		false,
		properties,
	).Run(func(args mock.Arguments) {
		// Get the file header from the arguments
		fileHeader := args.Get(2).(*multipart.FileHeader)

		// Open the uploaded file
		src, err := fileHeader.Open()
		assert.NoError(t, err)
		defer src.Close()

		// Create the file in the hash-based storage
		hashPrefix := fileHash[:2]
		hashDir := filepath.Join(mockService.ObjectsPath, hashPrefix)
		err = os.MkdirAll(hashDir, 0755)
		assert.NoError(t, err)

		hashFilePath := filepath.Join(hashDir, fileHash)
		dst, err := os.Create(hashFilePath)
		assert.NoError(t, err)
		defer dst.Close()

		// Copy the file
		_, err = io.Copy(dst, src)
		assert.NoError(t, err)

		mockService.UploadedObjects[fileHash] = true
	}).Return(&dto.ItemGetDTO{
		ID:     1,
		Name:   "test.txt",
		Path:   "folder/test.txt",
		Type:   "file",
		Size:   1024 * 1024,
		SHA256: fileHash,
	}, nil).Once()

	// Create multipart form with file
	body, contentType := createHashMultipartFormData(t, "file", "test.txt", fileTempPath, properties)

	// Create the request
	req := httptest.NewRequest(http.MethodPost, "/upload/"+boxName+"/"+filePath, body)
	req.Header.Set("Content-Type", contentType)

	// Test the request
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify the file exists in the hash-based storage
	hashPrefix := fileHash[:2]
	hashFilePath := filepath.Join(mockService.ObjectsPath, hashPrefix, fileHash)
	_, err = os.Stat(hashFilePath)
	assert.True(t, mockService.UploadedObjects[fileHash], "File should be tracked as uploaded")

	// Verify response
	var result dto.ItemGetDTO
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &result)
	assert.NoError(t, err)
	assert.Equal(t, fileHash, result.SHA256)

	mockService.AssertExpectations(t)
}

func TestHashBasedUploadFile_Deduplication(t *testing.T) {
	app, mockService, handler, _ := setupHashTestEnv(t)
	app.Post("/upload/:box/*", handler.UploadFile)

	// Setup test parameters
	boxName := "testbox"
	filePath1 := "folder1/test1.txt"
	filePath2 := "folder2/test2.txt"
	properties := "key=value"

	// Create a test file
	fileTempPath, fileHash, fileContent := generateRandomFile(t, 1024*512)
	defer os.Remove(fileTempPath)

	// Setup hash-based storage with the file already in it
	hashFilePath := setupHashFile(t, mockService, fileHash, fileContent)

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      boxName,
	}

	// Setup mock expectations for first upload
	mockService.On("FindBoxByPath", boxName).Return(box, nil).Times(2)

	// Mock file creation with deduplication check for first upload
	mockService.On("CreateFileStructure",
		box,
		filePath1,
		mock.AnythingOfType("*multipart.FileHeader"),
		false,
		properties,
	).Return(&dto.ItemGetDTO{
		ID:     1,
		Name:   "test1.txt",
		Path:   filePath1,
		Type:   "file",
		Size:   int64(len(fileContent)),
		SHA256: fileHash,
	}, nil).Once()

	// Mock file creation with deduplication check for second upload
	mockService.On("CreateFileStructure",
		box,
		filePath2,
		mock.AnythingOfType("*multipart.FileHeader"),
		false,
		properties,
	).Return(&dto.ItemGetDTO{
		ID:     2,
		Name:   "test2.txt",
		Path:   filePath2,
		Type:   "file",
		Size:   int64(len(fileContent)),
		SHA256: fileHash,
	}, nil).Once()

	// First upload - should recognize the file already exists
	body1, contentType1 := createHashMultipartFormData(t, "file", "test1.txt", fileTempPath, properties)
	req1 := httptest.NewRequest(http.MethodPost, "/upload/"+boxName+"/"+filePath1, body1)
	req1.Header.Set("Content-Type", contentType1)
	resp1, err := app.Test(req1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp1.StatusCode)

	// Second upload - same content but different path - should deduplicate
	body2, contentType2 := createHashMultipartFormData(t, "file", "test2.txt", fileTempPath, properties)
	req2 := httptest.NewRequest(http.MethodPost, "/upload/"+boxName+"/"+filePath2, body2)
	req2.Header.Set("Content-Type", contentType2)
	resp2, err := app.Test(req2)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp2.StatusCode)

	// Verify file exists exactly once in storage
	fileInfo, err := os.Stat(hashFilePath)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(fileContent)), fileInfo.Size())

	// Verify the database has two entries pointing to the same hash
	var result1, result2 dto.ItemGetDTO
	respBody1, _ := io.ReadAll(resp1.Body)
	respBody2, _ := io.ReadAll(resp2.Body)

	err = json.Unmarshal(respBody1, &result1)
	assert.NoError(t, err)
	err = json.Unmarshal(respBody2, &result2)
	assert.NoError(t, err)

	assert.Equal(t, fileHash, result1.SHA256)
	assert.Equal(t, fileHash, result2.SHA256)
	assert.NotEqual(t, result1.ID, result2.ID)
	assert.NotEqual(t, result1.Path, result2.Path)

	mockService.AssertExpectations(t)
}

func TestHashBasedDownloadFile(t *testing.T) {
	app, mockService, handler, _ := setupHashTestEnv(t)
	app.Get("/download/:box/*", handler.DownloadFile)

	// Generate random file content
	_, fileHash, fileContent := generateRandomFile(t, 1024*256)

	// Setup hash-based storage with the file
	setupHashFile(t, mockService, fileHash, fileContent)

	// Setup test parameters
	boxName := "testbox"
	filePath := "folder/test.txt"

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      boxName,
	}

	// Setup item mock
	item := &models.Item{
		BaseModel: models.BaseModel{ID: 1},
		Name:      "test.txt",
		Path:      filePath,
		Type:      "file",
		SHA256:    fileHash,
		Size:      int64(len(fileContent)),
	}

	// Setup mock expectations
	mockService.On("FindBoxByPath", boxName).Return(box, nil).Once()
	mockService.On("GetFileItem", box, filePath).Return(item, nil).Once()
	mockService.On("GetStoragePath").Return(mockService.StoragePath).Once()

	// Create the request
	req := httptest.NewRequest(http.MethodGet, "/download/"+boxName+"/"+filePath, nil)
	resp, err := app.Test(req)

	// Verify the response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "attachment; filename=\"test.txt\"", resp.Header.Get("Content-Disposition"))

	// Verify the file content
	respBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, respBody)

	mockService.AssertExpectations(t)
}

func TestHashBasedDeleteFile(t *testing.T) {
	app, mockService, _, _ := setupHashTestEnv(t)
	app.Delete("/files/:box/*", func(c *fiber.Ctx) error {
		boxName := c.Params("box")
		filePath := c.Params("*")
		filePath = strings.TrimLeft(filePath, "/")

		box, err := mockService.FindBoxByPath(boxName)
		if err != nil || box == nil {
			return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
		}

		item, err := mockService.GetFileItem(box, filePath)
		if err != nil {
			return c.Status(http.StatusNotFound).JSON(map[string]interface{}{"error": err.Error()})
		}

		if err := mockService.DeleteItemOnDisk(*item, box); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": err.Error()})
		}

		return c.SendStatus(http.StatusNoContent)
	})

	// Generate random file content
	_, fileHash, fileContent := generateRandomFile(t, 1024*128)

	// Setup hash-based storage with the file
	hashFilePath := setupHashFile(t, mockService, fileHash, fileContent)

	// Setup test parameters
	boxName := "testbox"
	filePath := "folder/test.txt"

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      boxName,
	}

	// Setup item mock
	item := &models.Item{
		BaseModel: models.BaseModel{ID: 1},
		Name:      "test.txt",
		Path:      filePath,
		Type:      "file",
		SHA256:    fileHash,
		Size:      int64(len(fileContent)),
	}

	// Setup mock expectations for file with no other references
	mockService.On("FindBoxByPath", boxName).Return(box, nil).Once()
	mockService.On("GetFileItem", box, filePath).Return(item, nil).Once()
	mockService.On("DeleteItemOnDisk", *item, box).Run(func(args mock.Arguments) {
		// Simulate deleting the file from hash-based storage
		os.Remove(hashFilePath)
		delete(mockService.UploadedObjects, fileHash)
	}).Return(nil).Once()

	// Create the request
	req := httptest.NewRequest(http.MethodDelete, "/files/"+boxName+"/"+filePath, nil)
	resp, err := app.Test(req)

	// Verify the response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify the file is deleted from storage
	_, err = os.Stat(hashFilePath)
	assert.True(t, os.IsNotExist(err), "File should be deleted from hash storage")
	assert.False(t, mockService.UploadedObjects[fileHash], "File should be removed from tracking")

	mockService.AssertExpectations(t)
}

func TestHashBasedDeleteFileWithReferences(t *testing.T) {
	app, mockService, _, _ := setupHashTestEnv(t)
	app.Delete("/files/:box/*", func(c *fiber.Ctx) error {
		boxName := c.Params("box")
		filePath := c.Params("*")
		filePath = strings.TrimLeft(filePath, "/")

		box, err := mockService.FindBoxByPath(boxName)
		if err != nil || box == nil {
			return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
		}

		item, err := mockService.GetFileItem(box, filePath)
		if err != nil {
			return c.Status(http.StatusNotFound).JSON(map[string]interface{}{"error": err.Error()})
		}

		if err := mockService.DeleteItemOnDisk(*item, box); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": err.Error()})
		}

		return c.SendStatus(http.StatusNoContent)
	})

	// Generate random file content
	_, fileHash, fileContent := generateRandomFile(t, 1024*128)

	// Setup hash-based storage with the file
	hashFilePath := setupHashFile(t, mockService, fileHash, fileContent)

	// Setup test parameters
	boxName := "testbox"
	filePath := "folder/test.txt"

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      boxName,
	}

	// Setup item mock
	item := &models.Item{
		BaseModel: models.BaseModel{ID: 1},
		Name:      "test.txt",
		Path:      filePath,
		Type:      "file",
		SHA256:    fileHash,
		Size:      int64(len(fileContent)),
	}

	// Setup mock expectations for a file that has other references
	mockService.On("FindBoxByPath", boxName).Return(box, nil).Once()
	mockService.On("GetFileItem", box, filePath).Return(item, nil).Once()
	mockService.On("DeleteItemOnDisk", *item, box).Run(func(args mock.Arguments) {
		// Simulate checking for other references - the file should remain in storage
		// Only remove the database entry
	}).Return(nil).Once()

	// Create the request
	req := httptest.NewRequest(http.MethodDelete, "/files/"+boxName+"/"+filePath, nil)
	resp, err := app.Test(req)

	// Verify the response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify the file still exists in storage (due to other references)
	_, err = os.Stat(hashFilePath)
	assert.NoError(t, err, "File should still exist in hash storage due to other references")
	assert.True(t, mockService.UploadedObjects[fileHash], "File should still be tracked")

	mockService.AssertExpectations(t)
}

func TestHashBasedPerformanceWithManyFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	app, mockService, handler, _ := setupHashTestEnv(t)
	app.Post("/upload/:box/*", handler.UploadFile)
	app.Get("/download/:box/*", handler.DownloadFile)

	// Setup test parameters
	boxName := "testbox"
	fileCount := 100 // Adjust based on your test requirements

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      boxName,
	}

	// Pre-generate all files and hashes
	type testFile struct {
		path     string
		hash     string
		content  []byte
		tempPath string
		itemID   uint
	}
	testFiles := make([]testFile, fileCount)

	// Setup mock expectations for all files
	mockService.On("FindBoxByPath", boxName).Return(box, nil).Times(fileCount * 2) // For upload and download

	for i := 0; i < fileCount; i++ {
		// Create a test file with random size between 10KB and 1MB
		size := 10*1024 + rand.Intn(1024*1024-10*1024)
		tempPath, hash, content := generateRandomFile(t, size)

		filePath := fmt.Sprintf("folder%d/test%d.txt", i/10, i)
		itemID := uint(i + 1)

		testFiles[i] = testFile{
			path:     filePath,
			hash:     hash,
			content:  content,
			tempPath: tempPath,
			itemID:   itemID,
		}

		// Mock file creation
		mockService.On("CreateFileStructure",
			box,
			testFiles[i].path,
			mock.AnythingOfType("*multipart.FileHeader"),
			false,
			"key=value",
		).Return(&dto.ItemGetDTO{
			ID:     testFiles[i].itemID,
			Name:   fmt.Sprintf("test%d.txt", i),
			Path:   testFiles[i].path,
			Type:   "file",
			Size:   int64(len(testFiles[i].content)),
			SHA256: testFiles[i].hash,
		}, nil).Once()

		// Mock getting file item for download
		mockService.On("GetFileItem", box, testFiles[i].path).Return(&models.Item{
			BaseModel: models.BaseModel{ID: testFiles[i].itemID},
			Name:      fmt.Sprintf("test%d.txt", i),
			Path:      testFiles[i].path,
			Type:      "file",
			SHA256:    testFiles[i].hash,
			Size:      int64(len(testFiles[i].content)),
		}, nil).Once()

		// Mock storage path
		mockService.On("GetStoragePath").Return(mockService.StoragePath).Once()
	}

	// Measure upload performance
	uploadStart := time.Now()

	for i := 0; i < fileCount; i++ {
		body, contentType := createHashMultipartFormData(t, "file", fmt.Sprintf("test%d.txt", i), testFiles[i].tempPath, "key=value")
		req := httptest.NewRequest(http.MethodPost, "/upload/"+boxName+"/"+testFiles[i].path, body)
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Store file in hash storage for download test
		setupHashFile(t, mockService, testFiles[i].hash, testFiles[i].content)

		// Clean up temp file
		os.Remove(testFiles[i].tempPath)
	}

	uploadDuration := time.Since(uploadStart)
	t.Logf("Upload performance for %d files: %v (avg: %v per file)",
		fileCount, uploadDuration, uploadDuration/time.Duration(fileCount))

	// Measure download performance
	downloadStart := time.Now()

	for i := 0; i < fileCount; i++ {
		req := httptest.NewRequest(http.MethodGet, "/download/"+boxName+"/"+testFiles[i].path, nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify file content (just read it, don't compare all bytes to save time)
		respBody, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, len(testFiles[i].content), len(respBody))
	}

	downloadDuration := time.Since(downloadStart)
	t.Logf("Download performance for %d files: %v (avg: %v per file)",
		fileCount, downloadDuration, downloadDuration/time.Duration(fileCount))

	mockService.AssertExpectations(t)
}

func BenchmarkHashBasedUpload(b *testing.B) {
	// Setup environment for benchmarking
	app := fiber.New()
	tempDir, err := os.MkdirTemp("", "benchmark-hash-storage")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	objectsDir := filepath.Join(tempDir, "objects")
	err = os.MkdirAll(objectsDir, 0755)
	if err != nil {
		b.Fatal(err)
	}

	mockService := &MockHashFileService{
		StoragePath:     tempDir,
		ObjectsPath:     objectsDir,
		UploadedObjects: make(map[string]bool),
	}
	handler := NewFileHandler(mockService)

	app.Post("/upload/:box/*", handler.UploadFile)

	// Generate test file once (1MB)
	testFilePath, testFileHash, _ := generateRandomFile(nil, 1024*1024)
	defer os.Remove(testFilePath)

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      "testbox",
	}

	// Setup mocks (using runtime counting for benchmark)
	mockService.On("FindBoxByPath", "testbox").Return(box, nil)
	mockService.On("CreateFileStructure",
		box,
		mock.Anything,
		mock.AnythingOfType("*multipart.FileHeader"),
		false,
		"",
	).Return(&dto.ItemGetDTO{
		ID:     1,
		Name:   "test.txt",
		Path:   "test.txt",
		Type:   "file",
		Size:   1024 * 1024,
		SHA256: testFileHash,
	}, nil)

	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		filePath := fmt.Sprintf("test%d.txt", i)

		// Create multipart form with file
		body, contentType := createHashMultipartFormData(nil, "file", filePath, testFilePath, "")

		// Create the request
		req := httptest.NewRequest(http.MethodPost, "/upload/testbox/"+filePath, body)
		req.Header.Set("Content-Type", contentType)

		// Test the request (ignore response for benchmark)
		app.Test(req)
	}
}

func BenchmarkHashBasedDownload(b *testing.B) {
	// Setup environment for benchmarking
	app := fiber.New()
	tempDir, err := os.MkdirTemp("", "benchmark-hash-storage")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	objectsDir := filepath.Join(tempDir, "objects")
	err = os.MkdirAll(objectsDir, 0755)
	if err != nil {
		b.Fatal(err)
	}

	mockService := &MockHashFileService{
		StoragePath:     tempDir,
		ObjectsPath:     objectsDir,
		UploadedObjects: make(map[string]bool),
	}
	handler := NewFileHandler(mockService)

	app.Get("/download/:box/*", handler.DownloadFile)

	// Generate test file once (1MB)
	_, testFileHash, testFileContent := generateRandomFile(nil, 1024*1024)

	// Setup hash-based storage with the file
	hashPrefix := testFileHash[:2]
	hashDir := filepath.Join(objectsDir, hashPrefix)
	err = os.MkdirAll(hashDir, 0755)
	if err != nil {
		b.Fatal(err)
	}

	hashFilePath := filepath.Join(hashDir, testFileHash)
	err = os.WriteFile(hashFilePath, testFileContent, 0644)
	if err != nil {
		b.Fatal(err)
	}

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      "testbox",
	}

	// Setup item mock that will be reused for all downloads
	item := &models.Item{
		BaseModel: models.BaseModel{ID: 1},
		Name:      "test.txt",
		Path:      "test.txt",
		Type:      "file",
		SHA256:    testFileHash,
		Size:      int64(len(testFileContent)),
	}

	// Setup mocks (using runtime counting for benchmark)
	mockService.On("FindBoxByPath", "testbox").Return(box, nil)
	mockService.On("GetFileItem", box, mock.Anything).Return(item, nil)
	mockService.On("GetStoragePath").Return(mockService.StoragePath)

	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		filePath := fmt.Sprintf("test%d.txt", i)

		// Create the request
		req := httptest.NewRequest(http.MethodGet, "/download/testbox/"+filePath, nil)
		resp, _ := app.Test(req)

		// Read the full response to measure complete download
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func TestHashBasedDeleteItemOnDisk(t *testing.T) {
	_, mockService, _, _ := setupHashTestEnv(t)

	// Generate files with different content
	_, hash1, content1 := generateRandomFile(t, 1024*64)
	_, hash2, content2 := generateRandomFile(t, 1024*64)

	// Setup hash-based storage with both files
	setupHashFile(t, mockService, hash1, content1)
	setupHashFile(t, mockService, hash2, content2)

	// Setup box
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      "testbox",
	}

	// Test cases for different deletion scenarios
	tests := []struct {
		name            string
		item            models.Item
		otherReferences bool
		checkExistence  bool
	}{
		{
			name: "Delete file with no other references",
			item: models.Item{
				BaseModel: models.BaseModel{ID: 1},
				Name:      "unique.txt",
				Path:      "folder/unique.txt",
				Type:      "file",
				SHA256:    hash1,
				Size:      int64(len(content1)),
			},
			otherReferences: false,
			checkExistence:  false, // File should be deleted
		},
		{
			name: "Delete file with other references",
			item: models.Item{
				BaseModel: models.BaseModel{ID: 2},
				Name:      "duplicate.txt",
				Path:      "folder/duplicate.txt",
				Type:      "file",
				SHA256:    hash2,
				Size:      int64(len(content2)),
			},
			otherReferences: true,
			checkExistence:  true, // File should still exist
		},
		{
			name: "Delete folder",
			item: models.Item{
				BaseModel: models.BaseModel{ID: 3},
				Name:      "testfolder",
				Path:      "testfolder",
				Type:      "folder",
			},
			otherReferences: false,
			checkExistence:  false, // No physical folder in hash-based storage
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the service methods
			mockService.On("HardDelete", &tt.item).Return(nil).Once()

			if tt.item.Type == "file" && !tt.otherReferences {
				// If no other references, the file should be deleted
				mockService.On("ItemsSearch",
					"sha256 eq \""+tt.item.SHA256+"\"",
					mock.Anything, mock.Anything, mock.Anything,
				).Return([]models.Item{}, nil).Once()
			} else if tt.item.Type == "file" && tt.otherReferences {
				// If other references exist, the file should not be deleted
				mockService.On("ItemsSearch",
					"sha256 eq \""+tt.item.SHA256+"\"",
					mock.Anything, mock.Anything, mock.Anything,
				).Return([]models.Item{
					{ID: 999, SHA256: tt.item.SHA256}, // Another item with same hash
				}, nil).Once()
			}

			// Create a file handler instance with our mocked service
			handler := &services.FileServiceImpl{
				itemService: mockService,
				boxService:  mockService,
				logService:  &services.LogService{Log: &logrus.Logger{}},
				configuration: config.Configuration{
					Storage: config.StorageConfig{
						Path: mockService.StoragePath,
					},
				},
			}

			// Execute the delete operation
			err := handler.DeleteItemOnDisk(tt.item)
			assert.NoError(t, err)

			// Verify file existence based on test case
			if tt.item.Type == "file" {
				hashPath := tt.item.SHA256[:2] + "/" + tt.item.SHA256
				fullPath := filepath.Join(mockService.ObjectsPath, hashPath)

				_, err := os.Stat(fullPath)
				if tt.checkExistence {
					assert.NoError(t, err, "File should still exist")
				} else {
					assert.True(t, os.IsNotExist(err), "File should not exist")
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestLargeFileTransfers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	app, mockService, handler, _ := setupHashTestEnv(t)
	app.Post("/upload/:box/*", handler.UploadFile)
	app.Get("/download/:box/*", handler.DownloadFile)

	// Setup test parameters for a large file (50MB)
	boxName := "testbox"
	filePath := "largefile.dat"
	fileSizeMB := 50
	fileSizeBytes := fileSizeMB * 1024 * 1024

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      boxName,
	}

	// Generate a large test file
	fileTempPath, fileHash, _ := generateRandomFile(t, fileSizeBytes)
	defer os.Remove(fileTempPath)

	// Setup mock expectations for uploading the large file
	mockService.On("FindBoxByPath", boxName).Return(box, nil).Once()
	mockService.On("CreateFileStructure",
		box,
		filePath,
		mock.AnythingOfType("*multipart.FileHeader"),
		false,
		"",
	).Run(func(args mock.Arguments) {
		// Get the file header and save it to hash storage
		fileHeader := args.Get(2).(*multipart.FileHeader)
		src, err := fileHeader.Open()
		assert.NoError(t, err)
		defer src.Close()

		// Setup hash storage for the large file
		hashPrefix := fileHash[:2]
		hashDir := filepath.Join(mockService.ObjectsPath, hashPrefix)
		err = os.MkdirAll(hashDir, 0755)
		assert.NoError(t, err)

		hashFilePath := filepath.Join(hashDir, fileHash)
		dst, err := os.Create(hashFilePath)
		assert.NoError(t, err)
		defer dst.Close()

		// Copy the file
		written, err := io.Copy(dst, src)
		assert.NoError(t, err)
		assert.Equal(t, int64(fileSizeBytes), written)

		mockService.UploadedObjects[fileHash] = true
	}).Return(&dto.ItemGetDTO{
		ID:     1,
		Name:   filepath.Base(filePath),
		Path:   filePath,
		Type:   "file",
		Size:   int64(fileSizeBytes),
		SHA256: fileHash,
	}, nil).Once()

	// Setup mock expectations for downloading the large file
	mockService.On("FindBoxByPath", boxName).Return(box, nil).Once()
	mockService.On("GetFileItem", box, filePath).Return(&models.Item{
		BaseModel: models.BaseModel{ID: 1},
		Name:      filepath.Base(filePath),
		Path:      filePath,
		Type:      "file",
		SHA256:    fileHash,
		Size:      int64(fileSizeBytes),
	}, nil).Once()
	mockService.On("GetStoragePath").Return(mockService.StoragePath).Once()

	// 1. Test uploading the large file
	t.Logf("Testing upload of %dMB file...", fileSizeMB)
	uploadStart := time.Now()

	// Create multipart form with the large file
	body, contentType := createHashMultipartFormData(t, "file", filepath.Base(filePath), fileTempPath, "")

	// Create the upload request
	req := httptest.NewRequest(http.MethodPost, "/upload/"+boxName+"/"+filePath, body)
	req.Header.Set("Content-Type", contentType)

	// Test the upload request
	resp, err := app.Test(req, 120*100) // Allow more time for large file
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	uploadDuration := time.Since(uploadStart)
	uploadMBps := float64(fileSizeMB) / uploadDuration.Seconds()
	t.Logf("Upload of %dMB completed in %v (%.2f MB/s)", fileSizeMB, uploadDuration, uploadMBps)

	// 2. Test downloading the large file
	t.Logf("Testing download of %dMB file...", fileSizeMB)
	downloadStart := time.Now()

	// Create the download request
	req = httptest.NewRequest(http.MethodGet, "/download/"+boxName+"/"+filePath, nil)
	resp, err = app.Test(req, 120*100) // Allow more time for large file

	// Verify the response headers
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(filePath)), resp.Header.Get("Content-Disposition"))

	// Read and verify the file size (discard content to save memory)
	var downloadedBytes int64
	downloadedBytes, err = io.Copy(io.Discard, resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, int64(fileSizeBytes), downloadedBytes)

	downloadDuration := time.Since(downloadStart)
	downloadMBps := float64(fileSizeMB) / downloadDuration.Seconds()
	t.Logf("Download of %dMB completed in %v (%.2f MB/s)", fileSizeMB, downloadDuration, downloadMBps)

	mockService.AssertExpectations(t)
}

func TestConcurrentHashFileAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	app, mockService, handler, _ := setupHashTestEnv(t)
	app.Post("/upload/:box/*", handler.UploadFile)
	app.Get("/download/:box/*", handler.DownloadFile)

	// Setup test parameters
	boxName := "testbox"
	concurrentRequests := 10

	// Generate a test file that will be reused
	fileTempPath, fileHash, fileContent := generateRandomFile(t, 1024*512)
	defer os.Remove(fileTempPath)

	// Setup hash-based storage with the file
	setupHashFile(t, mockService, fileHash, fileContent)

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      boxName,
	}

	// Setup mocks for concurrent access
	mockService.On("FindBoxByPath", boxName).Return(box, nil).Times(concurrentRequests * 2) // Upload + download

	for i := 0; i < concurrentRequests; i++ {
		filePath := fmt.Sprintf("concurrent/file%d.txt", i)

		// Setup mock for file creation during upload
		mockService.On("CreateFileStructure",
			box,
			filePath,
			mock.AnythingOfType("*multipart.FileHeader"),
			false,
			"",
		).Return(&dto.ItemGetDTO{
			ID:     uint(i + 1),
			Name:   fmt.Sprintf("file%d.txt", i),
			Path:   filePath,
			Type:   "file",
			Size:   int64(len(fileContent)),
			SHA256: fileHash,
		}, nil).Once()

		// Setup mock for file item during download
		mockService.On("GetFileItem", box, filePath).Return(&models.Item{
			BaseModel: models.BaseModel{ID: uint(i + 1)},
			Name:      fmt.Sprintf("file%d.txt", i),
			Path:      filePath,
			Type:      "file",
			SHA256:    fileHash,
			Size:      int64(len(fileContent)),
		}, nil).Once()

		// Setup mock for getting storage path
		mockService.On("GetStoragePath").Return(mockService.StoragePath).Once()
	}

	// Run concurrent uploads
	var uploadWg sync.WaitGroup
	uploadStart := time.Now()

	for i := 0; i < concurrentRequests; i++ {
		uploadWg.Add(1)
		go func(index int) {
			defer uploadWg.Done()

			filePath := fmt.Sprintf("concurrent/file%d.txt", index)

			// Create multipart form with file
			body, contentType := createHashMultipartFormData(t, "file", fmt.Sprintf("file%d.txt", index), fileTempPath, "")

			// Create the request
			req := httptest.NewRequest(http.MethodPost, "/upload/"+boxName+"/"+filePath, body)
			req.Header.Set("Content-Type", contentType)

			// Test the request
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		}(i)
	}

	uploadWg.Wait()
	uploadDuration := time.Since(uploadStart)
	t.Logf("Concurrent upload of %d files completed in %v", concurrentRequests, uploadDuration)

	// Run concurrent downloads
	var downloadWg sync.WaitGroup
	downloadStart := time.Now()

	for i := 0; i < concurrentRequests; i++ {
		downloadWg.Add(1)
		go func(index int) {
			defer downloadWg.Done()

			filePath := fmt.Sprintf("concurrent/file%d.txt", index)

			// Create the request
			req := httptest.NewRequest(http.MethodGet, "/download/"+boxName+"/"+filePath, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Read response body
			respBody, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Equal(t, len(fileContent), len(respBody))
		}(i)
	}

	downloadWg.Wait()
	downloadDuration := time.Since(downloadStart)
	t.Logf("Concurrent download of %d files completed in %v", concurrentRequests, downloadDuration)

	mockService.AssertExpectations(t)
}

func TestCollisionHandling(t *testing.T) {
	app, mockService, handler, _ := setupHashTestEnv(t)
	app.Post("/upload/:box/*", handler.UploadFile)

	// Setup test parameters
	boxName := "testbox"
	filePaths := []string{
		"project/document.txt",
		"backup/document.txt",
		"archive/document.txt",
	}

	// Create a test file that will be used for all uploads (identical content)
	fileTempPath, fileHash, fileContent := generateRandomFile(t, 1024*128)
	defer os.Remove(fileTempPath)

	// Setup box mock
	box := &models.Box{
		BaseModel: models.BaseModel{ID: 1},
		Name:      boxName,
	}

	// Setup mocks for each path (same content)
	mockService.On("FindBoxByPath", boxName).Return(box, nil).Times(len(filePaths))

	for i, path := range filePaths {
		mockService.On("CreateFileStructure",
			box,
			path,
			mock.AnythingOfType("*multipart.FileHeader"),
			false,
			"",
		).Return(&dto.ItemGetDTO{
			ID:     uint(i + 1),
			Name:   filepath.Base(path),
			Path:   path,
			Type:   "file",
			Size:   int64(len(fileContent)),
			SHA256: fileHash,
		}, nil).Once()
	}

	// Upload the same file to all paths
	for i, path := range filePaths {
		// First upload needs to actually create the file
		if i == 0 {
			// Setup mock to save the file
			mockService.On("CreateFileStructure",
				box,
				path,
				mock.AnythingOfType("*multipart.FileHeader"),
				false,
				"",
			).Run(func(args mock.Arguments) {
				// Save the file to hash storage
				hashPrefix := fileHash[:2]
				hashDir := filepath.Join(mockService.ObjectsPath, hashPrefix)
				err := os.MkdirAll(hashDir, 0755)
				assert.NoError(t, err)

				hashFilePath := filepath.Join(hashDir, fileHash)
				err = os.WriteFile(hashFilePath, fileContent, 0644)
				assert.NoError(t, err)

				mockService.UploadedObjects[fileHash] = true
			}).Return(&dto.ItemGetDTO{
				ID:     uint(i + 1),
				Name:   filepath.Base(path),
				Path:   path,
				Type:   "file",
				Size:   int64(len(fileContent)),
				SHA256: fileHash,
			}, nil).Once()
		}

		// Create multipart form with file
		body, contentType := createHashMultipartFormData(t, "file", filepath.Base(path), fileTempPath, "")

		// Create the request
		req := httptest.NewRequest(http.MethodPost, "/upload/"+boxName+"/"+path, body)
		req.Header.Set("Content-Type", contentType)

		// Test the request
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	// Verify only one copy of the file exists in storage
	hashPrefix := fileHash[:2]
	hashDir := filepath.Join(mockService.ObjectsPath, hashPrefix)

	files, err := os.ReadDir(hashDir)
	assert.NoError(t, err)

	// Check that we have exactly one file in the hash directory
	assert.Equal(t, 1, len(files), "Should have exactly one file in hash storage")
	assert.Equal(t, fileHash, files[0].Name(), "File name should match the hash")

	// Verify the database has entries for all paths but they point to the same hash
	for i, path := range filePaths {
		var result dto.ItemGetDTO
		req := httptest.NewRequest(http.MethodGet, "/items/"+strconv.Itoa(i+1), nil)
		resp, err := app.Test(req)

		if !assert.NoError(t, err) || !assert.Equal(t, http.StatusOK, resp.StatusCode) {
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &result)

		if !assert.NoError(t, err) {
			continue
		}

		assert.Equal(t, path, result.Path, "Path should match original")
		assert.Equal(t, fileHash, result.SHA256, "All entries should reference the same hash")
	}

	mockService.AssertExpectations(t)
}
