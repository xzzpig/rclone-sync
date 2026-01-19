package rclone_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/db/query"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/hook"
	"github.com/xzzpig/rclone-sync/internal/rclone"
	"github.com/xzzpig/rclone-sync/internal/rclone/testutil"

	_ "github.com/rclone/rclone/backend/local"
	_ "github.com/xzzpig/rclone-sync/internal/rclone/backend/metacache"
	_ "github.com/xzzpig/rclone-sync/internal/rclone/backend/notifylocal"
)

// setupIntegrationTest initializes a real database, queries and DBStorage for integration testing.
func setupIntegrationTest(t *testing.T) (*query.ConnectionQuery, *query.TaskQuery, *query.JobQuery, *rclone.DBStorage) {
	t.Helper()

	// Create test database client
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	t.Cleanup(func() { client.Close() })

	// Create encryptor (plaintext mode for testing)
	encryptor, err := crypto.NewEncryptor("")
	require.NoError(t, err)

	// Create queries
	connQuery := query.NewConnectionQuery(client, encryptor)
	taskQuery := query.NewTaskQuery(client)
	jobQuery := query.NewJobQuery(client)

	// Create DBStorage and install it (use temp dir for cache)
	storage := rclone.NewDBStorage(connQuery, t.TempDir())
	storage.Install()

	return connQuery, taskQuery, jobQuery, storage
}

func TestSyncEngine_RunTask_Integration(t *testing.T) {
	connQuery, taskQuery, jobQuery, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// 2. Create Connection and Task via ConnectionQuery (this goes to database)
	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestIntegrationSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, nil)

	// 4. Reload task with Connection edge before running
	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
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
	jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1, "Should be one job in the database")

	job := jobs[0]
	assert.Equal(t, string(model.JobStatusSuccess), string(job.Status), "Job status should be success")
	assert.Equal(t, 1, job.FilesTransferred, "Should have transferred one file")
	assert.Equal(t, int64(11), job.BytesTransferred, "Should have transferred 11 bytes")

	jobWithLogs, err := jobQuery.GetJobWithLogs(ctx, job.ID)
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

// TestSyncEngine_RunTask_AutoDeleteEmptyJob tests the rolling replacement logic for empty jobs.
// When autoDeleteEmptyJobs is enabled, completing a job deletes the PREVIOUS empty job, not the current one.
func TestSyncEngine_RunTask_AutoDeleteEmptyJob(t *testing.T) {
	tests := []struct {
		name                string
		autoDeleteEmptyJobs bool
		firstJobHasFile     bool
		secondJobHasFile    bool
		expectJobCount      int
	}{
		{
			name:                "two empty jobs: first deleted, second kept",
			autoDeleteEmptyJobs: true,
			firstJobHasFile:     false,
			secondJobHasFile:    false,
			expectJobCount:      1,
		},
		{
			name:                "empty then non-empty: first deleted, second kept",
			autoDeleteEmptyJobs: true,
			firstJobHasFile:     false,
			secondJobHasFile:    true,
			expectJobCount:      1,
		},
		{
			name:                "non-empty then empty: both kept",
			autoDeleteEmptyJobs: true,
			firstJobHasFile:     true,
			secondJobHasFile:    false,
			expectJobCount:      2,
		},
		{
			name:                "autoDelete disabled: both kept",
			autoDeleteEmptyJobs: false,
			firstJobHasFile:     false,
			secondJobHasFile:    false,
			expectJobCount:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connQuery, taskQuery, jobQuery, _ := setupIntegrationTest(t)
			ctx := context.Background()

			sourceDir := t.TempDir()
			destDir := t.TempDir()

			testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
			require.NoError(t, err)

			testTask, err := taskQuery.CreateTask(ctx,
				tt.name,
				sourceDir,
				testConn.ID,
				destDir,
				string(model.SyncDirectionUpload),
				"",
				false,
				true,
				nil,
			)
			require.NoError(t, err)

			dataDir := t.TempDir()
			syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, tt.autoDeleteEmptyJobs, 0, nil)

			testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
			require.NoError(t, err)

			if tt.firstJobHasFile {
				err = os.WriteFile(filepath.Join(sourceDir, "first.txt"), []byte("first"), 0644)
				require.NoError(t, err)
			}

			err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
			require.NoError(t, err)

			if tt.firstJobHasFile {
				os.Remove(filepath.Join(sourceDir, "first.txt"))
			}
			if tt.secondJobHasFile {
				err = os.WriteFile(filepath.Join(sourceDir, "second.txt"), []byte("second"), 0644)
				require.NoError(t, err)
			}

			err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
			require.NoError(t, err)

			jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
			require.NoError(t, err)
			assert.Len(t, jobs, tt.expectJobCount)
		})
	}
}

