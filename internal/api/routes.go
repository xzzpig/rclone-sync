// Package api provides HTTP API routes and server setup.
package api

import (
	"fmt"

	"github.com/xzzpig/rclone-sync/internal/api/handlers"
	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RouterDeps contains all dependencies required for setting up API routes.
type RouterDeps struct {
	Client *ent.Client
	Config *config.Config
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

	// Initialize services
	taskService := services.NewTaskService(deps.Client)
	connService := services.NewConnectionService(deps.Client, encryptor)

	// Initialize handlers
	taskHandler := handlers.NewTaskHandler(taskService)
	connHandler := handlers.NewConnectionHandler(connService)
	importHandler := handlers.NewImportHandler(connService)
	filesHandler := handlers.NewFilesHandler(connService)

	// Global SSE events endpoint
	router.GET("/events", handlers.GetGlobalEvents)

	// Job management
	jobs := router.Group("/jobs")
	{
		jobs.GET("", handlers.ListJobs)
		jobs.GET("/:id", handlers.GetJob)
		jobs.GET("/:id/progress", handlers.GetJobProgress)
	}

	// Log management
	logs := router.Group("/logs")
	{
		logs.GET("", handlers.ListLogs)
	}

	// Provider management
	providers := router.Group("/providers")
	{
		providers.GET("", handlers.ListProviders)
		providers.GET("/:name", handlers.GetProviderOptions)
	}

	// Connection management
	connections := router.Group("/connections")
	{
		connections.POST("", connHandler.Create)
		connections.GET("", connHandler.List)
		connections.POST("/test", connHandler.TestUnsavedConfig)
		connections.GET("/:id", connHandler.Get)
		connections.GET("/:id/config", connHandler.GetConfig)
		connections.POST("/:id/test", connHandler.Test)
		connections.GET("/:id/quota", connHandler.GetQuota)
		connections.PUT("/:id", connHandler.Update)
		connections.DELETE("/:id", connHandler.Delete)
	}

	// Task management
	tasks := router.Group("/tasks")
	{
		tasks.POST("", taskHandler.Create)
		tasks.GET("", taskHandler.List)
		tasks.POST("/:id/run", taskHandler.Run)
		tasks.GET("/:id", taskHandler.Get)
		tasks.PUT("/:id", taskHandler.Update)
		tasks.DELETE("/:id", taskHandler.Delete)
	}

	// File browsing
	files := router.Group("/files")
	{
		files.GET("/local", handlers.ListLocalFiles)
		files.GET("/remote/:id", filesHandler.ListRemoteFiles)
	}

	// Import management
	imports := router.Group("/import")
	{
		imports.POST("/parse", importHandler.Parse)
		imports.POST("/execute", importHandler.Execute)
	}

	return nil
}
