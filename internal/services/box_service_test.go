package services

import (
	"Boxed/internal/models"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockBoxRepository struct {
	mock.Mock
}

func (m *MockBoxRepository) Create(box *models.Box) error {
	args := m.Called(box)
	return args.Error(0)
}

func (m *MockBoxRepository) FindByID(id uint) (*models.Box, error) {
	args := m.Called(id)
	box, ok := args.Get(0).(*models.Box)
	if !ok {
		return nil, args.Error(1)
	}
	return box, args.Error(1)
}

func (m *MockBoxRepository) Update(box *models.Box) error {
	args := m.Called(box)
	return args.Error(0)
}

func (m *MockBoxRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *MockBoxRepository) FindAll() ([]models.Box, error) {
	args := m.Called()
	return args.Get(0).([]models.Box), args.Error(1)
}

func TestBoxService_GetBoxes(t *testing.T) {
	mockRepo := new(MockBoxRepository)
	service := NewBoxService(mockRepo)

	boxes := []models.Box{
		{BaseModel: models.BaseModel{ID: 1}, Name: "Box 1", Path: "/path/box1"},
		{BaseModel: models.BaseModel{ID: 1}, Name: "Box 2", Path: "/path/box2"},
	}

	// Ställ in mock-responsen för FindAll
	mockRepo.On("FindAll").Return(boxes, nil)

	// Anropa GetBoxes och kontrollera resultatet
	allBoxes, err := service.GetBoxes()

	assert.NoError(t, err)
	assert.Len(t, allBoxes, 2)
	assert.Equal(t, "Box 1", allBoxes[0].Name)
	assert.Equal(t, "Box 2", allBoxes[1].Name)
	mockRepo.AssertExpectations(t)
}
func TestBoxService_CreateBox(t *testing.T) {
	mockRepo := new(MockBoxRepository)
	service := NewBoxService(mockRepo)

	properties := map[string]interface{}{"key": "value"}
	propertiesJSON, _ := json.Marshal(properties)
	box := &models.Box{Name: "Test Box", Path: "/path/to/box", Properties: propertiesJSON}

	mockRepo.On("Create", box).Return(nil)

	createdBox, err := service.CreateBox("Test Box", "/path/to/box", properties)

	assert.NoError(t, err)
	assert.Equal(t, "Test Box", createdBox.Name)
	assert.Equal(t, "/path/to/box", createdBox.Path)
	mockRepo.AssertExpectations(t)
}

func TestBoxService_GetBoxByID(t *testing.T) {
	mockRepo := new(MockBoxRepository)
	service := NewBoxService(mockRepo)

	box := &models.Box{BaseModel: models.BaseModel{ID: 1}, Name: "Test Box", Path: "/path/to/box"}
	mockRepo.On("FindByID", uint(1)).Return(box, nil)

	foundBox, err := service.GetBoxByID(1)

	assert.NoError(t, err)
	assert.Equal(t, uint(1), foundBox.ID)
	assert.Equal(t, "Test Box", foundBox.Name)
	mockRepo.AssertExpectations(t)
}

func TestBoxService_UpdateBox(t *testing.T) {
	mockRepo := new(MockBoxRepository)
	service := NewBoxService(mockRepo)

	box := &models.Box{BaseModel: models.BaseModel{ID: 1}, Name: "Original Box", Path: "/original/path"}
	updatedProperties := map[string]interface{}{"newKey": "newValue"}
	updatedPropertiesJSON, _ := json.Marshal(updatedProperties)

	mockRepo.On("FindByID", uint(1)).Return(box, nil)
	mockRepo.On("Update", box).Return(nil)

	updatedBox, err := service.UpdateBox(1, "Updated Box", "/updated/path", updatedProperties)

	assert.NoError(t, err)
	assert.Equal(t, "Updated Box", updatedBox.Name)
	assert.Equal(t, "/updated/path", updatedBox.Path)
	assert.EqualValues(t, updatedPropertiesJSON, updatedBox.Properties)
	mockRepo.AssertExpectations(t)
}

func TestBoxService_DeleteBox(t *testing.T) {
	mockRepo := new(MockBoxRepository)
	service := NewBoxService(mockRepo)

	mockRepo.On("Delete", uint(1)).Return(nil)

	err := service.DeleteBox(1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
