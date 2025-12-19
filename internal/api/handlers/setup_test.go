package handlers_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api"
	apiContext "github.com/xzzpig/rclone-sync/internal/api/context"

	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
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
	Server            *httptest.Server
	Client            *ent.Client
	TaskService       *services.TaskService
	JobService        *services.JobService
	ConnectionService *services.ConnectionService
	Runner            ports.Runner
	AppDataDir        string
	Cleanup           func()
}

// setupTestServer initializes a test server with an in-memory database and all services.
func setupTestServer(t *testing.T) *TestServer {
	// Init logger
	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)

	// 2. Initialize in-memory database using db.InitDB
	client, err := db.InitDB(db.InitDBOptions{
		DSN:           "file:ent?mode=memory&cache=shared&_fk=1",
		MigrationMode: db.MigrationModeAuto,
	})
	require.NoError(t, err)

	// 3. Create temporary directories
	appDataDir := t.TempDir()

	// 4. Initialize services
	jobService := services.NewJobService(client)
	taskService := services.NewTaskService(client)

	// Initialize encryption for ConnectionService
	encryptor, err := crypto.NewEncryptor("test-encryption-key-32-bytes!!")
	require.NoError(t, err)
	connectionService := services.NewConnectionService(client, encryptor)

	// Install DBStorage for rclone configuration
	storage := rclone.NewDBStorage(connectionService)
	storage.Install()

	syncEngine := rclone.NewSyncEngine(jobService, appDataDir)
	runnerInstance := runner.NewRunner(syncEngine)

	// Create mock watcher and scheduler for testing
	mockWatcher := &mockWatcher{}
	mockScheduler := &mockScheduler{}

	// 5. Setup Gin router and register routes
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add TaskRunner middleware
	router.Use(apiContext.Middleware(syncEngine, runnerInstance, jobService, mockWatcher, mockScheduler))

	// Create RouterDeps
	routerDeps := api.RouterDeps{
		Client: client,
		Config: &config.Config{},
	}

	apiGroup := router.Group("/api")
	err = api.RegisterAPIRoutes(apiGroup, routerDeps)
	require.NoError(t, err)

	// 7. Create httptest server
	server := httptest.NewServer(router)

	// 8. Define cleanup function
	cleanup := func() {
		server.Close()
		client.Close()
	}

	return &TestServer{
		Server:            server,
		Client:            client,
		TaskService:       taskService,
		JobService:        jobService,
		ConnectionService: connectionService,
		Runner:            runnerInstance,
		AppDataDir:        appDataDir,
		Cleanup:           cleanup,
	}
}

// createTestConnection creates a test connection and returns its ID
func createTestConnection(t *testing.T, ts *TestServer, name string) uuid.UUID {
	conn, err := ts.ConnectionService.CreateConnection(context.Background(), name, "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)
	return conn.ID
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
