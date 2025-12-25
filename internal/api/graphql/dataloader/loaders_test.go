// Package dataloader provides dataloaders for efficient batch loading of data.
package dataloader_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/dataloader"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
)

// setupTestDB creates an in-memory database for testing.
func setupTestDB(t *testing.T) *ent.Client {
	t.Helper()

	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug, nil)

	client, err := db.InitDB(db.InitDBOptions{
		DSN:           db.InMemoryDSN(),
		MigrationMode: db.MigrationModeAuto,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

func TestNewLoaders(t *testing.T) {
	client := setupTestDB(t)

	loaders := dataloader.NewLoaders(client)

	assert.NotNil(t, loaders)
	assert.NotNil(t, loaders.ConnectionLoader)
	assert.NotNil(t, loaders.TaskLoader)
	assert.NotNil(t, loaders.JobLoader)
}

func TestMiddleware(t *testing.T) {
	client := setupTestDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add dataloader middleware
	router.Use(dataloader.Middleware(client))

	var capturedLoaders *dataloader.Loaders

	// Add a test handler that captures the loaders from context
	router.GET("/test", func(c *gin.Context) {
		capturedLoaders = dataloader.For(c.Request.Context())
		c.Status(http.StatusOK)
	})

	// Make a request
	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedLoaders)
	assert.NotNil(t, capturedLoaders.ConnectionLoader)
	assert.NotNil(t, capturedLoaders.TaskLoader)
	assert.NotNil(t, capturedLoaders.JobLoader)
}

func TestFor_PanicWhenNotInContext(t *testing.T) {
	// Create a context without loaders
	ctx := context.Background()

	// Should panic when loaders are not in context
	assert.Panics(t, func() {
		dataloader.For(ctx)
	})
}

func TestFor_ReturnsLoadersFromContext(t *testing.T) {
	client := setupTestDB(t)

	// Create loaders and add to context
	loaders := dataloader.NewLoaders(client)

	// Simulate adding loaders to context via middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(dataloader.Middleware(client))

	var retrievedLoaders *dataloader.Loaders
	router.GET("/test", func(c *gin.Context) {
		retrievedLoaders = dataloader.For(c.Request.Context())
		c.Status(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotNil(t, retrievedLoaders)
	// Each request should get its own loaders instance
	assert.NotSame(t, loaders, retrievedLoaders)
}

func TestLoaders_AreIsolatedPerRequest(t *testing.T) {
	client := setupTestDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(dataloader.Middleware(client))

	var loaders1, loaders2 *dataloader.Loaders

	router.GET("/test1", func(c *gin.Context) {
		loaders1 = dataloader.For(c.Request.Context())
		c.Status(http.StatusOK)
	})

	router.GET("/test2", func(c *gin.Context) {
		loaders2 = dataloader.For(c.Request.Context())
		c.Status(http.StatusOK)
	})

	// Make first request
	req1, _ := http.NewRequest(http.MethodGet, "/test1", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Make second request
	req2, _ := http.NewRequest(http.MethodGet, "/test2", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Each request should have its own loaders instance
	assert.NotNil(t, loaders1)
	assert.NotNil(t, loaders2)
	assert.NotSame(t, loaders1, loaders2)
}
