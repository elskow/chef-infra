package builder

import (
	"fmt"
	"os"
	"path/filepath"
)

type BuildContext struct {
	RootDir     string
	BuildDir    string
	ArtifactDir string
	CacheDir    string
}

func NewBuildContext(rootDir, buildID string) (*BuildContext, error) {
	bc := &BuildContext{
		RootDir:     rootDir,
		BuildDir:    filepath.Join(rootDir, "builds", buildID),
		ArtifactDir: filepath.Join(rootDir, "artifacts", buildID),
		CacheDir:    filepath.Join(rootDir, "cache", buildID),
	}

	// Create directories
	dirs := []string{bc.BuildDir, bc.ArtifactDir, bc.CacheDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return bc, nil
}

func (bc *BuildContext) Cleanup() error {
	// Cleanup everything except artifacts
	return os.RemoveAll(bc.BuildDir)
}
