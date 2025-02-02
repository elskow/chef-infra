package migration

import (
	"context"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/config"
)

// Module provides migration-related dependencies
func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(config *config.AppConfig) (*Migrator, error) {
					return NewMigrator(&config.Database)
				},
			),
		),
		fx.Invoke(registerHooks),
	)
}

func registerHooks(
	lifecycle fx.Lifecycle,
	migrator *Migrator,
	logger *zap.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Get current version before migration
			currentVersion, err := migrator.GetCurrentVersion()
			if err != nil {
				return fmt.Errorf("failed to get current migration version: %w", err)
			}

			// Get latest available version
			latestVersion, err := migrator.GetLatestVersion()
			if err != nil {
				return fmt.Errorf("failed to get latest migration version: %w", err)
			}

			logger.Info("Database migration status",
				zap.Int64("current_version", currentVersion),
				zap.Int64("latest_version", latestVersion))

			// If versions don't match, perform migration
			if currentVersion != latestVersion {
				if currentVersion > latestVersion {
					// Need to downgrade
					logger.Info("Downgrading database schema",
						zap.Int64("from_version", currentVersion),
						zap.Int64("to_version", latestVersion))

					if err := migrator.DownTo(latestVersion); err != nil {
						return fmt.Errorf("failed to downgrade database: %w", err)
					}
				} else {
					// Need to upgrade
					logger.Info("Upgrading database schema",
						zap.Int64("from_version", currentVersion),
						zap.Int64("to_version", latestVersion))

					if err := migrator.Up(); err != nil {
						return fmt.Errorf("failed to upgrade database: %w", err)
					}
				}
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return migrator.Close()
		},
	})
}
