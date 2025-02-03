package pipeline

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/elskow/chef-infra/internal/pipeline/builder"
	"github.com/elskow/chef-infra/internal/pipeline/config"
	"github.com/elskow/chef-infra/internal/pipeline/types"
)

type mockBuilder struct {
	buildCalled    bool
	validateCalled bool
	cleanupCalled  bool
	shouldFail     bool
	delay          time.Duration
}

type mockBuilderFactory struct {
	builder *mockBuilder
}

func (f *mockBuilderFactory) CreateBuilder(framework string, options *builder.Options) (builder.Builder, error) {
	return f.builder, nil
}

func (m *mockBuilder) Build(ctx context.Context, build *types.Build) (*types.BuildResult, error) {
	m.buildCalled = true

	// Set build status to building immediately
	build.Status = types.BuildStatusBuilding

	// Simulate work with delay if specified
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.shouldFail {
		return nil, fmt.Errorf("mock build failure")
	}

	return &types.BuildResult{
		Success:      true,
		ArtifactPath: "/tmp/test-artifact.tar.gz",
		ImageID:      "test-image:latest",
	}, nil
}

func (m *mockBuilder) Validate(build *types.Build) error {
	m.validateCalled = true
	if m.shouldFail {
		return fmt.Errorf("mock validation failure")
	}
	return nil
}

func (m *mockBuilder) Cleanup() error {
	m.cleanupCalled = true
	if m.shouldFail {
		return fmt.Errorf("mock cleanup failure")
	}
	return nil
}

type mockDeployer struct {
	deployCalled   bool
	rollbackCalled bool
	validateCalled bool
	shouldFail     bool
}

func (m *mockDeployer) Deploy(ctx context.Context, build *types.Build) error {
	m.deployCalled = true
	if m.shouldFail {
		return fmt.Errorf("mock deploy failure")
	}
	return nil
}

func (m *mockDeployer) Rollback(ctx context.Context, build *types.Build) error {
	m.rollbackCalled = true
	if m.shouldFail {
		return fmt.Errorf("mock rollback failure")
	}
	return nil
}

func (m *mockDeployer) Validate(build *types.Build) error {
	m.validateCalled = true
	if m.shouldFail {
		return fmt.Errorf("mock validation failure")
	}
	return nil
}

type mockValidator struct {
	validateBuildConfigCalled bool
	validateArtifactCalled    bool
	shouldFail                bool
}

func (m *mockValidator) ValidateBuildConfig(build *types.Build) error {
	m.validateBuildConfigCalled = true
	if m.shouldFail {
		return fmt.Errorf("mock validation failure")
	}
	return nil
}

func (m *mockValidator) ValidateArtifact(artifactPath string) error {
	m.validateArtifactCalled = true
	if m.shouldFail {
		return fmt.Errorf("mock artifact validation failure")
	}
	return nil
}

func setupTestPipeline(t *testing.T) (*Pipeline, *mockBuilder, *mockDeployer, *mockValidator) {
	// Create test config
	cfg := &config.PipelineConfig{
		BuildDir:       "/tmp/test-builds",
		ArtifactsDir:   "/tmp/test-artifacts",
		CacheDir:       "/tmp/test-cache",
		DefaultTimeout: 300,
	}

	// Create test logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create mocks
	mockBuilder := &mockBuilder{}
	mockDeployer := &mockDeployer{}
	mockValidator := &mockValidator{}

	// Create mock builder factory
	mockBuilderFactory := &mockBuilderFactory{
		builder: mockBuilder,
	}

	// Create pipeline
	pipeline := &Pipeline{
		config:         cfg,
		builderFactory: mockBuilderFactory,
		deployer:       mockDeployer,
		validator:      mockValidator,
		logger:         logger,
		builds:         make(map[string]*types.Build),
		metrics:        NewMetricsCollector(),
	}

	return pipeline, mockBuilder, mockDeployer, mockValidator
}

func createTestBuild() *types.Build {
	return &types.Build{
		ID:           "test-build-123",
		ProjectID:    "test-project",
		Framework:    "react",
		BuildCommand: "build",
		OutputDir:    "build",
		Status:       types.BuildStatusPending,
		BuilderConfig: map[string]interface{}{
			"sourceDir": "/tmp/test-source",
		},
	}
}

