package context

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

func getContextValue[T any](c *gin.Context, key string) (T, error) {
	var zero T
	val, exists := c.Get(key)
	if !exists {
		return zero, errors.New(key + " not initialized")
	}
	return val.(T), nil
}

// GetSyncEngine retrieves the SyncEngine from the gin context
// Returns an error if the SyncEngine is not found
func GetSyncEngine(c *gin.Context) (*rclone.SyncEngine, error) {
	return getContextValue[*rclone.SyncEngine](c, ContextKeySyncEngine)
}

// GetTaskRunner retrieves the TaskRunner from the gin context
// Returns an error if the TaskRunner is not found
func GetTaskRunner(c *gin.Context) (ports.Runner, error) {
	return getContextValue[ports.Runner](c, ContextKeyTaskRunner)
}

// GetJobService retrieves the JobService from the gin context
// Returns an error if the JobService is not found
func GetJobService(c *gin.Context) (ports.JobService, error) {
	return getContextValue[ports.JobService](c, ContextKeyJobService)
}

// GetWatcher retrieves the Watcher from the gin context
// Returns an error if the Watcher is not found
func GetWatcher(c *gin.Context) (ports.Watcher, error) {
	return getContextValue[ports.Watcher](c, ContextKeyWatcher)
}

// GetScheduler retrieves the Scheduler from the gin context
// Returns an error if the Scheduler is not found
func GetScheduler(c *gin.Context) (ports.Scheduler, error) {
	return getContextValue[ports.Scheduler](c, ContextKeyScheduler)
}
