package builder

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/pipeline/config"
)

type Factory struct {
	config *config.PipelineConfig
	logger *zap.Logger
}

type FactoryInterface interface {
	CreateBuilder(framework string, options *Options) (Builder, error)
}

func NewBuilderFactory(config *config.PipelineConfig, logger *zap.Logger) *Factory {
	return &Factory{
		config: config,
		logger: logger,
	}
}

func (f *Factory) CreateBuilder(framework string, options *Options) (Builder, error) {
	switch framework {
	case "react", "vue", "svelte", "angular":
		builder, err := NewNodeJSBuilder(&f.config.NodeJS, options, f.logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create nodejs builder: %w", err)
		}
		return builder, nil
	default:
		return nil, fmt.Errorf("unsupported framework: %s", framework)
	}
}
