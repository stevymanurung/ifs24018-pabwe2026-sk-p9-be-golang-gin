package auth

import (
	"app/internal/database"

	"gorm.io/gorm"
)

type Repository interface {
	Create(user *database.User) error
	FindByUsername(username string) (*database.User, error)
	FindByEmail(email string) (*database.User, error)
	FindByUsernameOrEmail(identifier string) (*database.User, error)
	FindByID(id uint) (*database.User, error)
	Update(user *database.User) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(user *database.User) error {
	return r.db.Create(user).Error
}

func (r *repository) FindByUsername(username string) (*database.User, error) {
	var user database.User
	if err := r.db.Where("LOWER(username) = LOWER(?)", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) FindByEmail(email string) (*database.User, error) {
	var user database.User
	if err := r.db.Where("LOWER(email) = LOWER(?)", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) FindByUsernameOrEmail(identifier string) (*database.User, error) {
	var user database.User
	if err := r.db.Where("LOWER(username) = LOWER(?) OR LOWER(email) = LOWER(?)", identifier, identifier).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) FindByID(id uint) (*database.User, error) {
	var user database.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) Update(user *database.User) error {
	return r.db.Save(user).Error
}
