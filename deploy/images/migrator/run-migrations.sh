#!/bin/sh
set -e

if [ -z "$MIGRATE_PATH" ] || [ -z "$MIGRATE_DB_URL" ]; then
  echo "Error: MIGRATE_PATH and MIGRATE_DB_URL environment variables are required."
  exit 1
fi

echo "Running migrations..."
migrate -path "$MIGRATE_PATH" -database "$MIGRATE_DB_URL" up

echo "Migrations applied successfully."