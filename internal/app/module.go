package app

import (
	"context"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/auth"
	"github.com/elskow/chef-infra/internal/config"
	"github.com/elskow/chef-infra/internal/database"
	"github.com/elskow/chef-infra/internal/migration"
	"github.com/elskow/chef-infra/internal/server"
)

func Module() fx.Option {
	return fx.Options(
		// Logger
		fx.Provide(newLogger),

		// Configuration
		fx.Provide(server.LoadConfig),

		// Database
		database.Module(),

		// Migration (after database is set up)
		migration.Module(),

		// Auth Module
		fx.Provide(
			// Provide AuthMiddleware
			fx.Annotate(
				func(config *config.AppConfig) *auth.AuthMiddleware {
					return auth.NewAuthMiddleware(&config.Auth)
				},
			),
			// Provide AuthService
			fx.Annotate(
				func(config *config.AppConfig, log *zap.Logger, dbm *database.Manager) *auth.Service {
					return auth.NewService(&config.Auth, log, auth.NewRepository(dbm.DB()))
				},
			),
			// Provide AuthHandler
			fx.Annotate(
				func(svc *auth.Service, log *zap.Logger) *auth.Handler {
					return auth.NewHandler(svc, log)
				},
			),
		),

		// Server
		fx.Provide(server.NewServer),

		// Start the server
		fx.Invoke(registerHooks),
	)
}

func newLogger() (*zap.Logger, error) {
	env := os.Getenv("APP_ENV")
	return server.NewLogger(env)
}

func registerHooks(
	lifecycle fx.Lifecycle,
	srv *server.Server,
	log *zap.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := srv.Start(); err != nil {
					log.Error("failed to start server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("shutting down server...")
			srv.Stop()
			return nil
		},
	})
}
