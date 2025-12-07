package services_test

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
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

	// 2. Setup Prerequisites: Create a Task
	task, err := taskSvc.CreateTask(ctx,
		"Crash Test Task",
		"/tmp/source",
		"my-remote",
		"/remote/dest",
		"upload",
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Simulate "Zombie" Jobs (Stuck in Running state from a previous crash)

	// Job 1: Running (Should be reset)
	runningJob, err := client.Job.Create().
		SetTask(task).
		SetTrigger("manual").
		SetStatus(job.StatusRunning). // FORCE STATUS RUNNING
		SetStartTime(time.Now().Add(-1 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// Job 2: Success (Should be untouched)
	successJob, err := client.Job.Create().
		SetTask(task).
		SetTrigger("schedule").
		SetStatus(job.StatusSuccess).
		SetStartTime(time.Now().Add(-2 * time.Hour)).
		SetEndTime(time.Now().Add(-1 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// Job 3: Pending (Should be untouched - theoretically pending jobs might need handling too, but current logic only targets running)
	pendingJob, err := client.Job.Create().
		SetTask(task).
		SetTrigger("manual").
		SetStatus(job.StatusPending).
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
	assert.Equal(t, job.StatusFailed, updatedRunningJob.Status, "Stuck running job should be marked as FAILED")
	assert.Contains(t, updatedRunningJob.Errors, "System crash", "Error message should indicate crash recovery")
	assert.NotNil(t, updatedRunningJob.EndTime, "EndTime should satisfy constraint")

	// Check Job 2 (Was Success -> Remains Success)
	updatedSuccessJob, err := client.Job.Get(ctx, successJob.ID)
	require.NoError(t, err)
	assert.Equal(t, job.StatusSuccess, updatedSuccessJob.Status, "Success job should not be changed")

	// Check Job 3 (Was Pending -> Remains Pending)
	updatedPendingJob, err := client.Job.Get(ctx, pendingJob.ID)
	require.NoError(t, err)
	assert.Equal(t, job.StatusPending, updatedPendingJob.Status, "Pending job should not be changed")
}
