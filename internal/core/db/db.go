package db

import (
	"context"
	"fmt"
	"log"

	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/ent"

	_ "github.com/mattn/go-sqlite3"
)

var Client *ent.Client

func InitDB() {
	var err error
	Client, err = ent.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1", config.Cfg.Database.Path))
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}

	// Run the auto migration tool.
	// TODO: versioned migrations
	if err := Client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}
}

func CloseDB() {
	if Client != nil {
		Client.Close()
	}
}