func TestSyncEngine_RunTask_Failure(t *testing.T) {
	connQuery, taskQuery, jobQuery, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories, but do NOT create the source directory
	sourceDir := filepath.Join(t.TempDir(), "non_existent_source")
	destDir := t.TempDir()

	// 2. Create Connection and Task via ConnectionQuery (this goes to database)
	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestFailureSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, nil)

	// 4. Reload task with Connection edge before running
	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 5. Run the task and expect an error
	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	assert.Error(t, err, "RunTask should return an error for non-existent source")

	// 6. Verify results
	jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1, "Should be one job in the database")

	job := jobs[0]
	assert.Equal(t, string(model.JobStatusFailed), string(job.Status), "Job status should be failed")
	assert.NotEmpty(t, job.Errors, "Job should have an error message")
}

func TestSyncEngine_RunTask_Cancel(t *testing.T) {
	connQuery, taskQuery, jobQuery, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// 2. Create Connection and Task via ConnectionQuery (this goes to database)
	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestCancelSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, nil)

	// 4. Reload task with Connection edge before running
	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 5. Create a context that is already cancelled
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// 6. Run the task with the cancelled context
	err = syncEngine.RunTask(cancelCtx, testTask, model.JobTriggerManual)
	assert.Error(t, err, "RunTask should return an error for a cancelled context")
	assert.Contains(t, err.Error(), "context canceled", "Error should mention context cancellation")

	// 7. Verify that no job was created because the context was cancelled before any work
	jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, jobs, "No job should be created if the context is already cancelled")
}

// TestSyncEngine_RunTask_CancelDuringSync tests that cancelling a context during an active sync operation
// properly handles the cancellation and marks the job as cancelled.
// This test covers the cancellation logic in sync.go lines 185-199.
func TestSyncEngine_RunTask_CancelDuringSync(t *testing.T) {
	connQuery, taskQuery, jobQuery, _ := setupIntegrationTest(t)
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
	slowConn, err := connQuery.CreateConnection(ctx, "slowlocal", "slowfs", map[string]string{
		"type":   "slowfs",
		"remote": "/",
	}, nil)
	require.NoError(t, err)

	// 4. Create task using slowfs - use upload direction to trigger Put on destination
	testTask, err := taskQuery.CreateTask(ctx,
		"TestCancelDuringSync",
		sourceDir,
		slowConn.ID,
		destDir,
		string(model.SyncDirectionUpload),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	// 5. Reload task with Connection edge
	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 6. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, nil)

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
	jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
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
	jobs, err = jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
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
			connQuery, taskQuery, jobQuery, _ := setupIntegrationTest(t)
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
			testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
			require.NoError(t, err)

			// Create options with noDelete setting
			options := &model.TaskSyncOptions{
				NoDelete: &tt.noDelete,
			}

			testTask, err := taskQuery.CreateTask(ctx,
				tt.name,
				sourceDir,
				testConn.ID,
				destDir,
				string(model.SyncDirectionUpload), // noDelete only applies to one-way sync
				"",
				false,
				true,
				options,
			)
			require.NoError(t, err)

			// 3. Setup SyncEngine
			dataDir := t.TempDir()
			syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, nil)

			// 4. Reload task with Connection edge
			testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
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
	connQuery, taskQuery, jobQuery, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// 2. Create Connection and Task
	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestProgressEventsSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		true,
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
	syncEngine := rclone.NewSyncEngine(jobQuery, jobProgressBus, transferProgressBus, dataDir, false, 0, nil)

	// 7. Reload task with Connection edge before running
	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
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

