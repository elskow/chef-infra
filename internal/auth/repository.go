package auth

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
)

type Repository interface {
	CreateUser(user *User) error
	GetUserByUsername(username string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateLoginAttempts(userID uint, failed bool) error
	LockAccount(userID uint, duration time.Duration) error
	UnlockAccount(userID uint) error
	VerifyEmail(userID uint) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	// Auto-migrate the user model
	if err := db.AutoMigrate(&User{}); err != nil {
		panic(err)
	}
	return &repository{db: db}
}

func (r *repository) CreateUser(user *User) error {
	return r.db.Create(user).Error
}

func (r *repository) GetUserByUsername(username string) (*User, error) {
	var user User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetUserByEmail(email string) (*User, error) {
	var user User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) UpdateLoginAttempts(userID uint, failed bool) error {
	updates := map[string]interface{}{
		"last_login_attempt": time.Now(),
	}

	if failed {
		updates["failed_login_count"] = gorm.Expr("failed_login_count + 1")
	} else {
		updates["failed_login_count"] = 0
	}

	return r.db.Model(&User{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *repository) LockAccount(userID uint, duration time.Duration) error {
	lockUntil := time.Now().Add(duration)
	return r.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"locked":     true,
		"lock_until": lockUntil,
	}).Error
}

func (r *repository) UnlockAccount(userID uint) error {
	return r.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"locked":             false,
		"lock_until":         nil,
		"failed_login_count": 0,
	}).Error
}

func (r *repository) VerifyEmail(userID uint) error {
	return r.db.Model(&User{}).Where("id = ?", userID).Update("email_verified", true).Error
}
