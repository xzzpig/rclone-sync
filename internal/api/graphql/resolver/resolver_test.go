// Package resolver provides GraphQL resolver implementations.
package resolver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xzzpig/rclone-sync/internal/api/graphql"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/dataloader"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/generated"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/resolver"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"github.com/xzzpig/rclone-sync/internal/core/runner"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/i18n"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// TestEnv holds the components for GraphQL resolver testing.
type TestEnv struct {
	Client            *ent.Client
	Server            *handler.Server
	Router            *gin.Engine
	Deps              *resolver.Dependencies
	TaskService       *services.TaskService
	JobService        *services.JobService
	ConnectionService *services.ConnectionService
	Runner            ports.Runner
	Cleanup           func()
}

// NewTestEnv initializes a test environment with a file-based database and all services.
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	// Init logger
	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug, nil)

	// Init i18n (required for i18n error handling)
	require.NoError(t, i18n.Init())

	// Create temporary directories
	appDataDir := t.TempDir()

	// Initialize file-based database (avoids SQLite memory database locking issues)
	dbPath := appDataDir + "/test.db"
	client, err := db.InitDB(db.InitDBOptions{
		DSN:           "file:" + dbPath + "?_fk=1&_journal_mode=WAL",
		MigrationMode: db.MigrationModeAuto,
	})
	require.NoError(t, err)

	// Initialize services
	jobService := services.NewJobService(client)
	taskService := services.NewTaskService(client)

	// Initialize encryption for ConnectionService
	encryptor, err := crypto.NewEncryptor("test-encryption-key-32-bytes!!")
	require.NoError(t, err)
	connectionService := services.NewConnectionService(client, encryptor)

	// Install DBStorage for rclone configuration
	storage := rclone.NewDBStorage(connectionService)
	storage.Install()

	syncEngine := rclone.NewSyncEngine(jobService, nil, nil, appDataDir, false, 0)
	runnerInstance := runner.NewRunner(syncEngine)

	// Create mock watcher and scheduler for testing
	mockWatcher := &mockWatcher{}
	mockScheduler := &mockScheduler{}

	// Create job progress bus and transfer progress bus
	jobProgressBus := subscription.NewJobProgressBus()
	transferProgressBus := subscription.NewTransferProgressBus()

	// Create dependencies
	deps := &resolver.Dependencies{
		SyncEngine:          syncEngine,
		Runner:              runnerInstance,
		JobService:          jobService,
		Watcher:             mockWatcher,
		Scheduler:           mockScheduler,
		TaskService:         taskService,
		ConnectionService:   connectionService,
		Encryptor:           encryptor,
		JobProgressBus:      jobProgressBus,
		TransferProgressBus: transferProgressBus,
	}

	// Create GraphQL handler
	srv := graphql.NewHandler(deps)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add dataloader middleware
	router.Use(dataloader.Middleware(client))

	router.POST("/graphql", graphql.GinHandler(srv))

	// Define cleanup function
	cleanup := func() {
		client.Close()
	}

	return &TestEnv{
		Client:            client,
		Server:            srv,
		Router:            router,
		Deps:              deps,
		TaskService:       taskService,
		JobService:        jobService,
		ConnectionService: connectionService,
		Runner:            runnerInstance,
		Cleanup:           cleanup,
	}
}

// GraphQLRequest represents a GraphQL request body.
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response body.
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data,omitempty"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message    string                 `json:"message"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// ExecuteGraphQL executes a GraphQL query and returns the response.
func (e *TestEnv) ExecuteGraphQL(t *testing.T, req GraphQLRequest) *GraphQLResponse {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(http.MethodPost, "/graphql", bytes.NewBuffer(body))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	e.Router.ServeHTTP(w, httpReq)

	var resp GraphQLResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return &resp
}

// ExecuteGraphQLWithVars executes a GraphQL query with variables.
func (e *TestEnv) ExecuteGraphQLWithVars(t *testing.T, query string, vars map[string]interface{}) *GraphQLResponse {
	return e.ExecuteGraphQL(t, GraphQLRequest{
		Query:     query,
		Variables: vars,
	})
}

// CreateTestConnection creates a test connection and returns its ID.
func (e *TestEnv) CreateTestConnection(t *testing.T, name string) uuid.UUID {
	t.Helper()
	conn, err := e.ConnectionService.CreateConnection(context.Background(), name, "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)
	return conn.ID
}

// CreateTestTask creates a test task and returns it.
func (e *TestEnv) CreateTestTask(t *testing.T, name string, connectionID uuid.UUID) *ent.Task {
	t.Helper()
	task, err := e.TaskService.CreateTask(
		context.Background(),
		name,
		"/tmp/source",
		connectionID,
		"/remote",
		"UPLOAD",
		"",
		false,
		nil,
	)
	require.NoError(t, err)
	return task
}

// mockWatcher is a mock implementation of ports.Watcher for testing.
type mockWatcher struct{}

func (m *mockWatcher) Start()                          {}
func (m *mockWatcher) Stop()                           {}
func (m *mockWatcher) AddTask(task *ent.Task) error    { return nil }
func (m *mockWatcher) RemoveTask(task *ent.Task) error { return nil }

// mockScheduler is a mock implementation of ports.Scheduler for testing.
type mockScheduler struct{}

func (m *mockScheduler) Start()                          {}
func (m *mockScheduler) Stop()                           {}
func (m *mockScheduler) AddTask(task *ent.Task) error    { return nil }
func (m *mockScheduler) RemoveTask(task *ent.Task) error { return nil }

// ResolverTestSuite is a base test suite for resolver tests.
type ResolverTestSuite struct {
	suite.Suite
	Env *TestEnv
}

// SetupTest runs before each test in the suite.
func (s *ResolverTestSuite) SetupTest() {
	s.Env = NewTestEnv(s.T())
}

// TearDownTest runs after each test in the suite.
func (s *ResolverTestSuite) TearDownTest() {
	if s.Env != nil {
		s.Env.Cleanup()
	}
}

// NewResolverForTest creates a new resolver with the test dependencies.
func NewResolverForTest(deps *resolver.Dependencies) generated.ResolverRoot {
	return resolver.New(deps)
}
