#!/bin/bash
# gen-migration.sh - Generate migration files from ent schema using Atlas CLI
#
# Usage: ./scripts/gen-migration.sh <migration_name>
# Example: ./scripts/gen-migration.sh add_user_table
#
# This script generates golang-migrate format migration files by comparing
# the current ent schema with the existing migrations.

set -e

# Check if migration name is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <migration_name>"
    echo "Example: $0 add_user_table"
    exit 1
fi

MIGRATION_NAME="$1"
MIGRATIONS_DIR="internal/core/db/migrations"
SCHEMA_DIR="internal/core/db/schema"

# Ensure we're in the project root
if [ ! -d "$SCHEMA_DIR" ]; then
    echo "Error: Schema directory not found. Please run this script from the project root."
    exit 1
fi

# Check if atlas is available
if ! command -v atlas &> /dev/null; then
    echo "Error: Atlas CLI not found. Please enter nix develop shell first."
    exit 1
fi

echo "Generating migration: $MIGRATION_NAME"
echo "Schema source: ent://$SCHEMA_DIR"
echo "Migrations dir: file://$MIGRATIONS_DIR"

# Generate migration using Atlas
atlas migrate diff "$MIGRATION_NAME" \
    --dir "file://$MIGRATIONS_DIR" \
    --dir-format golang-migrate \
    --to "ent://$SCHEMA_DIR" \
    --dev-url "sqlite://file?mode=memory"

echo ""
echo "Migration generated successfully!"
echo "Check the new files in $MIGRATIONS_DIR"
ls -la "$MIGRATIONS_DIR"/*.sql 2>/dev/null | tail -4 || echo "No SQL files found"
