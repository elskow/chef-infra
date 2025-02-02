package auth

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID            uint   `gorm:"primaryKey"`
	Username      string `gorm:"uniqueIndex;not null"`
	PasswordHash  string `gorm:"not null"`
	Email         string `gorm:"uniqueIndex;not null"`
	EmailVerified bool   `gorm:"default:false"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string {
	return "users"
}
