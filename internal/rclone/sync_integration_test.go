package rclone_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
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
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
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
	syncEngine := rclone.NewSyncEngine(jobService, nil, nil, dataDir, false, 0)

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

// TestSyncEngine_RunTask_AutoDeleteEmptyJob tests the auto-delete empty job logic.
// This test covers the auto-delete logic in sync.go lines 285-294.
func TestSyncEngine_RunTask_AutoDeleteEmptyJob(t *testing.T) {
	tests := []struct {
		name                string
		autoDeleteEmptyJobs bool
		hasFile             bool // whether to create a file to transfer
		expectJobDeleted    bool
		expectFiles         int
		expectBytes         int64
	}{
		{
			name:                "empty job deleted when autoDelete enabled",
			autoDeleteEmptyJobs: true,
			hasFile:             false,
			expectJobDeleted:    true,
		},
		{
			name:                "empty job kept when autoDelete disabled",
			autoDeleteEmptyJobs: false,
			hasFile:             false,
			expectJobDeleted:    false,
			expectFiles:         0,
			expectBytes:         0,
		},
		{
			name:                "non-empty job kept even when autoDelete enabled",
			autoDeleteEmptyJobs: true,
			hasFile:             true,
			expectJobDeleted:    false,
			expectFiles:         1,
			expectBytes:         11, // len("hello world")
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connService, taskService, jobService, _ := setupIntegrationTest(t)
			ctx := context.Background()

			// 1. Setup test directories
			sourceDir := t.TempDir()
			destDir := t.TempDir()

			// Create a test file if needed
			if tt.hasFile {
				testFilePath := filepath.Join(sourceDir, "test.txt")
				err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
				require.NoError(t, err)
			}

			// 2. Create Connection and Task via ConnectionService
			testConn, err := connService.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"})
			require.NoError(t, err)

			testTask, err := taskService.CreateTask(ctx,
				tt.name,
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
			syncEngine := rclone.NewSyncEngine(jobService, nil, nil, dataDir, tt.autoDeleteEmptyJobs, 0)

			// 4. Reload task with Connection edge before running
			testTask, err = taskService.GetTaskWithConnection(ctx, testTask.ID)
			require.NoError(t, err)

			// 5. Run the task
			err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
			require.NoError(t, err)

			// 6. Verify results
			jobs, err := jobService.ListJobs(ctx, &testTask.ID, nil, 10, 0)
			require.NoError(t, err)

			if tt.expectJobDeleted {
				assert.Empty(t, jobs, "Job should be auto-deleted")
			} else {
				require.Len(t, jobs, 1, "Job should exist in database")
				job := jobs[0]
				assert.Equal(t, string(model.JobStatusSuccess), string(job.Status), "Job status should be success")
				assert.Equal(t, tt.expectFiles, job.FilesTransferred, "FilesTransferred mismatch")
				assert.Equal(t, tt.expectBytes, job.BytesTransferred, "BytesTransferred mismatch")
			}
		})
	}
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
	syncEngine := rclone.NewSyncEngine(jobService, nil, nil, dataDir, false, 0)

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
	syncEngine := rclone.NewSyncEngine(jobService, nil, nil, dataDir, false, 0)

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
	syncEngine := rclone.NewSyncEngine(jobService, nil, nil, dataDir, false, 0)

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

// TestSyncEngine_RunTask_NoDelete tests the noDelete option behavior.
// When noDelete=true, files deleted from source should NOT be deleted from destination.
// When noDelete=false (default), files deleted from source should be deleted from destination.
// This test covers the noDelete logic in sync.go runOneWay function.
func TestSyncEngine_RunTask_NoDelete(t *testing.T) {
	tests := []struct {
		name                 string
		noDelete             bool
		expectDestFileExists bool // whether the deleted source file should still exist in dest
	}{
		{
			name:                 "noDelete=true preserves destination files",
			noDelete:             true,
			expectDestFileExists: true, // file should still exist in dest
		},
		{
			name:                 "noDelete=false (default) deletes destination files",
			noDelete:             false,
			expectDestFileExists: false, // file should be deleted from dest
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connService, taskService, jobService, _ := setupIntegrationTest(t)
			ctx := context.Background()

			// 1. Setup test directories
			sourceDir := t.TempDir()
			destDir := t.TempDir()

			// Create test files - one to keep, one to delete later
			keepFilePath := filepath.Join(sourceDir, "keep.txt")
			err := os.WriteFile(keepFilePath, []byte("keep this file"), 0644)
			require.NoError(t, err)

			deleteFilePath := filepath.Join(sourceDir, "delete.txt")
			err = os.WriteFile(deleteFilePath, []byte("delete this file"), 0644)
			require.NoError(t, err)

			// 2. Create Connection and Task
			testConn, err := connService.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"})
			require.NoError(t, err)

			// Create options with noDelete setting
			options := &model.TaskSyncOptions{
				NoDelete: &tt.noDelete,
			}

			testTask, err := taskService.CreateTask(ctx,
				tt.name,
				sourceDir,
				testConn.ID,
				destDir,
				string(model.SyncDirectionUpload), // noDelete only applies to one-way sync
				"",
				false,
				options,
			)
			require.NoError(t, err)

			// 3. Setup SyncEngine
			dataDir := t.TempDir()
			syncEngine := rclone.NewSyncEngine(jobService, nil, nil, dataDir, false, 0)

			// 4. Reload task with Connection edge
			testTask, err = taskService.GetTaskWithConnection(ctx, testTask.ID)
			require.NoError(t, err)

			// 5. Run initial sync - both files should be copied to dest
			err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
			require.NoError(t, err)

			// Verify both files exist in destination after initial sync
			destKeepPath := filepath.Join(destDir, "keep.txt")
			destDeletePath := filepath.Join(destDir, "delete.txt")

			_, err = os.Stat(destKeepPath)
			assert.NoError(t, err, "keep.txt should exist in destination after initial sync")
			_, err = os.Stat(destDeletePath)
			assert.NoError(t, err, "delete.txt should exist in destination after initial sync")

			// 6. Delete the file from source
			err = os.Remove(deleteFilePath)
			require.NoError(t, err)

			// Verify source file is deleted
			_, err = os.Stat(deleteFilePath)
			assert.True(t, os.IsNotExist(err), "delete.txt should not exist in source after deletion")

			// 7. Run sync again after deletion
			err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
			require.NoError(t, err)

			// 8. Verify results based on noDelete setting
			// keep.txt should always exist
			_, err = os.Stat(destKeepPath)
			assert.NoError(t, err, "keep.txt should always exist in destination")

			// delete.txt behavior depends on noDelete setting
			_, err = os.Stat(destDeletePath)
			if tt.expectDestFileExists {
				assert.NoError(t, err, "delete.txt should still exist in destination when noDelete=true")
			} else {
				assert.True(t, os.IsNotExist(err), "delete.txt should be deleted from destination when noDelete=false")
			}
		})
	}
}

