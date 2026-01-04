package context

import (
	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// Middleware returns a gin middleware that sets all required context values
func Middleware(syncEngine *rclone.SyncEngine, taskRunner ports.Runner, jobQuery ports.JobQuery, watcher ports.Watcher, scheduler ports.Scheduler) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ContextKeySyncEngine, syncEngine)
		c.Set(ContextKeyTaskRunner, taskRunner)
		c.Set(ContextKeyJobQuery, jobQuery)
		c.Set(ContextKeyWatcher, watcher)
		c.Set(ContextKeyScheduler, scheduler)
		c.Next()
	}
}
