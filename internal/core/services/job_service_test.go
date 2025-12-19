package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
	"github.com/xzzpig/rclone-sync/internal/core/ent/joblog"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
)

func init() {
	// Initialize logger for tests
	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)
}

func TestJobService(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	service := NewJobService(client)
	taskService := NewTaskService(client)
	ctx := context.Background()

	// Create a test connection for use across tests
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connService := NewConnectionService(client, encryptor)
	testConn, err := connService.CreateConnection(ctx, "test-local", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)
	testConnID := testConn.ID

	// Helper to create a task for jobs
	createTask := func(t *testing.T) uuid.UUID {
		task, err := taskService.CreateTask(ctx, "Job Test Task "+uuid.NewString(), "/l", testConnID, "/r", "bidirectional", "", false, nil)
		require.NoError(t, err)
		return task.ID
	}

	t.Run("CreateJob", func(t *testing.T) {
		taskID := createTask(t)

		t.Run("Success", func(t *testing.T) {
			j, err := service.CreateJob(ctx, taskID, "manual")
			assert.NoError(t, err)
			assert.NotNil(t, j)

			fetchedTask, err := j.QueryTask().Only(ctx)
			assert.NoError(t, err)
			assert.Equal(t, taskID, fetchedTask.ID)

			assert.Equal(t, job.StatusPending, j.Status)
			assert.Equal(t, job.TriggerManual, j.Trigger)
		})

		t.Run("InvalidTask", func(t *testing.T) {
			_, err := service.CreateJob(ctx, uuid.New(), "manual")
			assert.Error(t, err)
			// SQLite FK constraint error usually wraps as system error in our service layer currently
			// or specifically ErrSystem if not handled explicitly as ConstraintError in CreateJob
			assert.ErrorIs(t, err, errs.ErrSystem)
		})
	})

	t.Run("UpdateJobStatus", func(t *testing.T) {
		taskID := createTask(t)
		j, _ := service.CreateJob(ctx, taskID, "manual")

		t.Run("Success_Running", func(t *testing.T) {
			updated, err := service.UpdateJobStatus(ctx, j.ID, string(job.StatusRunning), "")
			assert.NoError(t, err)
			assert.Equal(t, job.StatusRunning, updated.Status)
			assert.True(t, updated.EndTime.IsZero())
		})

		t.Run("Success_Terminal", func(t *testing.T) {
			updated, err := service.UpdateJobStatus(ctx, j.ID, string(job.StatusSuccess), "")
			assert.NoError(t, err)
			assert.Equal(t, job.StatusSuccess, updated.Status)
			assert.NotNil(t, updated.EndTime)
		})

		t.Run("NotFound", func(t *testing.T) {
			_, err := service.UpdateJobStatus(ctx, uuid.New(), string(job.StatusRunning), "")
			assert.Error(t, err)
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})
	})

	t.Run("UpdateJobStats", func(t *testing.T) {
		taskID := createTask(t)
		j, _ := service.CreateJob(ctx, taskID, "manual")

		updated, err := service.UpdateJobStats(ctx, j.ID, 10, 1024)
		assert.NoError(t, err)
		assert.Equal(t, 10, updated.FilesTransferred)
		assert.Equal(t, int64(1024), updated.BytesTransferred)

		t.Run("NotFound", func(t *testing.T) {
			_, err := service.UpdateJobStats(ctx, uuid.New(), 10, 1024)
			assert.Error(t, err)
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})
	})

	t.Run("Logging", func(t *testing.T) {
		taskID := createTask(t)
		j, _ := service.CreateJob(ctx, taskID, "manual")

		t.Run("AddJobLog", func(t *testing.T) {
			l, err := service.AddJobLog(ctx, j.ID, string(joblog.LevelInfo), string(joblog.WhatUpload), "/path", 1024)
			assert.NoError(t, err)

			fetchedJob, err := l.QueryJob().Only(ctx)
			assert.NoError(t, err)
			assert.Equal(t, j.ID, fetchedJob.ID)

			assert.Equal(t, joblog.WhatUpload, l.What)
			assert.Equal(t, int64(1024), l.Size)
		})

		t.Run("AddJobLogsBatch", func(t *testing.T) {
			logs := []*ent.JobLog{
				{
					Level: joblog.LevelInfo,
					What:  joblog.WhatDownload,
					Path:  "/path/1",
					Size:  2048,
					Time:  time.Now(),
				},
				{
					Level: joblog.LevelError,
					What:  joblog.WhatError,
					Path:  "/path/2",
					Time:  time.Now(),
				},
			}
			err := service.AddJobLogsBatch(ctx, j.ID, logs)
			assert.NoError(t, err)

			// Verify logs were added
			savedLogs, err := j.QueryLogs().All(ctx)
			assert.NoError(t, err)
			// We added 1 log in previous test, and 2 here. Total 3.
			assert.Len(t, savedLogs, 3)
		})
	})

	t.Run("Retrieval", func(t *testing.T) {
		taskID := createTask(t)
		j1, _ := service.CreateJob(ctx, taskID, "manual")
		time.Sleep(10 * time.Millisecond) // Ensure time difference
		j2, _ := service.CreateJob(ctx, taskID, "schedule")

		t.Run("GetJob", func(t *testing.T) {
			got, err := service.GetJob(ctx, j1.ID)
			assert.NoError(t, err)
			assert.Equal(t, j1.ID, got.ID)
		})

		t.Run("GetJob_NotFound", func(t *testing.T) {
			_, err := service.GetJob(ctx, uuid.New())
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})

		t.Run("GetLastJobByTaskID", func(t *testing.T) {
			last, err := service.GetLastJobByTaskID(ctx, taskID)
			assert.NoError(t, err)
			assert.Equal(t, j2.ID, last.ID)
		})

		t.Run("ListJobs", func(t *testing.T) {
			list, err := service.ListJobs(ctx, &taskID, nil, 10, 0)
			assert.NoError(t, err)
			assert.Len(t, list, 2)
			// Should be ordered by StartTime Desc
			assert.Equal(t, j2.ID, list[0].ID)
			assert.Equal(t, j1.ID, list[1].ID)
		})
	})

	t.Run("ResetStuckJobs", func(t *testing.T) {
		taskID := createTask(t)
		// Create a stuck job
		j, _ := service.CreateJob(ctx, taskID, "manual")
		_, _ = service.UpdateJobStatus(ctx, j.ID, string(job.StatusRunning), "")

		err := service.ResetStuckJobs(ctx)
		assert.NoError(t, err)

		// Verify it is now failed
		updated, _ := service.GetJob(ctx, j.ID)
		assert.Equal(t, job.StatusCancelled, updated.Status)
		assert.Contains(t, updated.Errors, "System crash")
	})

	t.Run("GetJobWithLogs", func(t *testing.T) {
		taskID := createTask(t)
		j, _ := service.CreateJob(ctx, taskID, "manual")

		// Add multiple logs with slight delays to ensure different timestamps
		_, err := service.AddJobLog(ctx, j.ID, string(joblog.LevelInfo), string(joblog.WhatUpload), "", 512)
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // Ensure different timestamps
		_, err = service.AddJobLog(ctx, j.ID, string(joblog.LevelError), string(joblog.WhatError), "/path/to/file", 0)
		require.NoError(t, err)
		time.Sleep(time.Millisecond)
		_, err = service.AddJobLog(ctx, j.ID, string(joblog.LevelWarning), string(joblog.WhatDelete), "", 0)
		require.NoError(t, err)

		// Call the method under test
		got, err := service.GetJobWithLogs(ctx, j.ID)
		assert.NoError(t, err)

		// Verify basic Job info
		assert.Equal(t, j.ID, got.ID)

		// Verify associated Task is loaded
		assert.NotNil(t, got.Edges.Task)
		assert.Equal(t, taskID, got.Edges.Task.ID)

		// Verify logs are loaded and ordered by time ascending
		require.Len(t, got.Edges.Logs, 3)
		assert.Equal(t, joblog.WhatUpload, got.Edges.Logs[0].What)
		assert.Equal(t, joblog.LevelInfo, got.Edges.Logs[0].Level)
		assert.Equal(t, "", got.Edges.Logs[0].Path)
		assert.Equal(t, int64(512), got.Edges.Logs[0].Size)

		assert.Equal(t, joblog.WhatError, got.Edges.Logs[1].What)
		assert.Equal(t, joblog.LevelError, got.Edges.Logs[1].Level)
		assert.Equal(t, "/path/to/file", got.Edges.Logs[1].Path)
		assert.Equal(t, int64(0), got.Edges.Logs[1].Size)

		assert.Equal(t, joblog.WhatDelete, got.Edges.Logs[2].What)
		assert.Equal(t, joblog.LevelWarning, got.Edges.Logs[2].Level)
		assert.Equal(t, "", got.Edges.Logs[2].Path)
		assert.Equal(t, int64(0), got.Edges.Logs[2].Size)
	})

	t.Run("GetJobWithLogs_NotFound", func(t *testing.T) {
		_, err := service.GetJobWithLogs(ctx, uuid.New())
		assert.ErrorIs(t, err, errs.ErrNotFound)
	})

	t.Run("CountJobs", func(t *testing.T) {
		newTaskID := createTask(t)
		_, err := service.CreateJob(ctx, newTaskID, "manual")
		require.NoError(t, err)
		_, err = service.CreateJob(ctx, newTaskID, "schedule")
		require.NoError(t, err)

		t.Run("NoFilters", func(t *testing.T) {
			// There might be jobs from other tests, so we just check count > 0 or specific logic
			count, err := service.CountJobs(ctx, nil, nil)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, count, 2)
		})

		t.Run("FilterByTaskID", func(t *testing.T) {
			count, err := service.CountJobs(ctx, &newTaskID, nil)
			assert.NoError(t, err)
			assert.Equal(t, 2, count)
		})

		t.Run("FilterByConnectionID", func(t *testing.T) {
			// Create a unique connection for this test
			uniqueConn, err := connService.CreateConnection(ctx, "unique-conn-"+uuid.NewString(), "local", map[string]string{
				"type": "local",
			})
			require.NoError(t, err)
			uniqueTask, err := taskService.CreateTask(ctx, "Unique Connection Task", "/l", uniqueConn.ID, "/r", "bidirectional", "", false, nil)
			require.NoError(t, err)
			_, err = service.CreateJob(ctx, uniqueTask.ID, "manual")
			require.NoError(t, err)

			count, err := service.CountJobs(ctx, nil, &uniqueConn.ID)
			assert.NoError(t, err)
			assert.Equal(t, 1, count)
		})
	})

	t.Run("JobLogs_ListAndCount", func(t *testing.T) {
		// Setup: One task, One job, Multiple logs
		taskID := createTask(t)
		job, err := service.CreateJob(ctx, taskID, "manual")
		require.NoError(t, err)
		jobID := job.ID

		_, err = service.AddJobLog(ctx, jobID, "info", "upload", "", 1024)
		require.NoError(t, err)
		_, err = service.AddJobLog(ctx, jobID, "error", "error", "", 0)
		require.NoError(t, err)
		_, err = service.AddJobLog(ctx, jobID, "info", "download", "", 2048)
		require.NoError(t, err)

		t.Run("ListJobLogs_FilterByLevel", func(t *testing.T) {
			logs, err := service.ListJobLogs(ctx, nil, nil, &jobID, "info", 10, 0)
			assert.NoError(t, err)
			assert.Len(t, logs, 2)
			for _, l := range logs {
				assert.Equal(t, joblog.LevelInfo, l.Level)
			}
		})

		t.Run("CountJobLogs_FilterByLevel", func(t *testing.T) {
			count, err := service.CountJobLogs(ctx, nil, nil, &jobID, "error")
			assert.NoError(t, err)
			assert.Equal(t, 1, count)
		})

		t.Run("ListJobLogs_Pagination", func(t *testing.T) {
			// Determine order (default desc time)
			logs, err := service.ListJobLogs(ctx, nil, nil, &jobID, "", 1, 0) // First page, size 1
			assert.NoError(t, err)
			require.Len(t, logs, 1)

			logs2, err := service.ListJobLogs(ctx, nil, nil, &jobID, "", 1, 1) // Second page, size 1
			assert.NoError(t, err)
			require.Len(t, logs2, 1)

			assert.NotEqual(t, logs[0].ID, logs2[0].ID)
		})
	})

	t.Run("AddJobLogsBatch_Empty", func(t *testing.T) {
		taskID := createTask(t)
		j, err := service.CreateJob(ctx, taskID, "manual")
		require.NoError(t, err)

		err = service.AddJobLogsBatch(ctx, j.ID, []*ent.JobLog{})
		assert.NoError(t, err) // Should simply return nil
	})

	t.Run("ResetStuckJobs_NoOp", func(t *testing.T) {
		// Ensure no running jobs exist (or clean state)
		// We can't guarantee global state easily but we can check it doesn't error
		err := service.ResetStuckJobs(ctx)
		assert.NoError(t, err)
	})

	t.Run("GetLastJobByTaskID_NoJobs", func(t *testing.T) {
		emptyTask := createTask(t)
		_, err := service.GetLastJobByTaskID(ctx, emptyTask)
		assert.ErrorIs(t, err, errs.ErrNotFound)
	})

	t.Run("ListJobs_FilterByConnectionID", func(t *testing.T) {
		// Create a unique connection for this test
		uniqueConn, err := connService.CreateConnection(ctx, "list-jobs-conn-"+uuid.NewString(), "local", map[string]string{
			"type": "local",
		})
		require.NoError(t, err)

		// Create a task for this connection
		uniqueTask, err := taskService.CreateTask(ctx, "List Jobs Task "+uuid.NewString(), "/l", uniqueConn.ID, "/r", "bidirectional", "", false, nil)
		require.NoError(t, err)

		// Create jobs for this task
		job1, err := service.CreateJob(ctx, uniqueTask.ID, "manual")
		require.NoError(t, err)
		job2, err := service.CreateJob(ctx, uniqueTask.ID, "schedule")
		require.NoError(t, err)

		// List jobs by connectionID
		jobs, err := service.ListJobs(ctx, nil, &uniqueConn.ID, 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(jobs), 2)

		// Verify both jobs are in the results
		foundJob1 := false
		foundJob2 := false
		for _, j := range jobs {
			if j.ID == job1.ID {
				foundJob1 = true
			}
			if j.ID == job2.ID {
				foundJob2 = true
			}
		}
		assert.True(t, foundJob1)
		assert.True(t, foundJob2)
	})

	t.Run("UpdateJobStatus_AllStatuses", func(t *testing.T) {
		taskID := createTask(t)

		// Test Failed status
		t.Run("Failed", func(t *testing.T) {
			j, err := service.CreateJob(ctx, taskID, "manual")
			require.NoError(t, err)

			updated, err := service.UpdateJobStatus(ctx, j.ID, string(job.StatusFailed), "Test error message")
			assert.NoError(t, err)
			assert.Equal(t, job.StatusFailed, updated.Status)
			assert.NotNil(t, updated.EndTime)
			assert.Equal(t, "Test error message", updated.Errors)
		})

		// Test Cancelled status
		t.Run("Cancelled", func(t *testing.T) {
			j, err := service.CreateJob(ctx, taskID, "manual")
			require.NoError(t, err)

			updated, err := service.UpdateJobStatus(ctx, j.ID, string(job.StatusCancelled), "User cancelled")
			assert.NoError(t, err)
			assert.Equal(t, job.StatusCancelled, updated.Status)
			assert.NotNil(t, updated.EndTime)
			assert.Equal(t, "User cancelled", updated.Errors)
		})

		// Test Success with error string (edge case)
		t.Run("Success_WithErrorString", func(t *testing.T) {
			j, err := service.CreateJob(ctx, taskID, "manual")
			require.NoError(t, err)

			updated, err := service.UpdateJobStatus(ctx, j.ID, string(job.StatusSuccess), "Warning: some files skipped")
			assert.NoError(t, err)
			assert.Equal(t, job.StatusSuccess, updated.Status)
			assert.NotNil(t, updated.EndTime)
			assert.Equal(t, "Warning: some files skipped", updated.Errors)
		})

		// Test Running -> Failed transition
		t.Run("Running_ToFailed", func(t *testing.T) {
			j, err := service.CreateJob(ctx, taskID, "manual")
			require.NoError(t, err)

			// First set to running
			_, err = service.UpdateJobStatus(ctx, j.ID, string(job.StatusRunning), "")
			require.NoError(t, err)

			// Then fail it
			updated, err := service.UpdateJobStatus(ctx, j.ID, string(job.StatusFailed), "Connection lost")
			assert.NoError(t, err)
			assert.Equal(t, job.StatusFailed, updated.Status)
			assert.NotNil(t, updated.EndTime)
			assert.Equal(t, "Connection lost", updated.Errors)
		})
	})
}

