package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/config"
)

func newTestLogger(t *testing.T) *zap.Logger {
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)
	return logger
}

func newTestConfig() *config.AuthConfig {
	return &config.AuthConfig{
		JWTSecret:            "test-secret-key",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: time.Hour * 24,
		RefreshTokenEnabled:  true,
	}
}

func newTestService(t *testing.T) *Service {
	return NewService(
		newTestConfig(),
		newTestLogger(t),
		newMockRepository(),
	)
}

func newTestHandler(t *testing.T) *Handler {
	return NewHandler(newTestService(t), newTestLogger(t))
}

func newTestServiceWithRepo(t *testing.T, repo Repository) *Service {
	return NewService(
		newTestConfig(),
		newTestLogger(t),
		repo,
	)
}
