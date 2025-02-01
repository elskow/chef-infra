package auth

import (
	"github.com/elskow/chef-infra/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewModule returns the auth module options
func NewModule() fx.Option {
	return fx.Options(
		fx.Provide(
			// Provide the service
			fx.Annotate(
				func(config *config.AppConfig, log *zap.Logger) *Service {
					return NewService(&config.Auth, log)
				},
			),
			// Provide the handler
			fx.Annotate(
				func(svc *Service, log *zap.Logger) *Handler {
					return NewHandler(svc, log)
				},
			),
		),
	)
}
