package database

import (
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/elskow/chef-infra/internal/config"
)

type Manager struct {
	db     *gorm.DB
	config *config.DatabaseConfig
	logger *zap.Logger
}

func NewManager(config *config.DatabaseConfig, logger *zap.Logger) (*Manager, error) {
	db, err := newDatabase(config)
	if err != nil {
		return nil, err
	}

	return &Manager{
		db:     db,
		config: config,
		logger: logger,
	}, nil
}

func (m *Manager) DB() *gorm.DB {
	return m.db
}

func newDatabase(config *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		config.Host,
		config.User,
		config.Password,
		config.Name,
		config.Port,
		config.SSLMode,
	)

	gormConfig := &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	}

	return gorm.Open(postgres.Open(dsn), gormConfig)
}
