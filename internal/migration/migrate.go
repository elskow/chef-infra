package migration

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"

	"github.com/elskow/chef-infra/internal/config"
)

type Migrator struct {
	db     *sql.DB
	config *config.DatabaseConfig
}

func NewMigrator(config *config.DatabaseConfig) (*Migrator, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		config.Host,
		config.User,
		config.Password,
		config.Name,
		config.Port,
		config.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Migrator{
		db:     db,
		config: config,
	}, nil
}

func (m *Migrator) Up() error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return fmt.Errorf("failed to get migrations directory: %w", err)
	}

	if err := goose.Up(m.db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (m *Migrator) Down() error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return fmt.Errorf("failed to get migrations directory: %w", err)
	}

	if err := goose.Down(m.db, migrationsDir); err != nil {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	return nil
}

func (m *Migrator) Close() error {
	return m.db.Close()
}

// GetCurrentVersion returns the current migration version
func (m *Migrator) GetCurrentVersion() (int64, error) {
	return goose.GetDBVersion(m.db)
}

// GetLatestVersion returns the latest available migration version
func (m *Migrator) GetLatestVersion() (int64, error) {
	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return 0, err
	}

	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	if err != nil {
		return 0, err
	}

	if len(migrations) == 0 {
		return 0, nil
	}

	return migrations[len(migrations)-1].Version, nil
}

// DownTo migrates the database down to a specific version
func (m *Migrator) DownTo(version int64) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return fmt.Errorf("failed to get migrations directory: %w", err)
	}

	current, err := m.GetCurrentVersion()
	if err != nil {
		return err
	}

	// Perform one migration down at a time until we reach the target version
	for current > version {
		if err := goose.Down(m.db, migrationsDir); err != nil {
			return fmt.Errorf("failed to migrate down to version %d: %w", version, err)
		}
		current, err = m.GetCurrentVersion()
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) Status() error {
	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return fmt.Errorf("failed to get migrations directory: %w", err)
	}

	if err := goose.Status(m.db, migrationsDir); err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}
	return nil
}

func (m *Migrator) Version() (int64, error) {
	return goose.GetDBVersion(m.db)
}

func (m *Migrator) Reset() error {
	if err := m.Down(); err != nil {
		return err
	}
	return m.Up()
}
