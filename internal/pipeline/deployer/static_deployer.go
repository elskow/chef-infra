package deployer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/pipeline/config"
	"github.com/elskow/chef-infra/internal/pipeline/types"
)

const (
	defaultMaxDeploySize = 100 * 1024 * 1024 // 100MB default
)

type StaticDeployer struct {
	config *config.DeployConfig
	logger *zap.Logger
}

func NewStaticDeployer(config *config.DeployConfig, logger *zap.Logger) *StaticDeployer {
	if config.StaticPath == "" {
		logger.Warn("static_path not configured, using default /var/www/html")
		config.StaticPath = "/var/www/html"
	}
	if config.MaxDeploySize == 0 {
		logger.Warn("max_deploy_size not configured, using default 100MB")
		config.MaxDeploySize = defaultMaxDeploySize
	}

	return &StaticDeployer{
		config: config,
		logger: logger,
	}
}

func (d *StaticDeployer) Deploy(_ context.Context, build *types.Build) error {
	// Ensure static path exists
	if err := os.MkdirAll(d.config.StaticPath, 0755); err != nil {
		return fmt.Errorf("failed to create static directory: %w", err)
	}

	targetDir := filepath.Join(d.config.StaticPath, build.ProjectID)
	d.logger.Info("deploying to static directory",
		zap.String("target", targetDir),
		zap.String("project", build.ProjectID))

	// Create backup of current deployment
	if err := d.createBackup(targetDir, build); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Extract artifact to target directory
	if err := d.extractArtifact(build.ArtifactPath, targetDir); err != nil {
		return fmt.Errorf("failed to extract artifact: %w", err)
	}

	d.logger.Info("static deployment completed",
		zap.String("project", build.ProjectID),
		zap.String("location", targetDir))

	return nil
}

func (d *StaticDeployer) Rollback(_ context.Context, build *types.Build) error {
	targetDir := filepath.Join(d.config.StaticPath, build.ProjectID)
	backupPath := filepath.Join(d.config.StaticPath, "backups", fmt.Sprintf("%s.tar.gz", build.ID))

	d.logger.Info("rolling back deployment",
		zap.String("project", build.ProjectID),
		zap.String("backup", backupPath))

	if err := d.extractArtifact(backupPath, targetDir); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

func (d *StaticDeployer) Validate(build *types.Build) error {
	if build.ArtifactPath == "" {
		return fmt.Errorf("artifact path is required")
	}

	info, err := os.Stat(build.ArtifactPath)
	if err != nil {
		return fmt.Errorf("failed to stat artifact: %w", err)
	}

	if info.Size() > d.config.MaxDeploySize {
		return fmt.Errorf("artifact size %d exceeds maximum allowed size %d", info.Size(), d.config.MaxDeploySize)
	}

	return nil
}

func (d *StaticDeployer) createBackup(sourceDir string, build *types.Build) error {
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		d.logger.Info("no existing deployment to backup",
			zap.String("project", build.ProjectID))
		return nil
	}

	backupDir := filepath.Join(d.config.StaticPath, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s.tar.gz", build.ID))
	cmd := exec.Command("tar", "-czf", backupPath, "-C", sourceDir, ".")

	d.logger.Info("creating backup",
		zap.String("project", build.ProjectID),
		zap.String("backup_path", backupPath))

	return cmd.Run()
}

func (d *StaticDeployer) extractArtifact(artifactPath, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	cmd := exec.Command("tar", "-xzf", artifactPath, "-C", targetDir)
	d.logger.Info("extracting artifact",
		zap.String("source", artifactPath),
		zap.String("target", targetDir))

	return cmd.Run()
}
