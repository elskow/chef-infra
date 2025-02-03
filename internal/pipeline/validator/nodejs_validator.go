package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elskow/chef-infra/internal/pipeline/config"
	"github.com/elskow/chef-infra/internal/pipeline/types"
)

type PackageJSON struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
	Scripts      map[string]string `json:"scripts"`
	Engines      map[string]string `json:"engines"`
}

type NodeJSValidator struct {
	config *config.NodeJSConfig
}

func NewNodeJSValidator(config *config.NodeJSConfig) *NodeJSValidator {
	return &NodeJSValidator{
		config: config,
	}
}

func (v *NodeJSValidator) ValidateBuildConfig(build *types.Build) error {
	// Validate package.json
	pkgJSON, err := v.readPackageJSON(build)
	if err != nil {
		return err
	}

	// Validate node version compatibility
	if err := v.validateNodeVersion(pkgJSON); err != nil {
		return err
	}

	// Validate build script exists
	if err := v.validateBuildScript(pkgJSON, build); err != nil {
		return err
	}

	return nil
}

func (v *NodeJSValidator) ValidateArtifact(artifactPath string) error {
	// Check if artifact exists
	if _, err := os.Stat(artifactPath); err != nil {
		return fmt.Errorf("artifact not found: %w", err)
	}

	// Validate artifact size
	info, err := os.Stat(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to get artifact info: %w", err)
	}

	maxSize := int64(100 * 1024 * 1024) // 100MB
	if info.Size() > maxSize {
		return fmt.Errorf("artifact size exceeds maximum allowed size")
	}

	return nil
}

func (v *NodeJSValidator) readPackageJSON(build *types.Build) (*PackageJSON, error) {
	pkgPath := filepath.Join(build.BuilderConfig["workDir"].(string), "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("invalid package.json: %w", err)
	}

	return &pkg, nil
}

func (v *NodeJSValidator) validateNodeVersion(pkg *PackageJSON) error {
	if pkg.Engines == nil || pkg.Engines["node"] == "" {
		return nil // No engine constraints specified
	}

	// For now, we only support exact node version matching
	// TODO: Implement semver validation
	for _, allowed := range v.config.AllowedEngines {
		if pkg.Engines["node"] == allowed {
			return nil
		}
	}

	return fmt.Errorf("unsupported node version: %s", pkg.Engines["node"])
}

func (v *NodeJSValidator) validateBuildScript(pkg *PackageJSON, build *types.Build) error {
	if build.BuildCommand == "" {
		return fmt.Errorf("build command is required")
	}

	// Check if the build command exists in package.json scripts
	if _, exists := pkg.Scripts[build.BuildCommand]; !exists {
		return fmt.Errorf("build command '%s' not found in package.json scripts", build.BuildCommand)
	}

	return nil
}