// TestSyncEngine_RunTask_MetacacheIntegration tests the integration of metacache with ChangeNotify.
// This test verifies that:
// 1. metacache backend is correctly used when cache is enabled for a connection
// 2. ChangeNotify from notifylocal triggers cache invalidation in metacache
// 3. Out-of-band changes to files are detected and synced correctly
func TestSyncEngine_RunTask_MetacacheIntegration(t *testing.T) {
	connQuery, taskQuery, jobQuery, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup directories
	sourceDir := t.TempDir()
	remoteDir := t.TempDir()

	// 2. Create a notifylocal connection with cache enabled
	cacheEnabled := true
	infoAge := "1h"
	testConn, err := connQuery.CreateConnection(ctx,
		"metaremote",
		"notifylocal",
		map[string]string{"type": "notifylocal"},
		&model.ConnectionOptions{
			Cache: &model.ConnectionCacheOptions{
				Enabled: cacheEnabled,
				InfoAge: &infoAge,
			},
		},
	)
	require.NoError(t, err)

	// 3. Create task with bidirectional sync
	testTask, err := taskQuery.CreateTask(ctx,
		"TestMetacacheSync",
		sourceDir,
		testConn.ID,
		remoteDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	// 4. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, nil)

	// 5. Initial sync: create file in source and sync to remote
	testFilePath := filepath.Join(sourceDir, "sync-test.txt")
	err = os.WriteFile(testFilePath, []byte("version 1"), 0644)
	require.NoError(t, err)

	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	// Verify initial sync succeeded
	remoteFilePath := filepath.Join(remoteDir, "sync-test.txt")
	content, err := os.ReadFile(remoteFilePath)
	require.NoError(t, err)
	assert.Equal(t, "version 1", string(content), "Remote file should have version 1 after initial sync")

	// 6. Simulate out-of-band change: directly modify remote file on disk
	// Wait a moment for any pending file system events
	time.Sleep(200 * time.Millisecond)

	// Modify the remote file directly (simulating external change)
	err = os.WriteFile(remoteFilePath, []byte("version 2"), 0644)
	require.NoError(t, err)

	// Give ChangeNotify time to propagate the change notification
	time.Sleep(500 * time.Millisecond)

	// 7. Run sync again - if metacache and ChangeNotify work correctly,
	// the cache should be invalidated and the new content should be detected
	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	// 8. Verify that source file was updated from remote
	// This proves: notifylocal detected change -> ChangeNotify fired -> metacache invalidated -> bisync saw new content
	sourceContent, err := os.ReadFile(testFilePath)
	require.NoError(t, err)
	assert.Equal(t, "version 2", string(sourceContent), "Source file should be updated from remote out-of-band change")

	// 9. Verify job records
	jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 2, "Should have two jobs (initial sync + resync after change)")

	// Both jobs should be successful
	for _, job := range jobs {
		assert.Equal(t, string(model.JobStatusSuccess), string(job.Status), "Job should be successful")
	}
}

// setupIntegrationTestWithHook initializes test environment including hook query and executor.
func setupIntegrationTestWithHook(t *testing.T) (*query.ConnectionQuery, *query.TaskQuery, *query.JobQuery, *query.HookQuery, *rclone.DBStorage) {
	t.Helper()

	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	t.Cleanup(func() { client.Close() })

	encryptor, err := crypto.NewEncryptor("")
	require.NoError(t, err)

	connQuery := query.NewConnectionQuery(client, encryptor)
	taskQuery := query.NewTaskQuery(client)
	jobQuery := query.NewJobQuery(client)
	hookQuery := query.NewHookQuery(client)

	storage := rclone.NewDBStorage(connQuery, t.TempDir())
	storage.Install()

	return connQuery, taskQuery, jobQuery, hookQuery, storage
}

// TestSyncEngine_RunTask_Hook_OnSuccess tests that ON_SUCCESS hooks are triggered on successful sync.
func TestSyncEngine_RunTask_Hook_OnSuccess(t *testing.T) {
	connQuery, taskQuery, jobQuery, hookQuery, _ := setupIntegrationTestWithHook(t)
	ctx := context.Background()

	sourceDir := t.TempDir()
	destDir := t.TempDir()

	err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("hello"), 0644)
	require.NoError(t, err)

	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestHookOnSuccess",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionUpload),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	var hookCalled atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hookCalled.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
		Event:   model.HookEventOnSuccess,
		Type:    model.HookTypeHTTP,
		Enabled: func() *bool { v := true; return &v }(),
		Config: &model.HookConfigInput{
			URL: &server.URL,
		},
	})
	require.NoError(t, err)

	dataDir := t.TempDir()
	enabled := true
	hookExecutor := hook.NewExecutor(hookQuery, jobQuery, &enabled, 30)
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, hookExecutor)

	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), hookCalled.Load(), "ON_SUCCESS hook should be called exactly once")

	jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, string(model.JobStatusSuccess), string(jobs[0].Status))

	jobWithLogs, err := jobQuery.GetJobWithLogs(ctx, jobs[0].ID)
	require.NoError(t, err)
	foundHookLog := false
	for _, log := range jobWithLogs.Edges.Logs {
		if log.What == model.LogActionHook {
			foundHookLog = true
			assert.Equal(t, model.LogLevelInfo, log.Level)
			break
		}
	}
	assert.True(t, foundHookLog, "Should have a HOOK log entry")
}

