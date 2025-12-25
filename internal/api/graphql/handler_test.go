package graphql_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/dataloader"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/resolver"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"github.com/xzzpig/rclone-sync/internal/core/runner"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// setupHandlerTest creates a test environment for handler tests.
func setupHandlerTest(t *testing.T) (*gin.Engine, *ent.Client, func()) {
	t.Helper()

	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug, nil)

	client, err := db.InitDB(db.InitDBOptions{
		DSN:           db.InMemoryDSN(),
		MigrationMode: db.MigrationModeAuto,
	})
	require.NoError(t, err)

	appDataDir := t.TempDir()

	jobService := services.NewJobService(client)
	taskService := services.NewTaskService(client)

	encryptor, err := crypto.NewEncryptor("test-encryption-key-32-bytes!!")
	require.NoError(t, err)
	connectionService := services.NewConnectionService(client, encryptor)

	storage := rclone.NewDBStorage(connectionService)
	storage.Install()

	syncEngine := rclone.NewSyncEngine(jobService, nil, nil, appDataDir, false)
	runnerInstance := runner.NewRunner(syncEngine)

	mockWatcher := &mockWatcher{}
	mockScheduler := &mockScheduler{}

	jobProgressBus := subscription.NewJobProgressBus()

	deps := &resolver.Dependencies{
		SyncEngine:        syncEngine,
		Runner:            runnerInstance,
		JobService:        jobService,
		Watcher:           mockWatcher,
		Scheduler:         mockScheduler,
		TaskService:       taskService,
		ConnectionService: connectionService,
		Encryptor:         encryptor,
		JobProgressBus:    jobProgressBus,
	}

	srv := graphql.NewHandler(deps)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(dataloader.Middleware(client))
	router.POST("/graphql", graphql.GinHandler(srv))
	router.GET("/graphql/playground", graphql.PlaygroundHandler("/graphql"))

	cleanup := func() {
		client.Close()
	}

	return router, client, cleanup
}

// mockWatcher is a mock implementation of ports.Watcher for testing.
type mockWatcher struct{}

func (m *mockWatcher) Start()                          {}
func (m *mockWatcher) Stop()                           {}
func (m *mockWatcher) AddTask(task *ent.Task) error    { return nil }
func (m *mockWatcher) RemoveTask(task *ent.Task) error { return nil }

var _ ports.Watcher = (*mockWatcher)(nil)

// mockScheduler is a mock implementation of ports.Scheduler for testing.
type mockScheduler struct{}

func (m *mockScheduler) Start()                          {}
func (m *mockScheduler) Stop()                           {}
func (m *mockScheduler) AddTask(task *ent.Task) error    { return nil }
func (m *mockScheduler) RemoveTask(task *ent.Task) error { return nil }

var _ ports.Scheduler = (*mockScheduler)(nil)

// GraphQLRequest represents a GraphQL request body.
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response body.
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data,omitempty"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func executeGraphQL(t *testing.T, router *gin.Engine, req GraphQLRequest) *GraphQLResponse {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(http.MethodPost, "/graphql", bytes.NewBuffer(body))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	var resp GraphQLResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return &resp
}

func TestHandler_IntrospectionQuery(t *testing.T) {
	router, _, cleanup := setupHandlerTest(t)
	defer cleanup()

	resp := executeGraphQL(t, router, GraphQLRequest{
		Query: `query IntrospectionQuery { __schema { types { name } } }`,
	})

	assert.Empty(t, resp.Errors)
	assert.NotEmpty(t, resp.Data)
}

