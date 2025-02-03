package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/elskow/chef-infra/internal/pipeline/config"
	pipelinetypes "github.com/elskow/chef-infra/internal/pipeline/types"
	"go.uber.org/zap"
)

type NodeJSBuilder struct {
	config    *config.NodeJSConfig
	options   *Options
	logger    *zap.Logger
	dockerCli *client.Client
}

func NewNodeJSBuilder(config *config.NodeJSConfig, options *Options, logger *zap.Logger) (*NodeJSBuilder, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &NodeJSBuilder{
		config:    config,
		options:   options,
		logger:    logger,
		dockerCli: cli,
	}, nil
}

func (b *NodeJSBuilder) Build(ctx context.Context, build *pipelinetypes.Build) (*pipelinetypes.BuildResult, error) {
	b.logger.Info("starting nodejs build in docker",
		zap.String("project", build.ProjectID),
		zap.String("commit", build.CommitHash))

	// Create build directory and prepare files
	buildDir := filepath.Join(b.options.WorkDir, build.ID)
	if err := b.prepareBuildDirectory(buildDir, build); err != nil {
		return nil, fmt.Errorf("failed to prepare build directory: %w", err)
	}

	// Create Dockerfile
	if err := b.createDockerfile(buildDir, build); err != nil {
		return nil, fmt.Errorf("failed to create dockerfile: %w", err)
	}

	imageTag := fmt.Sprintf("chef-%s:%s", build.ProjectID, build.ID)
	if build.CommitHash != "" {
		imageTag = fmt.Sprintf("chef-%s:%s", build.ProjectID, build.CommitHash)
	}

	// Build Docker image with proper error handling
	buildOpts := dockertypes.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{imageTag},
		Remove:     true,
		// Add build args if needed
		BuildArgs: map[string]*string{
			"NODE_ENV": &[]string{"production"}[0],
		},
	}

	buildContext := b.createBuildContext(buildDir)
	if buildContext == nil {
		return nil, fmt.Errorf("failed to create build context")
	}

	resp, err := b.dockerCli.ImageBuild(ctx, buildContext, buildOpts)
	if err != nil {
		return nil, fmt.Errorf("docker build failed: %w", err)
	}
	defer resp.Body.Close()

	// Process build output
	if err := b.processBuildOutput(resp.Body); err != nil {
		return nil, err
	}

	// Create artifact from build output
	if err := b.createArtifactFromContainer(ctx, build); err != nil {
		return nil, fmt.Errorf("failed to create artifact: %w", err)
	}

	return &pipelinetypes.BuildResult{
		Success:      true,
		ArtifactPath: filepath.Join(b.options.WorkDir, "artifacts", fmt.Sprintf("%s.tar.gz", build.ID)),
		ImageID:      imageTag,
	}, nil
}

func (b *NodeJSBuilder) Validate(build *pipelinetypes.Build) error {
	// Validate required fields
	if build.BuildCommand == "" {
		return fmt.Errorf("build command is required")
	}
	if build.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}
	if build.BuilderConfig == nil {
		return fmt.Errorf("builder configuration is required")
	}
	sourceDir, ok := build.BuilderConfig["sourceDir"].(string)
	if !ok || sourceDir == "" {
		return fmt.Errorf("source directory is required in builder configuration")
	}

	// Validate source directory exists
	if _, err := os.Stat(sourceDir); err != nil {
		return fmt.Errorf("source directory does not exist: %w", err)
	}

	// Validate package.json exists
	pkgPath := filepath.Join(sourceDir, "package.json")
	if _, err := os.Stat(pkgPath); err != nil {
		return fmt.Errorf("package.json not found in source directory: %w", err)
	}

	return nil
}

func (b *NodeJSBuilder) createDockerfile(buildDir string, build *pipelinetypes.Build) error {
	dockerfile := fmt.Sprintf(`
FROM node:%s-alpine

WORKDIR /app

# Add build dependencies
RUN apk add --no-cache python3 make g++

# Copy package files
COPY package*.json ./
RUN npm install

# Copy source files
COPY . .

# Set environment variables
ENV NODE_ENV=production
ENV CI=true

# Build the application
RUN npm run %s

FROM nginx:alpine
COPY --from=0 /app/%s /usr/share/nginx/html
EXPOSE 80
`, b.config.DefaultVersion, build.BuildCommand, build.OutputDir)

	return os.WriteFile(filepath.Join(buildDir, "Dockerfile"), []byte(dockerfile), 0644)
}