// TestSyncEngine_RunTask_Hook_OnFailure tests that ON_FAILURE hooks are triggered on sync failure.
func TestSyncEngine_RunTask_Hook_OnFailure(t *testing.T) {
	connQuery, taskQuery, jobQuery, hookQuery, _ := setupIntegrationTestWithHook(t)
	ctx := context.Background()

	sourceDir := filepath.Join(t.TempDir(), "non_existent")
	destDir := t.TempDir()

	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestHookOnFailure",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionUpload),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	var hookCalled atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hookCalled.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
		Event:   model.HookEventOnFailure,
		Type:    model.HookTypeHTTP,
		Enabled: func() *bool { v := true; return &v }(),
		Config: &model.HookConfigInput{
			URL: &server.URL,
		},
	})
	require.NoError(t, err)

	dataDir := t.TempDir()
	enabled := true
	hookExecutor := hook.NewExecutor(hookQuery, jobQuery, &enabled, 30)
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, hookExecutor)

	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	assert.Error(t, err, "RunTask should return error for non-existent source")

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), hookCalled.Load(), "ON_FAILURE hook should be called exactly once")

	jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, string(model.JobStatusFailed), string(jobs[0].Status))
}

// TestSyncEngine_RunTask_Hook_OnEnd tests that ON_END hooks are triggered regardless of outcome.
func TestSyncEngine_RunTask_Hook_OnEnd(t *testing.T) {
	tests := []struct {
		name           string
		sourceExists   bool
		expectedStatus model.JobStatus
	}{
		{"success triggers ON_END", true, model.JobStatusSuccess},
		{"failure triggers ON_END", false, model.JobStatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connQuery, taskQuery, jobQuery, hookQuery, _ := setupIntegrationTestWithHook(t)
			ctx := context.Background()

			var sourceDir string
			if tt.sourceExists {
				sourceDir = t.TempDir()
				err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("data"), 0644)
				require.NoError(t, err)
			} else {
				sourceDir = filepath.Join(t.TempDir(), "non_existent")
			}
			destDir := t.TempDir()

			testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
			require.NoError(t, err)

			testTask, err := taskQuery.CreateTask(ctx,
				tt.name,
				sourceDir,
				testConn.ID,
				destDir,
				string(model.SyncDirectionUpload),
				"",
				false,
				true,
				nil,
			)
			require.NoError(t, err)

			var hookCalled atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hookCalled.Add(1)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
				Event:   model.HookEventOnEnd,
				Type:    model.HookTypeHTTP,
				Enabled: func() *bool { v := true; return &v }(),
				Config: &model.HookConfigInput{
					URL: &server.URL,
				},
			})
			require.NoError(t, err)

			dataDir := t.TempDir()
			enabled := true
			hookExecutor := hook.NewExecutor(hookQuery, jobQuery, &enabled, 30)
			syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, hookExecutor)

			testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
			require.NoError(t, err)

			_ = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)

			time.Sleep(100 * time.Millisecond)
			assert.Equal(t, int32(1), hookCalled.Load(), "ON_END hook should be called exactly once")

			jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
			require.NoError(t, err)
			require.Len(t, jobs, 1)
			assert.Equal(t, string(tt.expectedStatus), string(jobs[0].Status))
		})
	}
}

