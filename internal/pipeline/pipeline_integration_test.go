package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/elskow/chef-infra/internal/pipeline/builder"
	"github.com/elskow/chef-infra/internal/pipeline/config"
	"github.com/elskow/chef-infra/internal/pipeline/deployer"
	"github.com/elskow/chef-infra/internal/pipeline/types"
	"github.com/elskow/chef-infra/internal/pipeline/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPipelineIntegration(t *testing.T) {
	if err := checkDockerAvailable(); err != nil {
		t.Skip("Docker not available:", err)
	}

	// Skip if Docker is not available
	if os.Getenv("SKIP_DOCKER_TESTS") != "" {
		t.Skip("Skipping integration test that requires Docker")
	}

	// Setup test environment
	tmpDir := setupTestEnvironment(t)
	defer os.RemoveAll(tmpDir)

	// Create test configuration
	cfg := createTestConfig(t, tmpDir)

	// Create logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Initialize pipeline components
	pipeline := setupPipeline(t, cfg, logger)

	tests := []struct {
		name        string
		build       *types.Build
		setupFiles  func(string)
		validate    func(*testing.T, *types.Build, error)
		expectError bool
	}{
		{
			name: "successful react build",
			build: &types.Build{
				ID:           "test-react-1",
				ProjectID:    "test-react-app",
				Framework:    "react",
				BuildCommand: "build",
				OutputDir:    "build",
				BuilderConfig: map[string]interface{}{
					"sourceDir": filepath.Join(tmpDir, "test-react-app"),
					"workDir":   filepath.Join(tmpDir, "test-react-app"),
				},
			},
			setupFiles: func(dir string) {
				createValidReactProject(t, dir)
			},
			validate: func(t *testing.T, build *types.Build, err error) {
				if err != nil {
					t.Logf("Build error: %v", err)
				}
				assert.Equal(t, types.BuildStatusSuccess, build.Status)
				assert.NotEmpty(t, build.ArtifactPath, "Artifact path should not be empty")
				assert.NotEmpty(t, build.ImageID, "Image ID should not be empty")
				assert.NotNil(t, build.CompleteTime, "Complete time should not be nil")
			},
			expectError: false,
		},
		{
			name: "invalid package.json",
			build: &types.Build{
				ID:           "test-invalid-1",
				ProjectID:    "test-invalid-app",
				Framework:    "react",
				BuildCommand: "build",
				OutputDir:    "build",
				BuilderConfig: map[string]interface{}{
					"sourceDir": filepath.Join(tmpDir, "invalid-app"),
					"workDir":   filepath.Join(tmpDir, "invalid-app"),
				},
			},
			setupFiles: func(dir string) {
				// Create invalid package.json
				err := os.MkdirAll(dir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(dir, "package.json"), []byte("{invalid json}"), 0644)
				require.NoError(t, err)
			},
			validate: func(t *testing.T, build *types.Build, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid package.json")
			},
			expectError: true,
		},
		{
			name: "missing build command",
			build: &types.Build{
				ID:           "test-missing-cmd-1",
				ProjectID:    "test-missing-cmd-app",
				Framework:    "react",
				BuildCommand: "nonexistent",
				OutputDir:    "build",
				BuilderConfig: map[string]interface{}{
					"sourceDir": filepath.Join(tmpDir, "missing-cmd-app"),
					"workDir":   filepath.Join(tmpDir, "missing-cmd-app"),
				},
			},
			setupFiles: func(dir string) {
				createReactProjectWithoutBuildScript(t, dir)
			},
			validate: func(t *testing.T, build *types.Build, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "build command 'nonexistent' not found")
			},
			expectError: true,
		},
	}

	// Clean up any leftover resources from previous test runs
	cleanupDockerResources(t)
	defer cleanupDockerResources(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context without timeout for Docker operations
			ctx := context.Background()

			projectDir := filepath.Join(tmpDir, tt.build.ProjectID)
			err := os.MkdirAll(projectDir, 0755)
			require.NoError(t, err)

			tt.setupFiles(projectDir)

			// Verify files exist
			if _, err := os.Stat(filepath.Join(projectDir, "package.json")); err != nil {
				t.Fatalf("package.json not created: %v", err)
			}

			// Update build config with absolute paths
			tt.build.BuilderConfig["sourceDir"] = projectDir
			tt.build.BuilderConfig["workDir"] = projectDir

			err = pipeline.StartBuild(ctx, tt.build)

			if !tt.expectError {
				require.NoError(t, err)

				// Wait for build completion with timeout
				deadline := time.After(10 * time.Minute)
				ticker := time.NewTicker(time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-deadline:
						t.Fatal("build timed out")
					case <-ticker.C:
						build, err := pipeline.GetBuild(tt.build.ID)
						require.NoError(t, err)

						// If build is complete (either success or failure)
						if build.Status != types.BuildStatusBuilding {
							if build.Status == types.BuildStatusFailed {
								t.Logf("Build failed with error: %s", build.ErrorMessage)
							}
							tt.validate(t, build, nil)
							return
						}
					}
				}
			} else {
				tt.validate(t, tt.build, err)
			}
		})
	}
}

func checkDockerAvailable() error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	return err
}

