package pipeline

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/pipeline/builder"
	"github.com/elskow/chef-infra/internal/pipeline/config"
	"github.com/elskow/chef-infra/internal/pipeline/deployer"
	"github.com/elskow/chef-infra/internal/pipeline/validator"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(config *config.PipelineConfig, logger *zap.Logger) (*builder.Factory, error) {
					return builder.NewBuilderFactory(config, logger), nil
				},
			),
			fx.Annotate(
				func(config *config.PipelineConfig, logger *zap.Logger) (deployer.Deployer, error) {
					return deployer.NewDeployer(&config.Deploy, logger)
				},
			),
			fx.Annotate(
				func(config *config.PipelineConfig) validator.Validator {
					return validator.NewNodeJSValidator(&config.NodeJS)
				},
			),
			fx.Annotate(
				func(
					config *config.PipelineConfig,
					builderFactory *builder.Factory,
					deployer deployer.Deployer,
					validator validator.Validator,
					logger *zap.Logger,
				) *Pipeline {
					return NewPipeline(config, builderFactory, deployer, validator, logger)
				},
			),
		),
	)
}
