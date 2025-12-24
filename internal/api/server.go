package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/logger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xzzpig/rclone-sync/internal/ui"

	"github.com/xzzpig/rclone-sync/internal/api/context"
)

// srvLog returns a named logger for the api.server package.
func srvLog() *zap.Logger {
	return logger.Named("api.server")
}

// SetupRouter creates and configures the Gin router with all middleware and routes.
func SetupRouter(deps RouterDeps) *gin.Engine {
	cfg := deps.Config
	if cfg.App.Environment == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Middleware
	r.Use(ginLogger(logger.Named("api.http")))
	r.Use(gin.Recovery())
	r.Use(context.Middleware(deps.SyncEngine, deps.Runner, deps.JobService, deps.Watcher, deps.Scheduler))
	r.Use(context.LocaleMiddleware())    // Parse Accept-Language header
	r.Use(context.I18nErrorMiddleware()) // Handle I18nError responses
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Adjust for production
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "Accept-Language"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API Group
	apiGroup := r.Group("/api")
	{
		// Register routes here
		if err := RegisterAPIRoutes(apiGroup, deps); err != nil {
			srvLog().Fatal("Failed to register API routes", zap.Error(err))
		}
		// sse.RegisterRoutes(api)
	}

	// Serve Frontend
	if err := setupFrontendService(r, cfg); err != nil {
		srvLog().Error("Failed to setup frontend service", zap.Error(err))
	}

	return r
}

// setupFrontendService configures the frontend file serving for the router
// and returns an error if setup fails
func setupFrontendService(r *gin.Engine, cfg *config.Config) error {
	// In development, we can also serve from the dist folder if it exists,
	// which is useful for testing production build locally.
	// But usually in dev mode we use the Vite dev server.
	var fs http.FileSystem
	var err error

	if cfg.App.Environment != "development" {
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
			_ = f.Close()
			http.FileServer(fs).ServeHTTP(c.Writer, c.Request)
			return
		}
		// Fallback to index.html for SPA
		// Try to open index.html
		f, err = fs.Open("index.html")
		if err != nil {
			srvLog().Error("Failed to serve index.html", zap.Error(err))
			notFoundHandler(c)
			return
		}
		_ = f.Close()
		// Serve index.html
		c.Request.URL.Path = "/"
		http.FileServer(fs).ServeHTTP(c.Writer, c.Request)
	})

	return nil
}

// notFoundHandler handles 404 responses.
func notFoundHandler(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

func ginLogger(l *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 排除 GraphQL 路由（由 GraphQL 中间件单独处理）
		if strings.HasPrefix(path, "/api/graphql") {
			c.Next()
			return
		}

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
