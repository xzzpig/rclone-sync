package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/rclone/rclone/backend/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/runner"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"
	"github.com/xzzpig/rclone-sync/internal/rclone/testutil"
)

// testContext holds all the components needed for integration tests.
type testContext struct {
	client      *ent.Client
	jobService  *services.JobService
	taskService *services.TaskService
	connService *services.ConnectionService
	syncEngine  *rclone.SyncEngine
	runner      *runner.Runner
	dataDir     string
	cleanup     func()
}

type setupOption func(*setupOptions)

type setupOptions struct {
	useSlowFs bool
	dbName    string
}

func withSlowFs() setupOption {
	return func(o *setupOptions) {
		o.useSlowFs = true
		o.dbName = "ent_cancel"
	}
}

// setupIntegrationTest initializes all real components for integration testing.
func setupIntegrationTest(t *testing.T, opts ...setupOption) *testContext {
	t.Helper()

	options := &setupOptions{dbName: ""}
	for _, opt := range opts {
		opt(options)
	}

	// Initialize logger
	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug, nil)

	// Use in-memory sqlite for testing with db.InitDB
	dsn := db.InMemoryDSN()
	if options.dbName != "" {
		dsn = "file:" + options.dbName + "?mode=memory&cache=shared&_fk=1"
	}
	client, err := db.InitDB(db.InitDBOptions{
		DSN:           dsn,
		MigrationMode: db.MigrationModeAuto,
	})
	require.NoError(t, err)

	// Create services
	jobService := services.NewJobService(client)
	taskService := services.NewTaskService(client)
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connService := services.NewConnectionService(client, encryptor)

	// Setup data directory
	dataDir := t.TempDir()

	// Install DBStorage for rclone configuration
	storage := rclone.NewDBStorage(connService)
	storage.Install()

	// Create SyncEngine and Runner
	syncEngine := rclone.NewSyncEngine(jobService, nil, nil, dataDir, false, 0)
	r := runner.NewRunner(syncEngine)

	cleanup := func() {
		if options.useSlowFs {
			testutil.ClearSlowFsController()
		}
		r.Stop()
		client.Close()
	}

	return &testContext{
		client:      client,
		jobService:  jobService,
		taskService: taskService,
		connService: connService,
		syncEngine:  syncEngine,
		runner:      r,
		dataDir:     dataDir,
		cleanup:     cleanup,
	}
}

