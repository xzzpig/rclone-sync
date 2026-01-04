// Package api provides HTTP API routes and server setup.
package api

import (
	"fmt"

	"github.com/xzzpig/rclone-sync/internal/api/graphql"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/dataloader"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/resolver"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db/query"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"github.com/xzzpig/rclone-sync/internal/rclone"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RouterDeps contains all dependencies required for setting up API routes.
type RouterDeps struct {
	Client              *ent.Client
	Config              *config.Config
	SyncEngine          *rclone.SyncEngine
	Runner              ports.Runner
	JobQuery            *query.JobQuery
	Watcher             ports.Watcher
	Scheduler           ports.Scheduler
	JobProgressBus      *subscription.JobProgressBus
	TransferProgressBus *subscription.TransferProgressBus
}

// routesLog returns a named logger for the api.routes package.
func routesLog() *zap.Logger {
	return logger.Named("api.routes")
}

// RegisterAPIRoutes registers all API routes to the given router group.
func RegisterAPIRoutes(router *gin.RouterGroup, deps RouterDeps) error {
	// Initialize encryptor
	encryptor, err := crypto.NewEncryptor(deps.Config.Security.EncryptionKey)
	if err != nil {
		routesLog().Error("Failed to initialize encryptor", zap.Error(err))
		return fmt.Errorf("failed to initialize encryptor: %w", err)
	}

	// Initialize queries
	taskQuery := query.NewTaskQuery(deps.Client)
	connQuery := query.NewConnectionQuery(deps.Client, encryptor)

	// GraphQL endpoint
	gqlDeps := &resolver.Dependencies{
		SyncEngine:          deps.SyncEngine,
		Runner:              deps.Runner,
		JobQuery:            deps.JobQuery,
		Watcher:             deps.Watcher,
		Scheduler:           deps.Scheduler,
		TaskQuery:           taskQuery,
		ConnectionQuery:     connQuery,
		Encryptor:           encryptor,
		JobProgressBus:      deps.JobProgressBus,
		TransferProgressBus: deps.TransferProgressBus,
	}
	gqlHandler := graphql.NewHandler(gqlDeps)

	gqlGroup := router.Group("/graphql")
	gqlGroup.Use(dataloader.Middleware(deps.Client))
	{
		gqlGroup.POST("", graphql.GinHandler(gqlHandler))
		gqlGroup.GET("", graphql.GinHandler(gqlHandler)) // For WebSocket upgrade

		// GraphiQL Playground (development only)
		if deps.Config.App.Environment == "development" {
			gqlGroup.GET("/playground", graphql.PlaygroundHandler("/api/graphql"))
		}
	}

	return nil
}
