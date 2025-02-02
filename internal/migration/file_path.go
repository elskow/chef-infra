package migration

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// getMigrationsDir returns the absolute path to the migrations directory
func getMigrationsDir() (string, error) {
	// First, try to find go.mod
	dir, err := findModuleRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}

	return filepath.Join(dir, "migrations"), nil
}

// findModuleRoot returns the root directory of the module
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gomod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(gomod); err == nil {
			// Verify this is our module
			content, err := os.ReadFile(gomod)
			if err != nil {
				return "", err
			}

			modPath := modfile.ModulePath(content)
			if modPath == "github.com/elskow/chef-infra" {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
