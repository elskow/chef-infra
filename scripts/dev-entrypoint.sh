#!/bin/sh
set -e

echo "Starting development environment..."

echo "Running migrations..."
cd /app && GO111MODULE=on go run -tags postgres cmd/migrate/main.go -command up

echo "Starting application with Air..."
cd /app && exec air