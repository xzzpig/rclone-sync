// Package db provides database initialization and connection management.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3" // SQLite driver for database/sql
	"go.uber.org/zap"
)

// Client is the global database client instance.
var Client *ent.Client

// InitDBOptions contains options for database initialization.
type InitDBOptions struct {
	Path          string        // Database file path
	MigrationMode MigrationMode // Migration mode (versioned or auto)
	EnableDebug   bool          // Enable SQL debug logging
}

// InitDB initializes the database connection and runs migrations.
func InitDB(opts InitDBOptions) {
	var err error

	// Open database connection
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1", opts.Path))
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}

	// Create ent driver
	drv := entsql.OpenDB("sqlite3", db)

	// Create ent client with optional debug logging
	options := []ent.Option{ent.Driver(drv)}
	if opts.EnableDebug {
		options = append(options, ent.Debug())
	}
	Client = ent.NewClient(options...)

	// Execute migrations based on mode
	switch opts.MigrationMode {
	case MigrationModeAuto:
		logger.L.Info("Using auto migration mode (ent schema)")
		if err := Client.Schema.Create(context.Background()); err != nil {
			log.Fatalf("failed to run auto migration: %v", err)
		}
		logger.L.Info("Auto migration completed successfully")
	default:
		logger.L.Info("Using versioned migration mode", zap.String("mode", string(opts.MigrationMode)))
		if err := Migrate(db); err != nil {
			log.Fatalf("failed to run versioned migrations: %v", err)
		}
		// Log migration status
		LogMigrationStatus(db)
	}

	// Log initialization status
	if opts.EnableDebug {
		logger.L.Info("Database initialized with SQL query logging enabled")
	}
}

// CloseDB closes the database connection.
func CloseDB() {
	if Client != nil {
		_ = Client.Close()
	}
}