// Additional test for UpdateJobStatus with multiple status transitions
func TestJobService_UpdateJobStatus_Transitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	service := NewJobService(client)
	taskService := NewTaskService(client)
	ctx := context.Background()

	// Create test connection
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connService := NewConnectionService(client, encryptor)
	testConn, err := connService.CreateConnection(ctx, "test-transitions", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	// Create task
	task, err := taskService.CreateTask(ctx, "Transition Test Task", "/l", testConn.ID, "/r", "bidirectional", "", false, nil)
	require.NoError(t, err)

	// Create job
	j, err := service.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)
	assert.Equal(t, job.StatusPending, j.Status)

	// Pending -> Running
	updated, err := service.UpdateJobStatus(ctx, j.ID, string(job.StatusRunning), "")
	require.NoError(t, err)
	assert.Equal(t, job.StatusRunning, updated.Status)
	assert.True(t, updated.EndTime.IsZero())
	assert.Empty(t, updated.Errors)

	// Running -> Success
	updated, err = service.UpdateJobStatus(ctx, j.ID, string(job.StatusSuccess), "")
	require.NoError(t, err)
	assert.Equal(t, job.StatusSuccess, updated.Status)
	assert.False(t, updated.EndTime.IsZero())
	assert.Empty(t, updated.Errors)
}

