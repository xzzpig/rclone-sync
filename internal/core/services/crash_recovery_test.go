package services_test

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/services"
)

func TestCrashRecovery_ResetStuckJobs(t *testing.T) {
	// Initialize logger for test
	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)

	// 1. Setup in-memory database
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	// Ensure DB schema is created
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed creating schema resources: %v", err)
	}

	ctx := context.Background()
	jobSvc := services.NewJobService(client)
	taskSvc := services.NewTaskService(client)

	// Create a test connection
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connSvc := services.NewConnectionService(client, encryptor)
	testConn, err := connSvc.CreateConnection(ctx, "test-local", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	// 2. Setup Prerequisites: Create a Task
	task, err := taskSvc.CreateTask(ctx,
		"Crash Test Task",
		"/tmp/source",
		testConn.ID,
		"/remote/dest",
		string(model.SyncDirectionUpload),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Simulate "Zombie" Jobs (Stuck in Running state from a previous crash)

	// Job 1: Running (Should be reset)
	runningJob, err := client.Job.Create().
		SetTask(task).
		SetTrigger(model.JobTriggerManual).
		SetStatus(model.JobStatusRunning). // FORCE STATUS RUNNING
		SetStartTime(time.Now().Add(-1 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// Create some job logs for the running job to test statistics calculation
	// Upload log 1: 100 bytes
	_, err = client.JobLog.Create().
		SetJob(runningJob).
		SetLevel(model.LogLevelInfo).
		SetWhat(model.LogActionUpload).
		SetPath("/test/file1.txt").
		SetSize(100).
		SetTime(time.Now().Add(-50 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	// Upload log 2: 200 bytes
	_, err = client.JobLog.Create().
		SetJob(runningJob).
		SetLevel(model.LogLevelInfo).
		SetWhat(model.LogActionUpload).
		SetPath("/test/file2.txt").
		SetSize(200).
		SetTime(time.Now().Add(-40 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	// Download log: 300 bytes
	_, err = client.JobLog.Create().
		SetJob(runningJob).
		SetLevel(model.LogLevelInfo).
		SetWhat(model.LogActionDownload).
		SetPath("/test/file3.txt").
		SetSize(300).
		SetTime(time.Now().Add(-30 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	// Move log: 50 bytes (should be counted)
	_, err = client.JobLog.Create().
		SetJob(runningJob).
		SetLevel(model.LogLevelInfo).
		SetWhat(model.LogActionMove).
		SetPath("/test/file4.txt").
		SetSize(50).
		SetTime(time.Now().Add(-20 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	// Error log: 1000 bytes (should NOT be counted - wrong level)
	_, err = client.JobLog.Create().
		SetJob(runningJob).
		SetLevel(model.LogLevelError).
		SetWhat(model.LogActionUpload).
		SetPath("/test/error_file.txt").
		SetSize(1000).
		SetTime(time.Now().Add(-10 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	// Delete log: 500 bytes (should NOT be counted - delete action not counted as transfer)
	_, err = client.JobLog.Create().
		SetJob(runningJob).
		SetLevel(model.LogLevelInfo).
		SetWhat(model.LogActionDelete).
		SetPath("/test/deleted_file.txt").
		SetSize(500).
		SetTime(time.Now().Add(-5 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	// Job 2: Success (Should be untouched)
	successJob, err := client.Job.Create().
		SetTask(task).
		SetTrigger(model.JobTriggerSchedule).
		SetStatus(model.JobStatusSuccess).
		SetStartTime(time.Now().Add(-2 * time.Hour)).
		SetEndTime(time.Now().Add(-1 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// Job 3: Pending (Should be untouched - theoretically pending jobs might need handling too, but current logic only targets running)
	pendingJob, err := client.Job.Create().
		SetTask(task).
		SetTrigger(model.JobTriggerManual).
		SetStatus(model.JobStatusPending).
		SetStartTime(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 4. Execute Recovery Logic
	// This is the method called in serve.go on startup
	err = jobSvc.ResetStuckJobs(ctx)
	assert.NoError(t, err)

	// 5. Verify Results

	// Check Job 1 (Was Running -> Now Failed)
	updatedRunningJob, err := client.Job.Get(ctx, runningJob.ID)
	require.NoError(t, err)
	assert.Equal(t, model.JobStatusCancelled, updatedRunningJob.Status, "Stuck running job should be marked as Cancelled")
	assert.Contains(t, updatedRunningJob.Errors, "System crash", "Error message should indicate crash recovery")
	assert.NotNil(t, updatedRunningJob.EndTime, "EndTime should satisfy constraint")

	// Verify transfer statistics are correctly calculated
	// Expected: 4 files (2 uploads + 1 download + 1 move), total 650 bytes (100 + 200 + 300 + 50)
	// NOT counted: Error level log (1000 bytes), Delete action log (500 bytes)
	assert.Equal(t, 4, updatedRunningJob.FilesTransferred, "Should count 4 files (uploads, downloads, moves with INFO level)")
	assert.Equal(t, int64(650), updatedRunningJob.BytesTransferred, "Should sum bytes from counted files (100+200+300+50=650)")

	// Check Job 2 (Was Success -> Remains Success)
	updatedSuccessJob, err := client.Job.Get(ctx, successJob.ID)
	require.NoError(t, err)
	assert.Equal(t, model.JobStatusSuccess, updatedSuccessJob.Status, "Success job should not be changed")

	// Check Job 3 (Was Pending -> Remains Pending)
	updatedPendingJob, err := client.Job.Get(ctx, pendingJob.ID)
	require.NoError(t, err)
	assert.Equal(t, model.JobStatusPending, updatedPendingJob.Status, "Pending job should not be changed")
}
