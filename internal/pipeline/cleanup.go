package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/elskow/chef-infra/internal/pipeline/config"
	"go.uber.org/zap"
)

type CleanupManager struct {
	config *config.PipelineConfig
	logger *zap.Logger
}

func (cm *CleanupManager) CleanupOldBuilds(maxAge time.Duration) error {
	now := time.Now()
	buildDirs, err := os.ReadDir(cm.config.BuildDir)
	if err != nil {
		return fmt.Errorf("failed to read build directory: %w", err)
	}

	for _, dir := range buildDirs {
		if !dir.IsDir() {
			continue
		}

		info, err := dir.Info()
		if err != nil {
			cm.logger.Warn("failed to get directory info",
				zap.String("dir", dir.Name()),
				zap.Error(err))
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			path := filepath.Join(cm.config.BuildDir, dir.Name())
			if err := os.RemoveAll(path); err != nil {
				cm.logger.Error("failed to remove old build",
					zap.String("path", path),
					zap.Error(err))
			}
		}
	}

	return nil
}