// Additional test for CountJobLogs with multiple filters
func TestJobService_CountJobLogs_ComplexFilters(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	service := NewJobService(client)
	taskService := NewTaskService(client)
	ctx := context.Background()

	// Create test connection
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connService := NewConnectionService(client, encryptor)
	testConn, err := connService.CreateConnection(ctx, "test-count-logs", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	// Create task
	task, err := taskService.CreateTask(ctx, "Count Logs Task", "/l", testConn.ID, "/r", "bidirectional", "", false, nil)
	require.NoError(t, err)

	// Create job
	j, err := service.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	// Add various logs
	_, err = service.AddJobLog(ctx, j.ID, "info", "upload", "/file1", 100)
	require.NoError(t, err)
	_, err = service.AddJobLog(ctx, j.ID, "info", "download", "/file2", 200)
	require.NoError(t, err)
	_, err = service.AddJobLog(ctx, j.ID, "error", "error", "/file3", 0)
	require.NoError(t, err)
	_, err = service.AddJobLog(ctx, j.ID, "warning", "delete", "/file4", 0)
	require.NoError(t, err)

	t.Run("CountByConnectionAndLevel", func(t *testing.T) {
		count, err := service.CountJobLogs(ctx, &testConn.ID, nil, nil, "info")
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("CountByTaskAndLevel", func(t *testing.T) {
		count, err := service.CountJobLogs(ctx, nil, &task.ID, nil, "error")
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("CountByJobOnly", func(t *testing.T) {
		count, err := service.CountJobLogs(ctx, nil, nil, &j.ID, "")
		assert.NoError(t, err)
		assert.Equal(t, 4, count)
	})
}
