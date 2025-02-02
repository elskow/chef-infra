package database

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/config"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(config *config.AppConfig, logger *zap.Logger) (*Manager, error) {
					return NewManager(&config.Database, logger)
				},
			),
		),
		fx.Invoke(registerHooks),
	)
}

func registerHooks(
	lifecycle fx.Lifecycle,
	manager *Manager,
	logger *zap.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Closing database connections")
			sqlDB, err := manager.db.DB()
			if err != nil {
				return err
			}
			return sqlDB.Close()
		},
	})
}
