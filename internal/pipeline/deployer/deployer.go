package deployer

import (
	"context"

	"github.com/elskow/chef-infra/internal/pipeline/types"
)

type Deployer interface {
	Deploy(ctx context.Context, build *types.Build) error
	Rollback(ctx context.Context, build *types.Build) error
	Validate(build *types.Build) error
}
