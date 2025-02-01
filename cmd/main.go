package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/elskow/chef-infra/internal/server"
)

func main() {
	// Set environment through environment variable
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = server.EnvDevelopment // Default to development
	}

	log.Printf("Starting server in %s mode", env)

	config, err := server.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	srv := server.NewServer(config)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down server...")
	srv.Stop()
}
