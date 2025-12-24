package rclone_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"
	"github.com/xzzpig/rclone-sync/internal/rclone/testutil"

	_ "github.com/rclone/rclone/backend/local"
)

// setupIntegrationTest initializes a real database, services and DBStorage for integration testing.
func setupIntegrationTest(t *testing.T) (*services.ConnectionService, *services.TaskService, *services.JobService, *rclone.DBStorage) {
	t.Helper()

	// Create test database client
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	// Create encryptor (plaintext mode for testing)
	encryptor, err := crypto.NewEncryptor("")
	require.NoError(t, err)

	// Create services
	connService := services.NewConnectionService(client, encryptor)
	taskService := services.NewTaskService(client)
	jobService := services.NewJobService(client)

	// Create DBStorage and install it
	storage := rclone.NewDBStorage(connService)
	storage.Install()

	return connService, taskService, jobService, storage
}

func TestSyncEngine_RunTask_Integration(t *testing.T) {
	connService, taskService, jobService, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// 2. Create Connection and Task via ConnectionService (this goes to database)
	testConn, err := connService.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"})
	require.NoError(t, err)

	testTask, err := taskService.CreateTask(ctx,
		"TestIntegrationSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobService, nil, dataDir)

	// 4. Reload task with Connection edge before running
	testTask, err = taskService.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 5. Run the task - this should use DBStorage to read the connection config
	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	// 6. Verify results
	// Check if file was synced
	destFilePath := filepath.Join(destDir, "test.txt")
	_, err = os.Stat(destFilePath)
	assert.NoError(t, err, "File should exist in destination")

	// Check database for job and logs
	jobs, err := jobService.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1, "Should be one job in the database")

	job := jobs[0]
	assert.Equal(t, string(model.JobStatusSuccess), string(job.Status), "Job status should be success")
	assert.Equal(t, 1, job.FilesTransferred, "Should have transferred one file")
	assert.Equal(t, int64(11), job.BytesTransferred, "Should have transferred 11 bytes")

	jobWithLogs, err := jobService.GetJobWithLogs(ctx, job.ID)
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
	connService, taskService, jobService, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories, but do NOT create the source directory
	sourceDir := filepath.Join(t.TempDir(), "non_existent_source")
	destDir := t.TempDir()

	// 2. Create Connection and Task via ConnectionService (this goes to database)
	testConn, err := connService.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"})
	require.NoError(t, err)

	testTask, err := taskService.CreateTask(ctx,
		"TestFailureSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobService, nil, dataDir)

	// 4. Reload task with Connection edge before running
	testTask, err = taskService.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 5. Run the task and expect an error
	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	assert.Error(t, err, "RunTask should return an error for non-existent source")

	// 6. Verify results
	jobs, err := jobService.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1, "Should be one job in the database")

	job := jobs[0]
	assert.Equal(t, string(model.JobStatusFailed), string(job.Status), "Job status should be failed")
	assert.NotEmpty(t, job.Errors, "Job should have an error message")
}

func TestSyncEngine_RunTask_Cancel(t *testing.T) {
	connService, taskService, jobService, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// 2. Create Connection and Task via ConnectionService (this goes to database)
	testConn, err := connService.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"})
	require.NoError(t, err)

	testTask, err := taskService.CreateTask(ctx,
		"TestCancelSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobService, nil, dataDir)

	// 4. Reload task with Connection edge before running
	testTask, err = taskService.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 5. Create a context that is already cancelled
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// 6. Run the task with the cancelled context
	err = syncEngine.RunTask(cancelCtx, testTask, model.JobTriggerManual)
	assert.Error(t, err, "RunTask should return an error for a cancelled context")
	assert.Contains(t, err.Error(), "context canceled", "Error should mention context cancellation")

	// 7. Verify that no job was created because the context was cancelled before any work
	jobs, err := jobService.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, jobs, "No job should be created if the context is already cancelled")
}

// TestSyncEngine_RunTask_CancelDuringSync tests that cancelling a context during an active sync operation
// properly handles the cancellation and marks the job as cancelled.
// This test covers the cancellation logic in sync.go lines 185-199.
func TestSyncEngine_RunTask_CancelDuringSync(t *testing.T) {
	connService, taskService, jobService, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup control channels for slowfs
	startedCh := make(chan struct{}, 10)
	blockCh := make(chan struct{})
	testutil.SetSlowFsController(startedCh, blockCh)
	defer testutil.ClearSlowFsController()

	// 2. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// 3. Create slowfs connection
	slowConn, err := connService.CreateConnection(ctx, "slowlocal", "slowfs", map[string]string{
		"type":   "slowfs",
		"remote": "/",
	})
	require.NoError(t, err)

	// 4. Create task using slowfs - use upload direction to trigger Put on destination
	testTask, err := taskService.CreateTask(ctx,
		"TestCancelDuringSync",
		sourceDir,
		slowConn.ID,
		destDir,
		string(model.SyncDirectionUpload),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 5. Reload task with Connection edge
	testTask, err = taskService.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 6. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobService, nil, dataDir)

	// 7. Create cancellable context
	taskCtx, cancel := context.WithCancel(context.Background())

	// 8. Run task in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- syncEngine.RunTask(taskCtx, testTask, model.JobTriggerManual)
	}()

	// 9. Wait for sync operation to actually start
	t.Log("Waiting for sync to start...")
	select {
	case <-startedCh:
		t.Log("Sync started and is now blocking")
	case <-time.After(5 * time.Second):
		t.Fatal("Sync did not start within timeout")
	}

	// 10. Give it a moment to ensure job is created and status is "running"
	time.Sleep(200 * time.Millisecond)

	// 11. Verify job is running
	jobs, err := jobService.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, jobs, 1, "Should have exactly one job")
	jobID := jobs[0].ID
	assert.Equal(t, string(model.JobStatusRunning), string(jobs[0].Status), "Job should be running before cancellation")
	t.Logf("Job ID: %s, status: %s", jobID, jobs[0].Status)

	// 12. Cancel the context while sync is in progress
	t.Log("Cancelling context...")
	cancel()

	// 13. Wait for RunTask to return
	select {
	case err := <-errCh:
		assert.Error(t, err, "RunTask should return an error")
		// The error may be wrapped by rclone, so we just verify an error was returned
		t.Logf("RunTask returned error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("RunTask did not return within timeout after cancellation")
	}

	// 14. Give it time to update job status
	time.Sleep(500 * time.Millisecond)

	// 15. Verify job status was updated to cancelled
	jobs, err = jobService.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, jobs, 1, "Should still have one job")

	job := jobs[0]
	assert.Equal(t, jobID, job.ID, "Should be the same job")
	assert.Equal(t, string(model.JobStatusCancelled), string(job.Status), "Job status should be cancelled after context cancellation")
	assert.Contains(t, job.Errors, "cancelled", "Job errors should mention cancellation")
	t.Logf("Final job status: %s, errors: %s", job.Status, job.Errors)
}