func TestHandler_SimpleQuery(t *testing.T) {
	router, _, cleanup := setupHandlerTest(t)
	defer cleanup()

	resp := executeGraphQL(t, router, GraphQLRequest{
		Query: `query { task { list { totalCount items { id name } } } }`,
	})

	assert.Empty(t, resp.Errors)
	assert.NotEmpty(t, resp.Data)

	// Parse the response data
	var data struct {
		Task struct {
			List struct {
				TotalCount int `json:"totalCount"`
				Items      []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"items"`
			} `json:"list"`
		} `json:"task"`
	}
	err := json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, 0, data.Task.List.TotalCount)
}

func TestHandler_ConnectionQuery(t *testing.T) {
	router, client, cleanup := setupHandlerTest(t)
	defer cleanup()

	// Create a test connection
	encryptor, _ := crypto.NewEncryptor("test-encryption-key-32-bytes!!")
	connectionService := services.NewConnectionService(client, encryptor)
	storage := rclone.NewDBStorage(connectionService)
	storage.Install()

	_, err := connectionService.CreateConnection(context.Background(), "test-conn", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	resp := executeGraphQL(t, router, GraphQLRequest{
		Query: `query { connection { list { totalCount items { id name type } } } }`,
	})

	assert.Empty(t, resp.Errors)

	var data struct {
		Connection struct {
			List struct {
				TotalCount int `json:"totalCount"`
				Items      []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"items"`
			} `json:"list"`
		} `json:"connection"`
	}
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, 1, data.Connection.List.TotalCount)
	assert.Equal(t, "test-conn", data.Connection.List.Items[0].Name)
}

func TestHandler_ProviderQuery(t *testing.T) {
	router, _, cleanup := setupHandlerTest(t)
	defer cleanup()

	resp := executeGraphQL(t, router, GraphQLRequest{
		Query: `query { provider { list { name description } } }`,
	})

	assert.Empty(t, resp.Errors)

	var data struct {
		Provider struct {
			List []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"list"`
		} `json:"provider"`
	}
	err := json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.NotEmpty(t, data.Provider.List)
}

func TestHandler_MalformedQuery(t *testing.T) {
	router, _, cleanup := setupHandlerTest(t)
	defer cleanup()

	resp := executeGraphQL(t, router, GraphQLRequest{
		Query: `query { invalid { query }`,
	})

	assert.NotEmpty(t, resp.Errors)
}

func TestHandler_UnknownField(t *testing.T) {
	router, _, cleanup := setupHandlerTest(t)
	defer cleanup()

	resp := executeGraphQL(t, router, GraphQLRequest{
		Query: `query { task { list { unknownField } } }`,
	})

	assert.NotEmpty(t, resp.Errors)
}

func TestHandler_PlaygroundEndpoint(t *testing.T) {
	router, _, cleanup := setupHandlerTest(t)
	defer cleanup()

	req, err := http.NewRequest(http.MethodGet, "/graphql/playground", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "GraphQL Playground")
}

func TestHandler_HTTPMethods(t *testing.T) {
	router, _, cleanup := setupHandlerTest(t)
	defer cleanup()

	// Test GET method (should work for queries)
	req, err := http.NewRequest(http.MethodGet, "/graphql?query="+`{task{list{totalCount}}}`, nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// GET requests should return 404 since we only registered POST
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ContentTypeValidation(t *testing.T) {
	router, _, cleanup := setupHandlerTest(t)
	defer cleanup()

	body := []byte(`{"query": "{ task { list { totalCount } } }"}`)
	req, err := http.NewRequest(http.MethodPost, "/graphql", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// gqlgen rejects invalid content types (text/plain is not a valid GraphQL content type)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PaginationQuery(t *testing.T) {
	router, _, cleanup := setupHandlerTest(t)
	defer cleanup()

	resp := executeGraphQL(t, router, GraphQLRequest{
		Query: `query { 
			task { 
				list(pagination: { limit: 10, offset: 0 }) { 
					totalCount 
					pageInfo {
						limit
						offset
						hasNextPage
						hasPreviousPage
					}
				} 
			} 
		}`,
	})

	assert.Empty(t, resp.Errors)

	var data struct {
		Task struct {
			List struct {
				TotalCount int `json:"totalCount"`
				PageInfo   struct {
					Limit           int  `json:"limit"`
					Offset          int  `json:"offset"`
					HasNextPage     bool `json:"hasNextPage"`
					HasPreviousPage bool `json:"hasPreviousPage"`
				} `json:"pageInfo"`
			} `json:"list"`
		} `json:"task"`
	}
	err := json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, 10, data.Task.List.PageInfo.Limit)
	assert.Equal(t, 0, data.Task.List.PageInfo.Offset)
	assert.False(t, data.Task.List.PageInfo.HasNextPage)
	assert.False(t, data.Task.List.PageInfo.HasPreviousPage)
}