// createTestTask creates a task with the given source and destination directories.
func createTestTask(t *testing.T, tc *testContext, name, sourceDir, destDir string) *ent.Task {
	t.Helper()
	ctx := context.Background()

	// Create or get local connection
	conn, err := tc.connService.GetConnectionByName(ctx, "local")
	if err != nil {
		conn, err = tc.connService.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"})
		require.NoError(t, err)
	}

	// Create task
	task, err := tc.taskService.CreateTask(ctx,
		name,
		sourceDir,
		conn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// Reload task with Connection edge
	task, err = tc.taskService.GetTaskWithConnection(ctx, task.ID)
	require.NoError(t, err)

	return task
}

// createSlowTask creates a task that uses the slowfs backend
func createSlowTask(t *testing.T, tc *testContext, name, sourceDir, destDir string) *ent.Task {
	t.Helper()
	ctx := context.Background()

	// Create slowfs connection
	conn, err := tc.connService.GetConnectionByName(ctx, "slowlocal")
	if err != nil {
		conn, err = tc.connService.CreateConnection(ctx, "slowlocal", "slowfs", map[string]string{
			"type":   "slowfs",
			"remote": "/",
		})
		require.NoError(t, err)
	}

	// Create task - source uses local path, destination uses slowfs connection
	task, err := tc.taskService.CreateTask(ctx,
		name,
		sourceDir,
		conn.ID,
		destDir,
		string(model.SyncDirectionUpload), // Use upload direction to trigger Put on destination
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// Reload task with Connection edge
	task, err = tc.taskService.GetTaskWithConnection(ctx, task.ID)
	require.NoError(t, err)

	return task
}

// waitForTaskCompletion waits until the task is no longer running or timeout.
func waitForTaskCompletion(t *testing.T, r *runner.Runner, task *ent.Task, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !r.IsRunning(task.ID) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// TestRunner_Integration_BasicSyncFlow tests the complete flow of starting a task,
// syncing files, and verifying completion.
func TestRunner_Integration_BasicSyncFlow(t *testing.T) {
	tc := setupIntegrationTest(t)
	defer tc.cleanup()

	// Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// Create task
	task := createTestTask(t, tc, "BasicSyncTest", sourceDir, destDir)

	// Start task via Runner
	err = tc.runner.StartTask(task, model.JobTriggerManual)
	require.NoError(t, err)
	assert.True(t, tc.runner.IsRunning(task.ID), "Task should be running after StartTask")

	// Wait for task completion
	completed := waitForTaskCompletion(t, tc.runner, task, 10*time.Second)
	assert.True(t, completed, "Task should complete within timeout")
	assert.False(t, tc.runner.IsRunning(task.ID), "Task should not be running after completion")

	// Verify file was synced
	destFilePath := filepath.Join(destDir, "test.txt")
	content, err := os.ReadFile(destFilePath)
	require.NoError(t, err, "File should exist in destination")
	assert.Equal(t, "hello world", string(content))

	// Verify job record
	jobs, err := tc.jobService.ListJobs(context.Background(), &task.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1, "Should have one job")
	assert.Equal(t, string(model.JobStatusSuccess), string(jobs[0].Status), "Job should be successful")
}

// TestRunner_Integration_MultipleTasks tests running multiple independent tasks sequentially.
// Note: We run tasks sequentially because SQLite has limited concurrent write support.
func TestRunner_Integration_MultipleTasks(t *testing.T) {
	tc := setupIntegrationTest(t)
	defer tc.cleanup()

	// Setup first task
	sourceDir1 := t.TempDir()
	destDir1 := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir1, "file1.txt"), []byte("task1"), 0644)
	require.NoError(t, err)
	task1 := createTestTask(t, tc, "MultiTask1", sourceDir1, destDir1)

	// Setup second task
	sourceDir2 := t.TempDir()
	destDir2 := t.TempDir()
	err = os.WriteFile(filepath.Join(sourceDir2, "file2.txt"), []byte("task2"), 0644)
	require.NoError(t, err)
	task2 := createTestTask(t, tc, "MultiTask2", sourceDir2, destDir2)

	// Start first task and wait for completion
	err = tc.runner.StartTask(task1, model.JobTriggerManual)
	require.NoError(t, err)
	completed1 := waitForTaskCompletion(t, tc.runner, task1, 10*time.Second)
	assert.True(t, completed1, "Task1 should complete")

	// Start second task and wait for completion
	err = tc.runner.StartTask(task2, model.JobTriggerSchedule)
	require.NoError(t, err)
	completed2 := waitForTaskCompletion(t, tc.runner, task2, 10*time.Second)
	assert.True(t, completed2, "Task2 should complete")

	// Verify files were synced
	content1, err := os.ReadFile(filepath.Join(destDir1, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "task1", string(content1))

	content2, err := os.ReadFile(filepath.Join(destDir2, "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "task2", string(content2))

	// Verify job records
	jobs1, err := tc.jobService.ListJobs(context.Background(), &task1.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs1, 1)
	assert.Equal(t, string(model.JobStatusSuccess), string(jobs1[0].Status))

	jobs2, err := tc.jobService.ListJobs(context.Background(), &task2.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs2, 1)
	assert.Equal(t, string(model.JobStatusSuccess), string(jobs2[0].Status))
}

// TestRunner_Cancel_StopRunningTaskMarksCancelled tests that stopping a running task
// properly marks the job as cancelled.
func TestRunner_Cancel_StopRunningTaskMarksCancelled(t *testing.T) {
	tc := setupIntegrationTest(t, withSlowFs())
	defer tc.cleanup()

	// Setup control channels
	startedCh := make(chan struct{}, 10)
	blockCh := make(chan struct{})
	testutil.SetSlowFsController(startedCh, blockCh)

	// Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err)

	// Create task
	task := createSlowTask(t, tc, "StopCancelTest", sourceDir, destDir)

	// Start task
	t.Log("Starting task...")
	err = tc.runner.StartTask(task, model.JobTriggerManual)
	require.NoError(t, err)

	// Wait for the task to start
	t.Log("Waiting for task to start...")
	select {
	case <-startedCh:
		t.Log("Task started and is now blocking")
	case <-time.After(5 * time.Second):
		t.Fatal("Task did not start within timeout")
	}

	// Give it a moment
	time.Sleep(100 * time.Millisecond)

	// Check that the job is running
	jobs, err := tc.jobService.ListJobs(context.Background(), &task.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	jobID := jobs[0].ID
	assert.Equal(t, string(model.JobStatusRunning), string(jobs[0].Status))

	// Stop the task
	t.Log("Stopping task...")
	err = tc.runner.StopTask(task.ID)
	require.NoError(t, err)

	// Give it time to process
	time.Sleep(500 * time.Millisecond)

	// Check job status
	jobs, err = tc.jobService.ListJobs(context.Background(), &task.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, jobID, jobs[0].ID)
	t.Logf("Job status after stop: %s", jobs[0].Status)

	// The job should be cancelled
	assert.Equal(t, string(model.JobStatusCancelled), string(jobs[0].Status),
		"Job should be cancelled after StopTask. If this fails, the bug exists!")
}

// TestRunner_Integration_StopCancelsAllTasks tests that Runner.Stop() cancels all running tasks.
func TestRunner_Integration_StopCancelsAllTasks(t *testing.T) {
	tc := setupIntegrationTest(t)
	// Don't defer tc.cleanup() because we're testing Stop() explicitly

	// Setup two tasks
	sourceDir1 := t.TempDir()
	destDir1 := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir1, "file1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)
	task1 := createTestTask(t, tc, "StopAllTask1", sourceDir1, destDir1)

	sourceDir2 := t.TempDir()
	destDir2 := t.TempDir()
	err = os.WriteFile(filepath.Join(sourceDir2, "file2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)
	task2 := createTestTask(t, tc, "StopAllTask2", sourceDir2, destDir2)

	// Start both tasks
	err = tc.runner.StartTask(task1, model.JobTriggerManual)
	require.NoError(t, err)
	err = tc.runner.StartTask(task2, model.JobTriggerManual)
	require.NoError(t, err)

	// Give them a moment to start
	time.Sleep(50 * time.Millisecond)

	// Call Stop() - this should cancel all tasks and wait
	done := make(chan struct{})
	go func() {
		tc.runner.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success - Stop returned
	case <-time.After(5 * time.Second):
		t.Fatal("Runner.Stop() timed out")
	}

	// Verify both tasks are no longer running
	assert.False(t, tc.runner.IsRunning(task1.ID), "Task1 should not be running after Stop")
	assert.False(t, tc.runner.IsRunning(task2.ID), "Task2 should not be running after Stop")

	// Cleanup client
	tc.client.Close()
}

// TestRunner_Integration_TaskExecutionError tests handling of task execution failures.
func TestRunner_Integration_TaskExecutionError(t *testing.T) {
	tc := setupIntegrationTest(t)
	defer tc.cleanup()

	// Setup task with non-existent source directory
	sourceDir := filepath.Join(t.TempDir(), "non_existent")
	destDir := t.TempDir()

	// Create task
	task := createTestTask(t, tc, "ErrorTest", sourceDir, destDir)

	// Start task
	err := tc.runner.StartTask(task, model.JobTriggerManual)
	require.NoError(t, err)

	// Wait for completion (should fail quickly)
	completed := waitForTaskCompletion(t, tc.runner, task, 5*time.Second)
	assert.True(t, completed, "Task should complete (with failure)")
	assert.False(t, tc.runner.IsRunning(task.ID), "Task should not be running after failure")

	// Wait a bit for job status to be updated
	time.Sleep(200 * time.Millisecond)

	// Verify job has failed status
	jobs, err := tc.jobService.ListJobs(context.Background(), &task.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1, "Should have one job")
	assert.Equal(t, string(model.JobStatusFailed), string(jobs[0].Status), "Job should have failed status")
	assert.NotEmpty(t, jobs[0].Errors, "Job should have error message")
}

// TestRunner_Integration_ConcurrentStartStop tests thread safety of concurrent StartTask and StopTask calls.
func TestRunner_Integration_ConcurrentStartStop(t *testing.T) {
	tc := setupIntegrationTest(t)
	defer tc.cleanup()

	// Setup test task
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	task := createTestTask(t, tc, "ConcurrentTest", sourceDir, destDir)

	// Run concurrent operations
	var wg sync.WaitGroup
	numOps := 20

	// Start goroutines that call StartTask
	for i := 0; i < numOps/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = tc.runner.StartTask(task, model.JobTriggerManual)
		}()
	}

	// Start goroutines that call StopTask
	for i := 0; i < numOps/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = tc.runner.StopTask(task.ID)
		}()
	}

	// Wait for all operations to complete
	wg.Wait()

	// Wait for any in-flight tasks to complete
	time.Sleep(500 * time.Millisecond)

	// The test passes if no panic occurred
	// Verify we can still check status without issues
	_ = tc.runner.IsRunning(task.ID)

	// Verify jobs were created (count may vary due to race conditions)
	jobs, err := tc.jobService.ListJobs(context.Background(), &task.ID, nil, 100, 0)
	require.NoError(t, err)
	t.Logf("Created %d jobs during concurrent operations", len(jobs))
	assert.NotEmpty(t, jobs, "Should have created at least one job")
}

// TestRunner_Integration_DifferentTriggerTypes tests that different trigger types are correctly recorded.
func TestRunner_Integration_DifferentTriggerTypes(t *testing.T) {
	tc := setupIntegrationTest(t)
	defer tc.cleanup()

	triggers := []model.JobTrigger{
		model.JobTriggerManual,
		model.JobTriggerSchedule,
		model.JobTriggerRealtime,
	}

	for _, trigger := range triggers {
		t.Run(string(trigger), func(t *testing.T) {
			// Setup test directories
			sourceDir := t.TempDir()
			destDir := t.TempDir()
			err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("content"), 0644)
			require.NoError(t, err)

			// Create unique task for each trigger type
			task := createTestTask(t, tc, "TriggerTest_"+string(trigger), sourceDir, destDir)

			// Start task with specific trigger
			err = tc.runner.StartTask(task, trigger)
			require.NoError(t, err)

			// Wait for completion
			completed := waitForTaskCompletion(t, tc.runner, task, 10*time.Second)
			assert.True(t, completed, "Task should complete")

			// Verify trigger type is recorded correctly
			jobs, err := tc.jobService.ListJobs(context.Background(), &task.ID, nil, 10, 0)
			require.NoError(t, err)
			assert.Len(t, jobs, 1)
			assert.Equal(t, trigger, jobs[0].Trigger, "Trigger type should match")
		})
	}
}

// TestRunner_Integration_StopNonExistentTask tests stopping a task that doesn't exist or has already completed.
func TestRunner_Integration_StopNonExistentTask(t *testing.T) {
	tc := setupIntegrationTest(t)
	defer tc.cleanup()

	// Setup and run a task to completion
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	task := createTestTask(t, tc, "NonExistentStopTest", sourceDir, destDir)

	// Start and wait for completion
	err = tc.runner.StartTask(task, model.JobTriggerManual)
	require.NoError(t, err)

	completed := waitForTaskCompletion(t, tc.runner, task, 10*time.Second)
	assert.True(t, completed, "Task should complete")

	// Try to stop the completed task - should not panic
	err = tc.runner.StopTask(task.ID)
	assert.NoError(t, err, "StopTask on completed task should not return error")

	// Try to stop a completely random task ID - should not panic
	randomTask := &ent.Task{ID: task.ID}
	randomTask.ID = [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	err = tc.runner.StopTask(randomTask.ID)
	assert.NoError(t, err, "StopTask on non-existent task should not return error")
}

// TestRunner_Integration_TriggerBehaviorWhenRunning tests how different trigger types
// behave when a task is already running.
// - Realtime triggers should skip (not cancel the running task)
// - Manual triggers should cancel and restart
func TestRunner_Integration_TriggerBehaviorWhenRunning(t *testing.T) {
	tests := []struct {
		name            string
		secondTrigger   model.JobTrigger
		expectCancelled bool // whether the first job should be cancelled
		expectSecondJob bool // whether a second job should be created
	}{
		{
			name:            "Realtime_Skips",
			secondTrigger:   model.JobTriggerRealtime,
			expectCancelled: false,
			expectSecondJob: false,
		},
		{
			name:            "Manual_Cancels",
			secondTrigger:   model.JobTriggerManual,
			expectCancelled: true,
			expectSecondJob: true,
		},
		{
			name:            "Schedule_Cancels",
			secondTrigger:   model.JobTriggerSchedule,
			expectCancelled: true,
			expectSecondJob: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := setupIntegrationTest(t, withSlowFs())
			defer tc.cleanup()

			// Setup control channels
			startedCh := make(chan struct{}, 10)
			blockCh := make(chan struct{})
			testutil.SetSlowFsController(startedCh, blockCh)

			// Setup test directories
			sourceDir := t.TempDir()
			destDir := t.TempDir()
			err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("hello world"), 0644)
			require.NoError(t, err)

			// Create task
			task := createSlowTask(t, tc, "TriggerTest_"+tt.name, sourceDir, destDir)

			// Start first task with Manual trigger
			err = tc.runner.StartTask(task, model.JobTriggerManual)
			require.NoError(t, err)

			// Wait for the first task to start
			select {
			case <-startedCh:
			case <-time.After(5 * time.Second):
				t.Fatal("First task did not start within timeout")
			}
			time.Sleep(100 * time.Millisecond)

			// Check first job is running
			jobs, err := tc.jobService.ListJobs(context.Background(), &task.ID, nil, 10, 0)
			require.NoError(t, err)
			require.Len(t, jobs, 1)
			firstJobID := jobs[0].ID
			assert.Equal(t, string(model.JobStatusRunning), string(jobs[0].Status))

			// Trigger with second trigger type
			err = tc.runner.StartTask(task, tt.secondTrigger)
			require.NoError(t, err)
			time.Sleep(500 * time.Millisecond)

			// Check job statuses
			jobs, err = tc.jobService.ListJobs(context.Background(), &task.ID, nil, 10, 0)
			require.NoError(t, err)

			var firstJob *ent.Job
			for i := range jobs {
				if jobs[i].ID == firstJobID {
					firstJob = jobs[i]
					break
				}
			}
			require.NotNil(t, firstJob)

			if tt.expectCancelled {
				assert.Equal(t, string(model.JobStatusCancelled), string(firstJob.Status),
					"First job should be cancelled")
			} else {
				assert.Equal(t, string(model.JobStatusRunning), string(firstJob.Status),
					"First job should still be running")
			}

			if tt.expectSecondJob {
				assert.Greater(t, len(jobs), 1, "Should have created a second job")
			} else {
				assert.Len(t, jobs, 1, "Should not have created a second job")
			}

			// Unblock and complete
			close(blockCh)
			waitForTaskCompletion(t, tc.runner, task, 5*time.Second)

			// Final check - should have at least one successful job
			time.Sleep(200 * time.Millisecond)
			jobs, err = tc.jobService.ListJobs(context.Background(), &task.ID, nil, 10, 0)
			require.NoError(t, err)
			hasSuccess := false
			for _, j := range jobs {
				if j.Status == model.JobStatusSuccess {
					hasSuccess = true
					break
				}
			}
			assert.True(t, hasSuccess, "Should have at least one successful job")
		})
	}
}

// TestRunner_Integration_RapidStartStop tests rapid start/stop sequences on the same task.
func TestRunner_Integration_RapidStartStop(t *testing.T) {
	tc := setupIntegrationTest(t)
	defer tc.cleanup()

	// Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	task := createTestTask(t, tc, "RapidStartStopTest", sourceDir, destDir)

	// Rapidly start and stop the task multiple times
	for i := 0; i < 5; i++ {
		err = tc.runner.StartTask(task, model.JobTriggerManual)
		assert.NoError(t, err)
		err = tc.runner.StopTask(task.ID)
		assert.NoError(t, err)
	}

	// Final start and let it complete
	err = tc.runner.StartTask(task, model.JobTriggerManual)
	require.NoError(t, err)

	completed := waitForTaskCompletion(t, tc.runner, task, 10*time.Second)
	assert.True(t, completed, "Final task should complete")

	// Verify jobs were created
	jobs, err := tc.jobService.ListJobs(context.Background(), &task.ID, nil, 100, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, jobs, "Should have created jobs")
	t.Logf("Created %d jobs during rapid start/stop sequence", len(jobs))

	// At least the last one should be successful
	hasSuccess := false
	for _, j := range jobs {
		if j.Status == model.JobStatusSuccess {
			hasSuccess = true
			break
		}
	}
	assert.True(t, hasSuccess, "Should have at least one successful job")
}
