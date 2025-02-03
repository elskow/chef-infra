package pipeline

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/pipeline/builder"
	"github.com/elskow/chef-infra/internal/pipeline/config"
	"github.com/elskow/chef-infra/internal/pipeline/deployer"
	"github.com/elskow/chef-infra/internal/pipeline/types"
	"github.com/elskow/chef-infra/internal/pipeline/validator"
)

type Pipeline struct {
	config         *config.PipelineConfig
	builderFactory builder.FactoryInterface
	deployer       deployer.Deployer
	validator      validator.Validator
	logger         *zap.Logger
	builds         map[string]*types.Build
	metrics        *MetricsCollector
	mu             sync.RWMutex
}

func NewPipeline(
	config *config.PipelineConfig,
	builderFactory *builder.Factory,
	deployer deployer.Deployer,
	validator validator.Validator,
	logger *zap.Logger,
) *Pipeline {
	return &Pipeline{
		config:         config,
		builderFactory: builderFactory,
		deployer:       deployer,
		validator:      validator,
		logger:         logger,
		builds:         make(map[string]*types.Build),
		metrics:        NewMetricsCollector(),
	}
}

func (p *Pipeline) StartBuild(ctx context.Context, build *types.Build) error {
	// Validate build configuration
	if err := p.validator.ValidateBuildConfig(build); err != nil {
		return fmt.Errorf("build validation failed: %w", err)
	}

	p.mu.Lock()
	p.builds[build.ID] = build
	p.mu.Unlock()

	go func() {
		if err := p.executeBuild(ctx, build); err != nil {
			p.logger.Error("build failed",
				zap.String("build_id", build.ID),
				zap.Error(err))
			build.Status = types.BuildStatusFailed
			build.ErrorMessage = err.Error()
		}
	}()

	return nil
}

func (p *Pipeline) executeBuild(ctx context.Context, build *types.Build) error {
	// Set initial status
	build.Status = types.BuildStatusBuilding

	// Create context that can be cancelled
	buildCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Store cancel function for possible cancellation
	p.mu.Lock()
	build.CancelFunc = cancel // Updated to use public field name
	p.mu.Unlock()

	// Create build context with cleanup
	buildContext, err := builder.NewBuildContext(p.config.BuildDir, build.ID)
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}
	defer func() {
		if err := buildContext.Cleanup(); err != nil {
			p.logger.Error("cleanup failed",
				zap.String("build_id", build.ID),
				zap.Error(err))
		}
	}()

	// Create necessary directories
	if err := os.MkdirAll(buildContext.BuildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}
	if err := os.MkdirAll(buildContext.ArtifactDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}
	if err := os.MkdirAll(buildContext.CacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Execute build steps with timeouts
	buildTimeout := time.Duration(p.config.DefaultTimeout) * time.Second
	_, timeoutCancel := context.WithTimeout(buildCtx, buildTimeout)
	defer timeoutCancel()

	// Create builder
	builder, err := p.builderFactory.CreateBuilder(build.Framework, &builder.Options{
		WorkDir:     buildContext.BuildDir,
		CacheDir:    buildContext.CacheDir,
		Environment: p.config.NodeJS.EnvVars,
		Timeout:     p.config.DefaultTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}
	defer func() {
		if err := builder.Cleanup(); err != nil {
			p.logger.Error("cleanup failed",
				zap.String("build_id", build.ID),
				zap.Error(err))
		}
	}()

	// Run build
	buildResult, err := builder.Build(ctx, build)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Validate artifact
	if err := p.validator.ValidateArtifact(buildResult.ArtifactPath); err != nil {
		return fmt.Errorf("artifact validation failed: %w", err)
	}

	// Update build status
	build.Status = types.BuildStatusSuccess
	build.ArtifactPath = buildResult.ArtifactPath
	build.ImageID = buildResult.ImageID
	completeTime := time.Now()
	build.CompleteTime = &completeTime

	// Deploy
	if err := p.deployer.Deploy(ctx, build); err != nil {
		if rbErr := p.deployer.Rollback(ctx, build); rbErr != nil {
			p.logger.Error("rollback failed",
				zap.String("build_id", build.ID),
				zap.Error(rbErr))
		}
		return fmt.Errorf("deployment failed: %w", err)
	}

	return nil
}

func (p *Pipeline) CancelBuild(buildID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	build, exists := p.builds[buildID]
	if !exists {
		return fmt.Errorf("build not found: %s", buildID)
	}

	if build.Status != types.BuildStatusBuilding {
		return fmt.Errorf("cannot cancel build with status: %s", build.Status)
	}

	// Cancel the build context if it exists
	if build.CancelFunc != nil {
		build.CancelFunc()
	}

	build.Status = types.BuildStatusCancelled
	completeTime := time.Now()
	build.CompleteTime = &completeTime

	return nil
}

func (p *Pipeline) GetBuild(buildID string) (*types.Build, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	build, exists := p.builds[buildID]
	if !exists {
		return nil, fmt.Errorf("build not found: %s", buildID)
	}

	return build, nil
}
