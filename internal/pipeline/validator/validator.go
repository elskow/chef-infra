package validator

import (
	"github.com/elskow/chef-infra/internal/pipeline/types"
)

type Validator interface {
	ValidateBuildConfig(build *types.Build) error
	ValidateArtifact(artifactPath string) error
}
