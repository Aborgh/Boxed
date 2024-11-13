package repository

import (
	"gorm.io/gorm"
)

type GenericRepositoryImpl[T any] struct {
	db *gorm.DB
}

func NewGenericRepository[T any](db *gorm.DB) GenericRepository[T] {
	return &GenericRepositoryImpl[T]{db: db}
}

func (r *GenericRepositoryImpl[T]) Create(entity *T) error {
	return r.db.Create(entity).Error
}

func (r *GenericRepositoryImpl[T]) FindByID(id uint) (*T, error) {
	var entity T
	err := r.db.First(&entity, id).Error
	return &entity, err
}

func (r *GenericRepositoryImpl[T]) FindAll() ([]T, error) {
	var entities []T
	err := r.db.Find(&entities).Error
	return entities, err
}

func (r *GenericRepositoryImpl[T]) Update(entity *T) error {
	return r.db.Save(entity).Error
}

func (r *GenericRepositoryImpl[T]) Delete(id uint) error {
	var entity T
	return r.db.Delete(&entity, id).Error
}
