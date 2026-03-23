package repository

import (
	"errors"

	"github.com/unadkatdinky/devpulse/internal/models"
	"gorm.io/gorm"
)

// UserRepository handles all database operations for users
type UserRepository struct {
	DB *gorm.DB
}

// NewUserRepository creates a UserRepository with a DB connection
// This is called in main.go and the result is passed to handlers
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// Create inserts a new user row into the users table
// After creation, user.ID and user.CreatedAt are filled in by PostgreSQL
func (r *UserRepository) Create(user *models.User) error {
	result := r.DB.Create(user)
	return result.Error
}

// FindByEmail looks up a user by email address
// Returns (nil, nil) if no user found — not found is not an error
// Returns (nil, err) if something went wrong with the database
// Returns (*user, nil) if found successfully
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User

	result := r.DB.Where("email = ?", email).First(&user)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}

// FindByID looks up a user by their ID
// Used later when we need to get the logged-in user's profile
func (r *UserRepository) FindByID(id uint) (*models.User, error) {
	var user models.User

	result := r.DB.First(&user, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}