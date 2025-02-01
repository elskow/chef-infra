package auth

import (
	"testing"
	"time"

	"github.com/elskow/chef-infra/internal/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func newTestLogger(t *testing.T) *zap.Logger {
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)
	return logger
}

func newTestConfig() *config.AuthConfig {
	return &config.AuthConfig{
		JWTSecret:           "test-secret-key",
		TokenExpiration:     time.Hour,
		RefreshTokenEnabled: true,
	}
}

func newTestService(t *testing.T) *Service {
	return NewService(newTestConfig(), newTestLogger(t))
}