// TestSyncEngine_RunTask_ProgressEvents tests that JobProgressEvent and TransferProgressEvent
// are properly published during sync operations.
func TestSyncEngine_RunTask_ProgressEvents(t *testing.T) {
	connService, taskService, jobService, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// 2. Create Connection and Task
	testConn, err := connService.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"})
	require.NoError(t, err)

	testTask, err := taskService.CreateTask(ctx,
		"TestProgressEventsSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Create event buses
	jobProgressBus := subscription.NewJobProgressBus()
	transferProgressBus := subscription.NewTransferProgressBus()

	// 4. Subscribe to events
	jobSub := jobProgressBus.Subscribe(nil)
	transferSub := transferProgressBus.Subscribe(nil)

	// 5. Collect events in background goroutines
	var jobEvents []*model.JobProgressEvent
	var transferEvents []*model.TransferProgressEvent
	var mu sync.Mutex
	done := make(chan struct{})

	go func() {
		for {
			select {
			case event, ok := <-jobSub.Events:
				if !ok {
					return
				}
				mu.Lock()
				jobEvents = append(jobEvents, event)
				mu.Unlock()
			case <-done:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case event, ok := <-transferSub.Events:
				if !ok {
					return
				}
				mu.Lock()
				transferEvents = append(transferEvents, event)
				mu.Unlock()
			case <-done:
				return
			}
		}
	}()

	// 6. Setup SyncEngine with real buses
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobService, jobProgressBus, transferProgressBus, dataDir, false, 0)

	// 7. Reload task with Connection edge before running
	testTask, err = taskService.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 8. Run the task
	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	// 9. Wait a bit for events to be delivered
	time.Sleep(200 * time.Millisecond)
	close(done)

	// 10. Cleanup subscriptions
	jobProgressBus.Unsubscribe(jobSub.ID)
	transferProgressBus.Unsubscribe(transferSub.ID)

	// 11. Verify JobProgressEvent
	mu.Lock()
	jobEventsCopy := make([]*model.JobProgressEvent, len(jobEvents))
	copy(jobEventsCopy, jobEvents)
	transferEventsCopy := make([]*model.TransferProgressEvent, len(transferEvents))
	copy(transferEventsCopy, transferEvents)
	mu.Unlock()

	t.Logf("Received %d JobProgressEvents", len(jobEventsCopy))
	t.Logf("Received %d TransferProgressEvents", len(transferEventsCopy))

	// Verify at least some job events were received
	require.NotEmpty(t, jobEventsCopy, "Should receive at least one JobProgressEvent")

	// Verify job event fields
	for _, event := range jobEventsCopy {
		assert.Equal(t, testTask.ID, event.TaskID, "TaskID should match")
		assert.Equal(t, testConn.ID, event.ConnectionID, "ConnectionID should match")
		assert.NotZero(t, event.JobID, "JobID should not be zero")
		assert.False(t, event.StartTime.IsZero(), "StartTime should be set")
		t.Logf("JobEvent: Status=%s, Files=%d/%d, Bytes=%d/%d",
			event.Status, event.FilesTransferred, event.FilesTotal, event.BytesTransferred, event.BytesTotal)
	}

	// Find the last job event - should be SUCCESS
	lastJobEvent := jobEventsCopy[len(jobEventsCopy)-1]
	assert.Equal(t, model.JobStatusSuccess, lastJobEvent.Status, "Final job status should be SUCCESS")
	assert.NotNil(t, lastJobEvent.EndTime, "EndTime should be set for final event")
	assert.Equal(t, 1, lastJobEvent.FilesTransferred, "Should have transferred 1 file")
	assert.Equal(t, int64(11), lastJobEvent.BytesTransferred, "Should have transferred 11 bytes")

	// 12. Verify TransferProgressEvent
	// Note: For small files, we might not see in-progress transfers,
	// but we should see at least the completion event (bytes == size)
	if len(transferEventsCopy) > 0 {
		for _, event := range transferEventsCopy {
			assert.Equal(t, testTask.ID, event.TaskID, "TaskID should match")
			assert.Equal(t, testConn.ID, event.ConnectionID, "ConnectionID should match")
			assert.NotZero(t, event.JobID, "JobID should not be zero")
			t.Logf("TransferEvent: JobID=%s, Transfers=%d", event.JobID, len(event.Transfers))

			for _, tr := range event.Transfers {
				t.Logf("  Transfer: Name=%s, Size=%d, Bytes=%d", tr.Name, tr.Size, tr.Bytes)
				assert.NotEmpty(t, tr.Name, "Transfer name should not be empty")
				assert.GreaterOrEqual(t, tr.Bytes, int64(0), "Bytes should be >= 0")
				assert.GreaterOrEqual(t, tr.Size, tr.Bytes, "Size should be >= Bytes")
			}
		}

		// Check if any transfer completed (bytes == size)
		foundCompleted := false
		for _, event := range transferEventsCopy {
			for _, tr := range event.Transfers {
				if tr.Bytes == tr.Size && tr.Size > 0 {
					foundCompleted = true
					assert.Equal(t, "test.txt", tr.Name, "Completed transfer should be test.txt")
					assert.Equal(t, int64(11), tr.Size, "File size should be 11 bytes")
					break
				}
			}
			if foundCompleted {
				break
			}
		}
		assert.True(t, foundCompleted, "Should find at least one completed transfer (bytes == size)")
	}
}
