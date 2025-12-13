package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
)

var Client *ent.Client

func InitDB() {
	var err error

	// Open database connection
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1", config.Cfg.Database.Path))
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}

	// Create ent driver
	drv := entsql.OpenDB("sqlite3", db)

	// Create ent client with optional debug logging
	options := []ent.Option{ent.Driver(drv)}
	if config.Cfg.App.Environment == "development" {
		options = append(options, ent.Debug())
	}
	Client = ent.NewClient(options...)

	// Run the auto migration tool.
	// TODO: versioned migrations
	if err := Client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	// Log initialization status
	if config.Cfg.App.Environment == "development" {
		logger.L.Info("Database initialized with SQL query logging enabled")
	}
}

func CloseDB() {
	if Client != nil {
		Client.Close()
	}
}
