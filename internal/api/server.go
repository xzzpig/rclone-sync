package api

import (
	"net/http"
	"time"

	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xzzpig/rclone-sync/internal/core/runner"
	"github.com/xzzpig/rclone-sync/internal/rclone"
	"github.com/xzzpig/rclone-sync/internal/ui"

	"github.com/xzzpig/rclone-sync/internal/api/context"
	"github.com/xzzpig/rclone-sync/internal/api/handlers"
)

func SetupRouter(syncEngine *rclone.SyncEngine, taskRunner *runner.Runner, jobService ports.JobService, watcher ports.Watcher, scheduler ports.Scheduler) *gin.Engine {
	if config.Cfg.App.Environment == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Middleware
	r.Use(ginLogger(logger.L))
	r.Use(gin.Recovery())
	r.Use(context.Middleware(syncEngine, taskRunner, jobService, watcher, scheduler))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Adjust for production
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API Group
	api := r.Group("/api")
	{
		// Register routes here
		RegisterAPIRoutes(api)
		// sse.RegisterRoutes(api)
	}

	// Serve Frontend
	if err := setupFrontendService(r); err != nil {
		logger.L.Error("Failed to setup frontend service", zap.Error(err))
	}

	return r
}

// setupFrontendService configures the frontend file serving for the router
// and returns an error if setup fails
func setupFrontendService(r *gin.Engine) error {
	// In development, we can also serve from the dist folder if it exists,
	// which is useful for testing production build locally.
	// But usually in dev mode we use the Vite dev server.
	var fs http.FileSystem
	var err error

	if config.Cfg.App.Environment != "development" {
		fs, err = ui.GetFileSystem()
	} else {
		// In development, try to serve from local dist folder directly
		// This requires running `pnpm build` in web folder first
		fs = http.Dir("internal/ui/dist")
	}

	if err != nil {
		return err
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Check if file exists in FS
		f, err := fs.Open(path)
		if err == nil {
			f.Close()
			http.FileServer(fs).ServeHTTP(c.Writer, c.Request)
			return
		}
		// Fallback to index.html for SPA
		// Try to open index.html
		f, err = fs.Open("index.html")
		if err == nil {
			f.Close()
			// Serve index.html
			c.Request.URL.Path = "/"
			http.FileServer(fs).ServeHTTP(c.Writer, c.Request)
			return
		} else {
			logger.L.Error("Failed to serve index.html", zap.Error(err))
			handlers.NotFoundHandler(c)
		}
	})

	return nil
}

func ginLogger(l *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				l.Error(e)
			}
		} else {
			l.Info(path,
				zap.Int("status", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("ip", c.ClientIP()),
				zap.String("user-agent", c.Request.UserAgent()),
				zap.Duration("latency", latency),
			)
		}
	}
}
