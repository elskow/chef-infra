package main

import (
	"os"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/app"
	"github.com/elskow/chef-infra/internal/server"
)

func main() {
	if os.Getenv("APP_ENV") == "" {
		os.Setenv("APP_ENV", "development")
	}

	logger, err := server.NewLogger(os.Getenv("APP_ENV"))
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	app := fx.New(
		app.Module(),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{
				Logger: log,
			}
		}),
	)

	app.Run()
}
