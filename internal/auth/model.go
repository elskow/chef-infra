package auth

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID               uint   `gorm:"primaryKey"`
	Username         string `gorm:"uniqueIndex;not null"`
	PasswordHash     string `gorm:"not null"`
	Email            string `gorm:"uniqueIndex;not null"`
	EmailVerified    bool   `gorm:"default:false"`
	FailedLoginCount int    `gorm:"default:0"`
	LastLoginAttempt *time.Time
	Locked           bool `gorm:"default:false"`
	LockUntil        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string {
	return "users"
}