func cleanupDockerResources(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Logf("Failed to create Docker client: %v", err)
		return
	}
	defer cli.Close()

	ctx := context.Background()

	// Stop and remove containers first
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err == nil {
		for _, c := range containers {
			if strings.HasPrefix(c.Image, "chef-") {
				t.Logf("Removing container %s", c.ID[:12])
				err := cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{
					Force:         true,
					RemoveVolumes: true,
				})
				if err != nil {
					t.Logf("Failed to remove container: %v", err)
				}
			}
		}
	}

	// Then remove images
	images, err := cli.ImageList(ctx, image.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("reference", "chef-*"),
		),
	})
	if err == nil {
		for _, img := range images {
			t.Logf("Removing image %s", img.ID[:12])
			_, err := cli.ImageRemove(ctx, img.ID, image.RemoveOptions{
				Force:         true,
				PruneChildren: true,
			})
			if err != nil {
				t.Logf("Failed to remove image: %v", err)
			}
		}
	}
}

func setupTestEnvironment(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "pipeline-integration-test-")
	require.NoError(t, err)

	dirs := []string{
		filepath.Join(tmpDir, "builds"),
		filepath.Join(tmpDir, "artifacts"),
		filepath.Join(tmpDir, "cache"),
		filepath.Join(tmpDir, "deploy"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
	}

	return tmpDir
}

func createTestConfig(_ *testing.T, tmpDir string) *config.PipelineConfig {
	return &config.PipelineConfig{
		BuildDir:       filepath.Join(tmpDir, "builds"),
		ArtifactsDir:   filepath.Join(tmpDir, "artifacts"),
		CacheDir:       filepath.Join(tmpDir, "cache"),
		DefaultTimeout: 300,
		NodeJS: config.NodeJSConfig{
			DefaultVersion: "16",
			AllowedEngines: []string{"14", "16", "18"},
			MaxBuildTime:   1800,
			BuildCache:     true,
			EnvVars: map[string]string{
				"NODE_ENV": "production",
			},
		},
		Deploy: config.DeployConfig{
			Platform:      "static",
			StaticPath:    filepath.Join(tmpDir, "deploy"),
			MaxDeploySize: 100 * 1024 * 1024,
		},
	}
}

func createValidReactProject(t *testing.T, dir string) {
	// Create all required directories
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	// Create necessary React project directories
	dirs := []string{
		filepath.Join(dir, "src"),
		filepath.Join(dir, "public"),
		filepath.Join(dir, "build"),
	}

	for _, d := range dirs {
		err := os.MkdirAll(d, 0755)
		require.NoError(t, err)
	}

	// Create package.json with valid content
	packageJSON := map[string]interface{}{
		"name":    "test-react-app",
		"version": "1.0.0",
		"scripts": map[string]string{
			"build": "react-scripts build",
		},
		"dependencies": map[string]string{
			"react":         "^17.0.2",
			"react-dom":     "^17.0.2",
			"react-scripts": "4.0.3",
		},
		"engines": map[string]string{
			"node": "16",
		},
		"browserslist": []string{
			">0.2%",
			"not dead",
			"not ie <= 11",
			"not op_mini all",
		},
	}

	data, err := json.MarshalIndent(packageJSON, "", "  ")
	require.NoError(t, err)

	// Write package.json
	err = os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)
	require.NoError(t, err)

	// Create public/index.html
	indexHTML := `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>React App</title>
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>`

	err = os.WriteFile(filepath.Join(dir, "public", "index.html"), []byte(indexHTML), 0644)
	require.NoError(t, err)

	// Create src/index.js
	indexJS := `import React from 'react';
import ReactDOM from 'react-dom';
import App from './App';

ReactDOM.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
  document.getElementById('root')
);`

	err = os.WriteFile(filepath.Join(dir, "src", "index.js"), []byte(indexJS), 0644)
	require.NoError(t, err)

	// Create src/App.js
	appJS := `import React from 'react';

function App() {
  return (
    <div>
      <h1>Hello, Chef!</h1>
    </div>
  );
}

export default App;`

	err = os.WriteFile(filepath.Join(dir, "src", "App.js"), []byte(appJS), 0644)
	require.NoError(t, err)
}

func createReactProjectWithoutBuildScript(t *testing.T, dir string) {
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	packageJSON := map[string]interface{}{
		"name":    "test-react-app",
		"version": "1.0.0",
		"scripts": map[string]string{
			"start": "react-scripts start",
		},
		"dependencies": map[string]string{
			"react":         "^17.0.2",
			"react-dom":     "^17.0.2",
			"react-scripts": "4.0.3",
		},
	}

	data, err := json.MarshalIndent(packageJSON, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)
	require.NoError(t, err)
}

func setupPipeline(t *testing.T, cfg *config.PipelineConfig, logger *zap.Logger) *Pipeline {
	// Create builder factory
	builderFactory := builder.NewBuilderFactory(cfg, logger)

	// Create deployer
	deployer, err := deployer.NewDeployer(&cfg.Deploy, logger)
	require.NoError(t, err)

	// Create validator
	validator := validator.NewNodeJSValidator(&cfg.NodeJS)

	// Create pipeline
	pipeline := NewPipeline(cfg, builderFactory, deployer, validator, logger)
	require.NotNil(t, pipeline)

	return pipeline
}
