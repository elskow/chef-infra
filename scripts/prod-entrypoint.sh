#!/bin/bash
set -e

# Run migrations
echo "Running database migrations..."
/app/bin/migrate -command up

# Start the application
echo "Starting application in production mode..."
exec /app/bin/chef-infra