package repository

type GenericRepository[T any] interface {
	Create(entity *T) error
	FindByID(id uint) (*T, error)
	FindAll() ([]T, error)
	Update(entity *T) error
	Delete(id uint) error
}
