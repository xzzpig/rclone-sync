// Package db provides database initialization and connection management.
package db

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

// MigrationMode represents the database migration mode.
type MigrationMode string

const (
	// MigrationModeVersioned uses versioned migration files (production default).
	MigrationModeVersioned MigrationMode = "versioned"
	// MigrationModeAuto uses ent's automatic schema migration (development/testing).
	MigrationModeAuto MigrationMode = "auto"
)

// ParseMigrationMode parses a string to MigrationMode.
// Returns MigrationModeVersioned for unknown values.
func ParseMigrationMode(s string) MigrationMode {
	switch s {
	case "auto":
		return MigrationModeAuto
	default:
		return MigrationModeVersioned
	}
}

// migrateLogger implements migrate.Logger interface for golang-migrate.
type migrateLogger struct {
	environment string
}

// Printf logs migration messages.
func (l *migrateLogger) Printf(format string, v ...interface{}) {
	log().Info(fmt.Sprintf(format, v...))
}

// Verbose returns true if verbose logging is enabled.
func (l *migrateLogger) Verbose() bool {
	return l.environment == "development"
}

// Migrate executes database migrations from embedded SQL files.
// The environment parameter is used for logging verbosity control.
func Migrate(db *sql.DB, environment string) error {
	// 1. Create migration source from embed.FS (specify migrations subdirectory)
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// 2. Create SQLite database driver
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// 3. Create migrate instance
	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// 4. Set logger
	m.Log = &migrateLogger{environment: environment}

	// 5. Execute migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log().Info("No pending migrations")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	log().Info("Migrations completed successfully")
	return nil
}

// MigrationStatus represents the current migration status.
type MigrationStatus struct {
	Version uint  // Current migration version
	Dirty   bool  // Whether the database is in a dirty state
	Error   error // Any error encountered
}

// GetMigrationStatus returns the current migration status.
func GetMigrationStatus(db *sql.DB) (*MigrationStatus, error) {
	// Create migration source from embed.FS
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create SQLite database driver
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			// No migrations have been applied yet
			return &MigrationStatus{Version: 0, Dirty: false}, nil
		}
		return nil, fmt.Errorf("failed to get migration version: %w", err)
	}

	return &MigrationStatus{Version: version, Dirty: dirty}, nil
}

// GetPendingMigrations returns the list of pending migration versions.
func GetPendingMigrations(db *sql.DB) ([]uint, error) {
	// Create migration source from embed.FS
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create SQLite database driver
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Get current version
	currentVersion, _, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return nil, fmt.Errorf("failed to get migration version: %w", err)
	}

	// Collect all available versions from source
	var pending []uint
	version, err := source.First()
	if err != nil {
		// No migrations available
		return pending, nil
	}

	for {
		if version > currentVersion {
			pending = append(pending, version)
		}
		nextVersion, err := source.Next(version)
		if err != nil {
			break // No more versions
		}
		version = nextVersion
	}

	return pending, nil
}

// LogMigrationStatus logs the current migration status.
func LogMigrationStatus(db *sql.DB) {
	status, err := GetMigrationStatus(db)
	if err != nil {
		log().Warn("Failed to get migration status", zap.Error(err))
		return
	}

	pending, err := GetPendingMigrations(db)
	if err != nil {
		log().Warn("Failed to get pending migrations", zap.Error(err))
		return
	}

	log().Info("Database migration status",
		zap.Uint("current_version", status.Version),
		zap.Bool("dirty", status.Dirty),
		zap.Int("pending_count", len(pending)),
	)
}