func TestPipeline_StartBuild(t *testing.T) {
	tests := []struct {
		name       string
		buildMod   func(*types.Build)
		shouldFail bool
		setup      func(*mockBuilder, *mockDeployer, *mockValidator)
		validate   func(*testing.T, *Pipeline, *mockBuilder, *mockDeployer, *mockValidator, error)
	}{
		{
			name: "successful build",
			setup: func(b *mockBuilder, d *mockDeployer, v *mockValidator) {
				b.shouldFail = false
				d.shouldFail = false
				v.shouldFail = false
			},
			validate: func(t *testing.T, p *Pipeline, b *mockBuilder, d *mockDeployer, v *mockValidator, err error) {
				assert.NoError(t, err)
				assert.True(t, v.validateBuildConfigCalled)

				// Wait for async build to complete
				time.Sleep(100 * time.Millisecond)

				build, err := p.GetBuild("test-build-123")
				require.NoError(t, err)
				assert.Equal(t, types.BuildStatusSuccess, build.Status)
			},
		},
		{
			name: "validation failure",
			setup: func(b *mockBuilder, d *mockDeployer, v *mockValidator) {
				v.shouldFail = true
			},
			validate: func(t *testing.T, p *Pipeline, b *mockBuilder, d *mockDeployer, v *mockValidator, err error) {
				assert.Error(t, err)
				assert.True(t, v.validateBuildConfigCalled)
				assert.False(t, b.buildCalled)
				assert.False(t, d.deployCalled)
			},
		},
		{
			name: "build failure",
			setup: func(b *mockBuilder, d *mockDeployer, v *mockValidator) {
				b.shouldFail = true
			},
			validate: func(t *testing.T, p *Pipeline, b *mockBuilder, d *mockDeployer, v *mockValidator, err error) {
				assert.NoError(t, err) // Initial start should succeed

				// Wait for async build to complete
				time.Sleep(100 * time.Millisecond)

				build, err := p.GetBuild("test-build-123")
				require.NoError(t, err)
				assert.Equal(t, types.BuildStatusFailed, build.Status)
				assert.False(t, d.deployCalled)
			},
		},
		{
			name: "deployment failure with rollback",
			setup: func(b *mockBuilder, d *mockDeployer, v *mockValidator) {
				b.shouldFail = false // Builder should succeed
				d.shouldFail = true  // Deployer should fail
			},
			validate: func(t *testing.T, p *Pipeline, b *mockBuilder, d *mockDeployer, v *mockValidator, err error) {
				assert.NoError(t, err) // Initial start should succeed

				// Wait for async build to complete
				time.Sleep(100 * time.Millisecond)

				build, err := p.GetBuild("test-build-123")
				require.NoError(t, err)
				assert.Equal(t, types.BuildStatusFailed, build.Status)
				assert.True(t, b.buildCalled, "Build should have been called")
				assert.True(t, d.deployCalled, "Deploy should have been called")
				assert.True(t, d.rollbackCalled, "Rollback should have been called")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			pipeline, builder, deployer, validator := setupTestPipeline(t)
			if tt.setup != nil {
				tt.setup(builder, deployer, validator)
			}

			// Create test build
			build := createTestBuild()
			if tt.buildMod != nil {
				tt.buildMod(build)
			}

			// Execute
			err := pipeline.StartBuild(context.Background(), build)

			// Validate
			tt.validate(t, pipeline, builder, deployer, validator, err)
		})
	}
}

func TestPipeline_CancelBuild(t *testing.T) {
	pipeline, builder, _, _ := setupTestPipeline(t)

	// Set a delay to ensure we can cancel the build
	builder.delay = 500 * time.Millisecond

	build := createTestBuild()

	// Start build in background
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := pipeline.StartBuild(context.Background(), build)
		require.NoError(t, err)
	}()

	// Wait for build to start
	time.Sleep(100 * time.Millisecond)

	// Verify build is in building state
	buildStatus, err := pipeline.GetBuild(build.ID)
	require.NoError(t, err)
	assert.Equal(t, types.BuildStatusBuilding, buildStatus.Status)

	// Test cancellation
	err = pipeline.CancelBuild(build.ID)
	assert.NoError(t, err)

	// Wait for build to complete
	<-done

	// Verify final build status
	cancelledBuild, err := pipeline.GetBuild(build.ID)
	require.NoError(t, err)
	assert.Equal(t, types.BuildStatusCancelled, cancelledBuild.Status)
	assert.NotNil(t, cancelledBuild.CompleteTime)
}

func TestPipeline_GetBuild(t *testing.T) {
	pipeline, _, _, _ := setupTestPipeline(t)

	// Test non-existent build
	_, err := pipeline.GetBuild("non-existent")
	assert.Error(t, err)

	// Add a build and test retrieval
	build := createTestBuild()
	err = pipeline.StartBuild(context.Background(), build)
	require.NoError(t, err)

	retrievedBuild, err := pipeline.GetBuild(build.ID)
	assert.NoError(t, err)
	assert.Equal(t, build.ID, retrievedBuild.ID)
}

func TestMain(m *testing.M) {
	// Setup
	if err := os.MkdirAll("/tmp/test-source", 0755); err != nil {
		panic(err)
	}

	// Create dummy package.json
	if err := os.WriteFile("/tmp/test-source/package.json", []byte("{}"), 0644); err != nil {
		panic(err)
	}

	// Create test directories
	testDirs := []string{
		"/tmp/test-builds",
		"/tmp/test-artifacts",
		"/tmp/test-cache",
	}
	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(err)
		}
	}

	// Run tests
	code := m.Run()

	// Cleanup
	cleanupDirs := append(testDirs, "/tmp/test-source")
	for _, dir := range cleanupDirs {
		os.RemoveAll(dir)
	}

	os.Exit(code)
}
