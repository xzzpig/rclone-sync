package api

import (
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/services"

	"github.com/xzzpig/rclone-sync/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterAPIRoutes(router *gin.RouterGroup) {
	// Initialize Services and Handlers
	client := db.Client
	taskService := services.NewTaskService(client)
	taskHandler := handlers.NewTaskHandler(taskService)

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

	// Remote management
	remotes := router.Group("/remotes")
	{
		remotes.GET("", handlers.ListRemotes)
		remotes.POST("/test", handlers.TestRemote)
		remotes.POST("/:name", handlers.CreateRemote)
		remotes.GET("/:name", handlers.GetRemoteInfo)
		remotes.GET("/:name/quota", handlers.GetRemoteQuota)
		remotes.DELETE("/:name", handlers.DeleteRemote)
		remotes.GET("/:name/events", handlers.GetConnectionEvents)
	}

	// Provider management
	providers := router.Group("/providers")
	{
		providers.GET("", handlers.ListProviders)
		providers.GET("/:name", handlers.GetProviderOptions)
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
		files.GET("/remote/:name", handlers.ListRemoteFiles)
	}
}
