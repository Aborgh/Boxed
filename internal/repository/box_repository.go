package repository

import (
	"Boxed/internal/models"
	"errors"
	"gorm.io/gorm"
)

type BoxRepository interface {
	GenericRepository[models.Box]
	FindByName(path string) (*models.Box, error)
}

type BoxRepositoryImpl[T models.Box] struct {
	GenericRepository[models.Box]
	db *gorm.DB
}

func NewBoxRepository(db *gorm.DB) BoxRepository {
	return &BoxRepositoryImpl[models.Box]{
		GenericRepository: NewGenericRepository[models.Box](db),
		db:                db,
	}
}

func (r *BoxRepositoryImpl[T]) FindByName(path string) (*models.Box, error) {
	var box models.Box
	err := r.db.Where("name = ?", path).First(&box).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &box, nil
}