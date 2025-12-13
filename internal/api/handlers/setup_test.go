package handlers_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api"
	apiContext "github.com/xzzpig/rclone-sync/internal/api/context"

	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"github.com/xzzpig/rclone-sync/internal/core/runner"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// TestServer holds the components for API integration testing.
type TestServer struct {
	Server      *httptest.Server
	Client      *ent.Client
	TaskService *services.TaskService
	JobService  *services.JobService
	Runner      ports.Runner
	AppDataDir  string
	Cleanup     func()
}

// setupTestServer initializes a test server with an in-memory database and all services.
func setupTestServer(t *testing.T) *TestServer {
	// Init logger
	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)

	// 2. Initialize in-memory database
	client, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)

	// Run migrations
	err = client.Schema.Create(context.Background())
	require.NoError(t, err)

	// Set the global client for the api package to use
	db.Client = client

	// 3. Create temporary directories and files
	appDataDir := t.TempDir()

	rcloneConfPath := filepath.Join(appDataDir, "rclone.conf")
	confContent := `[local]
type = local
`
	err = os.WriteFile(rcloneConfPath, []byte(confContent), 0644)
	require.NoError(t, err)
	rclone.InitConfig(rcloneConfPath)

	// 4. Initialize services
	jobService := services.NewJobService(client)
	taskService := services.NewTaskService(client)
	syncEngine := rclone.NewSyncEngine(jobService, appDataDir)
	runner := runner.NewRunner(syncEngine)

	// Create mock watcher and scheduler for testing
	mockWatcher := &mockWatcher{}
	mockScheduler := &mockScheduler{}

	// 5. Setup Gin router and register routes
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add TaskRunner middleware
	router.Use(apiContext.Middleware(syncEngine, runner, jobService, mockWatcher, mockScheduler))

	apiGroup := router.Group("/api")
	api.RegisterAPIRoutes(apiGroup)

	// 7. Create httptest server
	server := httptest.NewServer(router)

	// 8. Define cleanup function
	cleanup := func() {
		server.Close()
		client.Close()
	}

	return &TestServer{
		Server:      server,
		Client:      client,
		TaskService: taskService,
		JobService:  jobService,
		Runner:      runner,
		AppDataDir:  appDataDir,
		Cleanup:     cleanup,
	}
}

// mockWatcher is a mock implementation of ports.Watcher for testing
type mockWatcher struct{}

func (m *mockWatcher) Start()                          {}
func (m *mockWatcher) Stop()                           {}
func (m *mockWatcher) AddTask(task *ent.Task) error    { return nil }
func (m *mockWatcher) RemoveTask(task *ent.Task) error { return nil }

// mockScheduler is a mock implementation of ports.Scheduler for testing
type mockScheduler struct{}

func (m *mockScheduler) Start()                          {}
func (m *mockScheduler) Stop()                           {}
func (m *mockScheduler) AddTask(task *ent.Task) error    { return nil }
func (m *mockScheduler) RemoveTask(task *ent.Task) error { return nil }
