package app

import (
	"context"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"os"

	"github.com/elskow/chef-infra/internal/auth"
	"github.com/elskow/chef-infra/internal/server"
)

// Module combines all application modules
func Module() fx.Option {
	return fx.Options(
		// Logger
		fx.Provide(newLogger),

		// Configuration
		fx.Provide(server.LoadConfig),

		// Auth Module
		auth.NewModule(),

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
