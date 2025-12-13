package rclone_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// setupIntegrationTest initializes a real database and services for integration testing.
func setupIntegrationTest(t *testing.T) (*ent.Client, *services.JobService, func()) {
	// Use in-memory sqlite for testing
	client, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)

	// Run migrations
	require.NoError(t, client.Schema.Create(context.Background()))

	jobService := services.NewJobService(client)

	cleanup := func() {
		client.Close()
	}

	return client, jobService, cleanup
}

func TestSyncEngine_RunTask_Integration(t *testing.T) {
	client, jobService, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// 2. Create a test task in the database
	taskService := services.NewTaskService(client)
	testTask, err := taskService.CreateTask(context.Background(),
		"TestIntegrationSync",
		sourceDir,
		"local", // Using local backend for simplicity
		destDir,
		"bidirectional",
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine with real services
	dataDir := t.TempDir()

	// Create a dummy rclone.conf for the test
	rcloneConfPath := filepath.Join(dataDir, "rclone.conf")
	confContent := `[local]
type = local
`
	err = os.WriteFile(rcloneConfPath, []byte(confContent), 0644)
	require.NoError(t, err)
	rclone.InitConfig(rcloneConfPath)

	syncEngine := rclone.NewSyncEngine(jobService, dataDir)

	// 4. Run the task
	err = syncEngine.RunTask(context.Background(), testTask, "manual")
	require.NoError(t, err)

	// 5. Verify results
	// Check if file was synced
	destFilePath := filepath.Join(destDir, "test.txt")
	_, err = os.Stat(destFilePath)
	assert.NoError(t, err, "File should exist in destination")

	// Check database for job and logs
	jobs, err := jobService.ListJobs(context.Background(), &testTask.ID, "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1, "Should be one job in the database")

	job := jobs[0]
	assert.Equal(t, "success", string(job.Status), "Job status should be success")
	assert.Equal(t, 1, job.FilesTransferred, "Should have transferred one file")

	assert.Equal(t, int64(11), job.BytesTransferred, "Should have transferred 11 bytes")

	jobWithLogs, err := jobService.GetJobWithLogs(context.Background(), job.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, jobWithLogs.Edges.Logs, "Job should have logs")

	foundLog := false
	for _, log := range jobWithLogs.Edges.Logs {
		if log.Path == "test.txt" {
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Should find a log entry for test.txt")
}

func TestSyncEngine_RunTask_Failure(t *testing.T) {
	client, jobService, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// 1. Setup test directories, but do NOT create the source directory
	sourceDir := filepath.Join(t.TempDir(), "non_existent_source")
	destDir := t.TempDir()

	// 2. Create a test task in the database with the non-existent source path
	taskService := services.NewTaskService(client)
	testTask, err := taskService.CreateTask(context.Background(),
		"TestFailureSync",
		sourceDir,
		"local",
		destDir,
		"bidirectional",
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()

	rcloneConfPath := filepath.Join(dataDir, "rclone.conf")
	confContent := `[local]
type = local
`
	err = os.WriteFile(rcloneConfPath, []byte(confContent), 0644)
	require.NoError(t, err)
	rclone.InitConfig(rcloneConfPath)

	syncEngine := rclone.NewSyncEngine(jobService, dataDir)

	// 4. Run the task and expect an error
	err = syncEngine.RunTask(context.Background(), testTask, "manual")
	assert.Error(t, err, "RunTask should return an error for non-existent source")

	// 5. Verify results
	jobs, err := jobService.ListJobs(context.Background(), &testTask.ID, "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1, "Should be one job in the database")

	job := jobs[0]
	assert.Equal(t, "failed", string(job.Status), "Job status should be failed")
	assert.NotEmpty(t, job.Errors, "Job should have an error message")
}

func TestSyncEngine_RunTask_Cancel(t *testing.T) {
	client, jobService, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// 2. Create a test task in the database
	taskService := services.NewTaskService(client)
	testTask, err := taskService.CreateTask(context.Background(),
		"TestCancelSync",
		sourceDir,
		"local",
		destDir,
		"bidirectional",
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()

	rcloneConfPath := filepath.Join(dataDir, "rclone.conf")
	confContent := `[local]
type = local
`
	err = os.WriteFile(rcloneConfPath, []byte(confContent), 0644)
	require.NoError(t, err)
	rclone.InitConfig(rcloneConfPath)

	syncEngine := rclone.NewSyncEngine(jobService, dataDir)

	// 4. Create a context that is already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// 5. Run the task with the cancelled context
	err = syncEngine.RunTask(ctx, testTask, "manual")
	assert.Error(t, err, "RunTask should return an error for a cancelled context")
	assert.Contains(t, err.Error(), "context canceled", "Error should mention context cancellation")

	// 6. Verify that no job was created because the context was cancelled before any work
	jobs, err := jobService.ListJobs(context.Background(), &testTask.ID, "", 10, 0)
	require.NoError(t, err)
	assert.Empty(t, jobs, "No job should be created if the context is already cancelled")
}
