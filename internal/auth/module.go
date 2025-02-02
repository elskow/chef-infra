package auth

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/elskow/chef-infra/internal/config"
)

// NewModule returns the auth module options
func NewModule(db *gorm.DB) fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func() Repository {
					return NewRepository(db)
				},
			),
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