// TestSyncEngine_RunTask_Hook_OnStart_Cancel tests ON_START hook with CANCEL behavior.
func TestSyncEngine_RunTask_Hook_OnStart_Cancel(t *testing.T) {
	connQuery, taskQuery, jobQuery, hookQuery, _ := setupIntegrationTestWithHook(t)
	ctx := context.Background()

	sourceDir := t.TempDir()
	destDir := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("data"), 0644)
	require.NoError(t, err)

	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestHookOnStartCancel",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionUpload),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
		Event:   model.HookEventOnStart,
		Type:    model.HookTypeHTTP,
		OnError: func() *model.HookOnError { v := model.HookOnErrorCancel; return &v }(),
		Enabled: func() *bool { v := true; return &v }(),
		Config: &model.HookConfigInput{
			URL: &server.URL,
		},
	})
	require.NoError(t, err)

	dataDir := t.TempDir()
	enabled := true
	hookExecutor := hook.NewExecutor(hookQuery, jobQuery, &enabled, 30)
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, hookExecutor)

	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	assert.Error(t, err, "RunTask should return error when ON_START hook cancels")

	var cancelErr *hook.CancelError
	assert.ErrorAs(t, err, &cancelErr, "Error should be CancelError")

	jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, string(model.JobStatusCancelled), string(jobs[0].Status))

	destFilePath := filepath.Join(destDir, "test.txt")
	_, err = os.Stat(destFilePath)
	assert.True(t, os.IsNotExist(err), "File should NOT be synced when ON_START cancels")
}

// TestSyncEngine_RunTask_Hook_OnStart_Fatal tests ON_START hook with FATAL behavior.
func TestSyncEngine_RunTask_Hook_OnStart_Fatal(t *testing.T) {
	connQuery, taskQuery, jobQuery, hookQuery, _ := setupIntegrationTestWithHook(t)
	ctx := context.Background()

	sourceDir := t.TempDir()
	destDir := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("data"), 0644)
	require.NoError(t, err)

	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestHookOnStartFatal",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionUpload),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	var onFailureCalled atomic.Int32
	var onEndCalled atomic.Int32
	failureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		onFailureCalled.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer failureServer.Close()

	endServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		onEndCalled.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer endServer.Close()

	fatalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer fatalServer.Close()

	_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
		Event:    model.HookEventOnStart,
		Type:     model.HookTypeHTTP,
		OnError:  func() *model.HookOnError { v := model.HookOnErrorFatal; return &v }(),
		Priority: func() *int { v := 0; return &v }(),
		Enabled:  func() *bool { v := true; return &v }(),
		Config:   &model.HookConfigInput{URL: &fatalServer.URL},
	})
	require.NoError(t, err)

	_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
		Event:   model.HookEventOnFailure,
		Type:    model.HookTypeHTTP,
		Enabled: func() *bool { v := true; return &v }(),
		Config:  &model.HookConfigInput{URL: &failureServer.URL},
	})
	require.NoError(t, err)

	_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
		Event:   model.HookEventOnEnd,
		Type:    model.HookTypeHTTP,
		Enabled: func() *bool { v := true; return &v }(),
		Config:  &model.HookConfigInput{URL: &endServer.URL},
	})
	require.NoError(t, err)

	dataDir := t.TempDir()
	enabled := true
	hookExecutor := hook.NewExecutor(hookQuery, jobQuery, &enabled, 30)
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, hookExecutor)

	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	assert.Error(t, err, "RunTask should return error when ON_START hook fails fatally")

	var fatalErr *hook.FatalError
	assert.ErrorAs(t, err, &fatalErr, "Error should be FatalError")

	time.Sleep(100 * time.Millisecond)

	jobs, err := jobQuery.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, string(model.JobStatusFailed), string(jobs[0].Status))

	assert.Equal(t, int32(1), onFailureCalled.Load(), "ON_FAILURE should be called for fatal ON_START")
	assert.Equal(t, int32(1), onEndCalled.Load(), "ON_END should be called for fatal ON_START")

	destFilePath := filepath.Join(destDir, "test.txt")
	_, err = os.Stat(destFilePath)
	assert.True(t, os.IsNotExist(err), "File should NOT be synced when ON_START fails fatally")
}

