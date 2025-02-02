package auth

import (
	"github.com/elskow/chef-infra/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NewModule returns the auth module options
func NewModule() fx.Option {
	return fx.Options(
		fx.Provide(
			// Provide repository
			fx.Annotate(
				func(db *gorm.DB) Repository {
					return NewRepository(db)
				},
			),
			// Provide service
			fx.Annotate(
				func(config *config.AppConfig, log *zap.Logger, repo Repository) *Service {
					return NewService(&config.Auth, log, repo)
				},
			),
			// Provide handler
			fx.Annotate(
				func(svc *Service, log *zap.Logger) *Handler {
					return NewHandler(svc, log)
				},
			),
			// Provide middleware
			fx.Annotate(
				func(config *config.AppConfig) *AuthMiddleware {
					return NewAuthMiddleware(&config.Auth)
				},
			),
		),
	)
}
