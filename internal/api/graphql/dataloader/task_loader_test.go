package dataloader_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/dataloader"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// setupTaskTestDB creates an in-memory database with task and connection services for testing.
func setupTaskTestDB(t *testing.T) (*ent.Client, *services.TaskService, *services.ConnectionService) {
	t.Helper()

	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug, nil)

	client, err := db.InitDB(db.InitDBOptions{
		DSN:           db.InMemoryDSN(),
		MigrationMode: db.MigrationModeAuto,
	})
	require.NoError(t, err)

	encryptor, err := crypto.NewEncryptor("test-encryption-key-32-bytes!!")
	require.NoError(t, err)

	connectionService := services.NewConnectionService(client, encryptor)
	taskService := services.NewTaskService(client)

	// Install DBStorage for rclone configuration
	storage := rclone.NewDBStorage(connectionService)
	storage.Install()

	t.Cleanup(func() {
		client.Close()
	})

	return client, taskService, connectionService
}

// createTestConnectionForTask creates a test connection for task tests.
func createTestConnectionForTask(t *testing.T, connectionService *services.ConnectionService) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	conn, err := connectionService.CreateConnection(ctx, "test-connection", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)
	return conn.ID
}

func TestTaskLoader_Load_ExistingTask(t *testing.T) {
	client, taskService, connectionService := setupTaskTestDB(t)
	ctx := context.Background()

	// Create a connection first
	connID := createTestConnectionForTask(t, connectionService)

	// Create a test task
	task, err := taskService.CreateTask(
		ctx,
		"test-task",
		"/tmp/source",
		connID,
		"/remote",
		"UPLOAD",
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// Create loader
	loader := dataloader.NewTaskLoader(client)

	// Load the task
	result, err := loader.Load(ctx, task.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, task.ID, result.ID)
	assert.Equal(t, "test-task", result.Name)
	assert.Equal(t, "/tmp/source", result.SourcePath)
	assert.Equal(t, "/remote", result.RemotePath)
}

func TestTaskLoader_Load_NonExistentTask(t *testing.T) {
	client, _, _ := setupTaskTestDB(t)
	ctx := context.Background()

	// Create loader
	loader := dataloader.NewTaskLoader(client)

	// Try to load a non-existent task
	nonExistentID := uuid.New()
	result, err := loader.Load(ctx, nonExistentID)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestTaskLoader_LoadAll_MultipleTasks(t *testing.T) {
	client, taskService, connectionService := setupTaskTestDB(t)
	ctx := context.Background()

	// Create a connection first
	connID := createTestConnectionForTask(t, connectionService)

	// Create multiple test tasks
	task1, err := taskService.CreateTask(ctx, "task-1", "/source1", connID, "/remote1", "UPLOAD", "", false, nil)
	require.NoError(t, err)

	task2, err := taskService.CreateTask(ctx, "task-2", "/source2", connID, "/remote2", "DOWNLOAD", "", false, nil)
	require.NoError(t, err)

	task3, err := taskService.CreateTask(ctx, "task-3", "/source3", connID, "/remote3", "BIDIRECTIONAL", "", false, nil)
	require.NoError(t, err)

	// Create loader
	loader := dataloader.NewTaskLoader(client)

	// Load all tasks in batch
	ids := []uuid.UUID{task1.ID, task2.ID, task3.ID}
	results, err := loader.LoadAll(ctx, ids)

	require.NoError(t, err)
	require.Len(t, results, 3)

	// Check results are in correct order
	assert.Equal(t, task1.ID, results[0].ID)
	assert.Equal(t, task2.ID, results[1].ID)
	assert.Equal(t, task3.ID, results[2].ID)

	// Verify task data
	assert.Equal(t, "task-1", results[0].Name)
	assert.Equal(t, "task-2", results[1].Name)
	assert.Equal(t, "task-3", results[2].Name)
}

func TestTaskLoader_LoadAll_MixedExistingAndNonExistent(t *testing.T) {
	client, taskService, connectionService := setupTaskTestDB(t)
	ctx := context.Background()

	// Create a connection first
	connID := createTestConnectionForTask(t, connectionService)

	// Create one test task
	task1, err := taskService.CreateTask(ctx, "task-1", "/source1", connID, "/remote1", "UPLOAD", "", false, nil)
	require.NoError(t, err)

	nonExistentID := uuid.New()

	// Create loader
	loader := dataloader.NewTaskLoader(client)

	// Load batch with mix of existing and non-existing
	ids := []uuid.UUID{task1.ID, nonExistentID}
	results, err := loader.LoadAll(ctx, ids)

	// LoadAll returns error if any item fails
	assert.Error(t, err)

	// Results will still have the length matching input
	require.Len(t, results, 2)

	// First should be present
	assert.NotNil(t, results[0])
	assert.Equal(t, task1.ID, results[0].ID)
}

func TestTaskLoader_LoadAll_PreservesOrder(t *testing.T) {
	client, taskService, connectionService := setupTaskTestDB(t)
	ctx := context.Background()

	// Create a connection first
	connID := createTestConnectionForTask(t, connectionService)

	// Create tasks in specific order
	var tasks []*ent.Task
	for i := 0; i < 5; i++ {
		task, err := taskService.CreateTask(
			ctx,
			"task-"+string(rune('A'+i)),
			"/source"+string(rune('A'+i)),
			connID,
			"/remote"+string(rune('A'+i)),
			"UPLOAD",
			"",
			false,
			nil,
		)
		require.NoError(t, err)
		tasks = append(tasks, task)
	}

	// Create loader
	loader := dataloader.NewTaskLoader(client)

	// Request in reverse order
	ids := []uuid.UUID{
		tasks[4].ID,
		tasks[2].ID,
		tasks[0].ID,
		tasks[3].ID,
		tasks[1].ID,
	}

	results, err := loader.LoadAll(ctx, ids)

	require.NoError(t, err)
	require.Len(t, results, 5)

	// Results should match request order
	assert.Equal(t, tasks[4].ID, results[0].ID)
	assert.Equal(t, tasks[2].ID, results[1].ID)
	assert.Equal(t, tasks[0].ID, results[2].ID)
	assert.Equal(t, tasks[3].ID, results[3].ID)
	assert.Equal(t, tasks[1].ID, results[4].ID)
}

func TestTaskLoader_LoadAll_EmptySlice(t *testing.T) {
	client, _, _ := setupTaskTestDB(t)
	ctx := context.Background()

	// Create loader
	loader := dataloader.NewTaskLoader(client)

	// Load with empty slice
	results, err := loader.LoadAll(ctx, []uuid.UUID{})

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestTaskLoader_LoadAll_DuplicateIDs(t *testing.T) {
	client, taskService, connectionService := setupTaskTestDB(t)
	ctx := context.Background()

	// Create a connection first
	connID := createTestConnectionForTask(t, connectionService)

	// Create a test task
	task, err := taskService.CreateTask(ctx, "test-task", "/source", connID, "/remote", "UPLOAD", "", false, nil)
	require.NoError(t, err)

	// Create loader
	loader := dataloader.NewTaskLoader(client)

	// Load same ID multiple times
	ids := []uuid.UUID{task.ID, task.ID, task.ID}
	results, err := loader.LoadAll(ctx, ids)

	require.NoError(t, err)
	require.Len(t, results, 3)

	// All should return the same task
	for _, result := range results {
		assert.NotNil(t, result)
		assert.Equal(t, task.ID, result.ID)
		assert.Equal(t, "test-task", result.Name)
	}
}

func TestTaskLoader_Load_TaskWithSchedule(t *testing.T) {
	client, taskService, connectionService := setupTaskTestDB(t)
	ctx := context.Background()

	// Create a connection first
	connID := createTestConnectionForTask(t, connectionService)

	// Create a task with schedule
	task, err := taskService.CreateTask(
		ctx,
		"scheduled-task",
		"/source",
		connID,
		"/remote",
		"UPLOAD",
		"0 * * * *", // Every hour
		false,
		nil,
	)
	require.NoError(t, err)

	// Create loader
	loader := dataloader.NewTaskLoader(client)

	// Load the task
	result, err := loader.Load(ctx, task.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "0 * * * *", result.Schedule)
}

func TestTaskLoader_Load_TaskWithRealtime(t *testing.T) {
	client, taskService, connectionService := setupTaskTestDB(t)
	ctx := context.Background()

	// Create a connection first
	connID := createTestConnectionForTask(t, connectionService)

	// Create a task with realtime enabled
	task, err := taskService.CreateTask(
		ctx,
		"realtime-task",
		"/source",
		connID,
		"/remote",
		"UPLOAD",
		"",
		true, // realtime enabled
		nil,
	)
	require.NoError(t, err)

	// Create loader
	loader := dataloader.NewTaskLoader(client)

	// Load the task
	result, err := loader.Load(ctx, task.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Realtime)
}