// TestSyncEngine_RunTask_Hook_ConnectionLevel tests that connection-level hooks are triggered.
func TestSyncEngine_RunTask_Hook_ConnectionLevel(t *testing.T) {
	connQuery, taskQuery, jobQuery, hookQuery, _ := setupIntegrationTestWithHook(t)
	ctx := context.Background()

	sourceDir := t.TempDir()
	destDir := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("data"), 0644)
	require.NoError(t, err)

	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestHookConnectionLevel",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionUpload),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	var hookCalled atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hookCalled.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err = hookQuery.CreateHook(ctx, nil, &testConn.ID, model.HookInput{
		Event:   model.HookEventOnSuccess,
		Type:    model.HookTypeHTTP,
		Enabled: func() *bool { v := true; return &v }(),
		Config:  &model.HookConfigInput{URL: &server.URL},
	})
	require.NoError(t, err)

	dataDir := t.TempDir()
	enabled := true
	hookExecutor := hook.NewExecutor(hookQuery, jobQuery, &enabled, 30)
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, hookExecutor)

	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), hookCalled.Load(), "Connection-level hook should be called")
}

// TestSyncEngine_RunTask_Hook_Disabled tests that hooks are not executed when globally disabled.
func TestSyncEngine_RunTask_Hook_Disabled(t *testing.T) {
	connQuery, taskQuery, jobQuery, hookQuery, _ := setupIntegrationTestWithHook(t)
	ctx := context.Background()

	sourceDir := t.TempDir()
	destDir := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("data"), 0644)
	require.NoError(t, err)

	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestHookDisabled",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionUpload),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	var hookCalled atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hookCalled.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
		Event:   model.HookEventOnSuccess,
		Type:    model.HookTypeHTTP,
		Enabled: func() *bool { v := true; return &v }(),
		Config:  &model.HookConfigInput{URL: &server.URL},
	})
	require.NoError(t, err)

	dataDir := t.TempDir()
	enabled := false
	hookExecutor := hook.NewExecutor(hookQuery, jobQuery, &enabled, 30)
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, hookExecutor)

	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(0), hookCalled.Load(), "Hook should NOT be called when globally disabled")
}

// TestSyncEngine_RunTask_Hook_MultipleHooks tests execution order of multiple hooks.
func TestSyncEngine_RunTask_Hook_MultipleHooks(t *testing.T) {
	connQuery, taskQuery, jobQuery, hookQuery, _ := setupIntegrationTestWithHook(t)
	ctx := context.Background()

	sourceDir := t.TempDir()
	destDir := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("data"), 0644)
	require.NoError(t, err)

	testConn, err := connQuery.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	testTask, err := taskQuery.CreateTask(ctx,
		"TestHookMultiple",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionUpload),
		"",
		false,
		true,
		nil,
	)
	require.NoError(t, err)

	var callOrder []int
	var mu sync.Mutex

	createServer := func(id int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			callOrder = append(callOrder, id)
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}))
	}

	server1 := createServer(1)
	defer server1.Close()
	server2 := createServer(2)
	defer server2.Close()
	server3 := createServer(3)
	defer server3.Close()

	_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
		Event:    model.HookEventOnSuccess,
		Type:     model.HookTypeHTTP,
		Priority: func() *int { v := 2; return &v }(),
		Enabled:  func() *bool { v := true; return &v }(),
		Config:   &model.HookConfigInput{URL: &server2.URL},
	})
	require.NoError(t, err)

	_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
		Event:    model.HookEventOnSuccess,
		Type:     model.HookTypeHTTP,
		Priority: func() *int { v := 0; return &v }(),
		Enabled:  func() *bool { v := true; return &v }(),
		Config:   &model.HookConfigInput{URL: &server1.URL},
	})
	require.NoError(t, err)

	_, err = hookQuery.CreateHook(ctx, &testTask.ID, nil, model.HookInput{
		Event:    model.HookEventOnSuccess,
		Type:     model.HookTypeHTTP,
		Priority: func() *int { v := 5; return &v }(),
		Enabled:  func() *bool { v := true; return &v }(),
		Config:   &model.HookConfigInput{URL: &server3.URL},
	})
	require.NoError(t, err)

	dataDir := t.TempDir()
	enabled := true
	hookExecutor := hook.NewExecutor(hookQuery, jobQuery, &enabled, 30)
	syncEngine := rclone.NewSyncEngine(jobQuery, nil, nil, dataDir, false, 0, hookExecutor)

	testTask, err = taskQuery.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	orderCopy := make([]int, len(callOrder))
	copy(orderCopy, callOrder)
	mu.Unlock()

	assert.Len(t, orderCopy, 3, "All three hooks should be called")
	assert.Equal(t, []int{1, 2, 3}, orderCopy, "Hooks should be called in priority order (0, 2, 5)")
}
