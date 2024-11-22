package services

import (
	"Boxed/internal/models"
	"Boxed/internal/repository"
	"encoding/json"
)

type BoxService interface {
	CreateBox(name string, properties map[string]interface{}) (*models.Box, error)
	GetBoxByID(id uint) (*models.Box, error)
	UpdateBox(id uint, name string, properties map[string]interface{}) (*models.Box, error)
	DeleteBox(id uint) error
	GetBoxes() ([]models.Box, error)
	GetBoxByPath(path string) (*models.Box, error)
}

type boxServiceImpl struct {
	boxRepo repository.BoxRepository
}

func NewBoxService(boxRepo repository.BoxRepository) BoxService {
	return &boxServiceImpl{boxRepo: boxRepo}
}

func (s *boxServiceImpl) CreateBox(name string, properties map[string]interface{}) (*models.Box, error) {
	propertiesJSON, _ := json.Marshal(properties)
	box := &models.Box{Name: name, Properties: propertiesJSON}
	if err := s.boxRepo.Create(box); err != nil {
		return nil, err
	}
	return box, nil
}

func (s *boxServiceImpl) GetBoxByID(id uint) (*models.Box, error) {
	return s.boxRepo.FindByID(id)
}
func (s *boxServiceImpl) GetBoxByPath(path string) (*models.Box, error) {
	return s.boxRepo.FindByName(path)
}
func (s *boxServiceImpl) UpdateBox(id uint, name string, properties map[string]interface{}) (*models.Box, error) {
	box, err := s.boxRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	box.Name = name
	box.Properties, _ = json.Marshal(properties)
	if err := s.boxRepo.Update(box); err != nil {
		return nil, err
	}
	return box, nil
}

func (s *boxServiceImpl) DeleteBox(id uint) error {
	return s.boxRepo.Delete(id)
}

func (s *boxServiceImpl) GetBoxes() ([]models.Box, error) {
	boxes, err := s.boxRepo.FindAll()
	if err != nil {
		return nil, err
	}
	return boxes, nil
}
