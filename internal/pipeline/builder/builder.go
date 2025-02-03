package builder

import (
	"context"

	"github.com/elskow/chef-infra/internal/pipeline/types"
)

type Builder interface {
	Build(ctx context.Context, build *types.Build) (*types.BuildResult, error)
	Validate(build *types.Build) error
	Cleanup() error
}
type Options struct {
	WorkDir     string
	CacheDir    string
	Environment map[string]string
	Timeout     int
}
