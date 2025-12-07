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

	// Job management
	jobs := router.Group("/jobs")
	{
		jobs.GET("", handlers.ListJobs)
		jobs.GET("/:id", handlers.GetJob)
		jobs.GET("/:id/progress", handlers.GetJobProgress)
	}

	// Remote management
	remotes := router.Group("/remotes")
	{
		remotes.GET("", handlers.ListRemotes)
		remotes.POST("/:name", handlers.CreateRemote)
		remotes.GET("/:name", handlers.GetRemoteInfo)
		remotes.DELETE("/:name", handlers.DeleteRemote)
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
}
