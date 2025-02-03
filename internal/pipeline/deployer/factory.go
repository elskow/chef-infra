package deployer

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/pipeline/config"
)

func NewDeployer(config *config.DeployConfig, logger *zap.Logger) (Deployer, error) {
	switch config.Platform {
	case "kubernetes":
		return NewK8sDeployer(config, logger)
	case "static":
		deployer := NewStaticDeployer(config, logger)
		return deployer, nil
	default:
		return nil, fmt.Errorf("unsupported deployment platform: %s", config.Platform)
	}
}