func (b *NodeJSBuilder) createBuildContext(buildDir string) io.Reader {
	tar, err := archive.TarWithOptions(buildDir, &archive.TarOptions{})
	if err != nil {
		return nil
	}
	return tar
}

func (b *NodeJSBuilder) processBuildOutput(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	for {
		var message struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
			Status string `json:"status"`
			ID     string `json:"id"`
		}

		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if message.Error != "" {
			return fmt.Errorf("docker build error: %s", message.Error)
		}

		// Log all types of Docker messages
		if message.Stream != "" {
			b.logger.Debug("docker build output", zap.String("output", strings.TrimSpace(message.Stream)))
		}
		if message.Status != "" {
			b.logger.Debug("docker status",
				zap.String("status", message.Status),
				zap.String("id", message.ID))
		}
	}
}

func (b *NodeJSBuilder) prepareBuildDirectory(buildDir string, build *pipelinetypes.Build) error {
	// Create build directory
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// Copy source files to build directory
	sourceDir := build.BuilderConfig["sourceDir"].(string)
	if err := b.copySourceFiles(sourceDir, buildDir); err != nil {
		return fmt.Errorf("failed to copy source files: %w", err)
	}

	return nil
}

func (b *NodeJSBuilder) copySourceFiles(sourceDir, targetDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip node_modules and .git
		if info.IsDir() && (info.Name() == "node_modules" || info.Name() == ".git") {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return b.copyFile(path, targetPath)
	})
}

func (b *NodeJSBuilder) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer target.Close()

	_, err = io.Copy(target, source)
	return err
}

func (b *NodeJSBuilder) createArtifactFromContainer(ctx context.Context, build *pipelinetypes.Build) error {
	imageTag := fmt.Sprintf("chef-%s:%s", build.ProjectID, build.ID)
	if build.CommitHash != "" {
		imageTag = fmt.Sprintf("chef-%s:%s", build.ProjectID, build.CommitHash)
	}

	// Verify image exists before creating container
	_, _, err := b.dockerCli.ImageInspectWithRaw(ctx, imageTag)
	if err != nil {
		return fmt.Errorf("image not found: %s: %w", imageTag, err)
	}

	containerConfig := &container.Config{
		Image: imageTag,
	}

	containerID, err := b.createContainer(ctx, containerConfig)
	if err != nil {
		return err
	}
	defer func() {
		err := b.dockerCli.ContainerRemove(ctx, containerID, container.RemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})

		if err != nil {
			b.logger.Warn("failed to remove container", zap.String("container", containerID), zap.Error(err))
		}
	}()

	// Copy the built files from the container
	artifactDir := filepath.Join(b.options.WorkDir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return err
	}

	reader, _, err := b.dockerCli.CopyFromContainer(ctx, containerID, "/usr/share/nginx/html")
	if err != nil {
		return err
	}
	defer reader.Close()

	artifactPath := filepath.Join(artifactDir, fmt.Sprintf("%s.tar.gz", build.ID))
	outFile, err := os.Create(artifactPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, reader)
	return err
}

func (b *NodeJSBuilder) createContainer(ctx context.Context, config *container.Config) (string, error) {
	config.Image = b.getImageTag(config.Image)
	resp, err := b.dockerCli.ContainerCreate(ctx, config, nil, nil, nil, "")
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (b *NodeJSBuilder) getImageTag(imageID string) string {
	// Strip any existing prefix if present
	if len(imageID) > 5 && imageID[:5] == "chef-" {
		return imageID
	}
	return fmt.Sprintf("chef-%s", imageID)
}

func (b *NodeJSBuilder) Cleanup() error {
	b.logger.Info("cleaning up nodejs builder resources")
	return os.RemoveAll(b.options.WorkDir)
}
