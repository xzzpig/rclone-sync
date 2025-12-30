// Package db provides database initialization and connection management.
package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3" // SQLite driver for database/sql
	"go.uber.org/zap"
)

// log returns a named logger for the db package.
func log() *zap.Logger {
	return logger.Named("core.db")
}

// InitDBOptions contains options for database initialization.
type InitDBOptions struct {
	DSN           string        // SQLite DSN connection string (e.g., "file:data.db?cache=shared&_fk=1")
	MigrationMode MigrationMode // Migration mode (versioned or auto)
	EnableDebug   bool          // Enable SQL debug logging
	Environment   string        // Application environment (for migrations)
}

// InitDB initializes the database connection and runs migrations.
// Returns the ent client and any error encountered.
func InitDB(opts InitDBOptions) (*ent.Client, error) {
	// Open database connection
	sqlDB, err := sql.Open("sqlite3", opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to sqlite: %w", err)
	}

	// Create ent driver
	drv := entsql.OpenDB("sqlite3", sqlDB)

	// Create ent client with optional debug logging
	options := []ent.Option{ent.Driver(drv)}
	if opts.EnableDebug {
		options = append(options, ent.Debug(), ent.Log(logger.Named("core.db.query").Sugar().Debug))
	}
	client := ent.NewClient(options...)

	// Execute migrations based on mode
	switch opts.MigrationMode {
	case MigrationModeAuto:
		log().Info("Using auto migration mode (ent schema)")
		if err := client.Schema.Create(context.Background()); err != nil {
			if closeErr := client.Close(); closeErr != nil {
				log().Warn("Failed to close client after migration error", zap.Error(closeErr))
			}
			return nil, fmt.Errorf("failed to run auto migration: %w", err)
		}
		log().Info("Auto migration completed successfully")
	default:
		log().Info("Using versioned migration mode", zap.String("mode", string(opts.MigrationMode)))
		if err := Migrate(sqlDB, opts.Environment); err != nil {
			if closeErr := client.Close(); closeErr != nil {
				log().Warn("Failed to close client after migration error", zap.Error(closeErr))
			}
			return nil, fmt.Errorf("failed to run versioned migrations: %w", err)
		}
		// Log migration status
		LogMigrationStatus(sqlDB)
	}

	// Log initialization status
	if opts.EnableDebug {
		log().Info("Database initialized with SQL query logging enabled")
	}

	return client, nil
}

// CloseDB closes the database connection.
func CloseDB(client *ent.Client) {
	if client != nil {
		_ = client.Close()
	}
}

// FileSDN constructs a SQLite DSN for a file-based database with common parameters.
// Note: cache=shared is intentionally not used because it introduces table-level locking
// which causes SQLITE_LOCKED errors that cannot be resolved with busy_timeout.
// Without shared cache, each connection has its own page cache, using more memory
// but providing better concurrency with WAL mode.
func FileSDN(path string) string {
	return fmt.Sprintf("file:%s?_fk=1&_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL", path)
}

// InMemoryDSN returns the DSN for an in-memory SQLite database.
func InMemoryDSN() string {
	return "file:ent?mode=memory&cache=shared&_fk=1&_busy_timeout=5000"
}
