package main

import (
	"flag"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/elskow/chef-infra/internal/migration"
	"github.com/elskow/chef-infra/internal/server"
)

func main() {
	command := flag.String("command", "up", "migration command (up/down/status/version/reset)")
	flag.Parse()

	if os.Getenv("APP_ENV") == "" {
		os.Setenv("APP_ENV", "development")
	}

	// Load config
	cfg, err := server.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create migrator
	migrator, err := migration.NewMigrator(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	// Run migration command
	switch *command {
	case "up":
		if err := migrator.Up(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Successfully ran migrations")

	case "down":
		if err := migrator.Down(); err != nil {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		log.Println("Successfully rolled back migrations")

	case "status":
		if err := migrator.Status(); err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}

	case "version":
		version, err := migrator.Version()
		if err != nil {
			log.Fatalf("Failed to get migration version: %v", err)
		}
		log.Printf("Current migration version: %d", version)

	case "reset":
		if err := migrator.Reset(); err != nil {
			log.Fatalf("Failed to reset migrations: %v", err)
		}
		log.Println("Successfully reset migrations")

	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}
