package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
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
		task, err := taskService.CreateTask(ctx, "Job Test Task "+uuid.NewString(), "/l", testConnID, "/r", string(model.SyncDirectionBidirectional), "", false, nil)
		require.NoError(t, err)
		return task.ID
	}

	t.Run("CreateJob", func(t *testing.T) {
		taskID := createTask(t)

		t.Run("Success", func(t *testing.T) {
			j, err := service.CreateJob(ctx, taskID, model.JobTriggerManual)
			assert.NoError(t, err)
			assert.NotNil(t, j)

			fetchedTask, err := j.QueryTask().Only(ctx)
			assert.NoError(t, err)
			assert.Equal(t, taskID, fetchedTask.ID)

			assert.Equal(t, model.JobStatusPending, j.Status)
			assert.Equal(t, model.JobTriggerManual, j.Trigger)
		})

		t.Run("InvalidTask", func(t *testing.T) {
			_, err := service.CreateJob(ctx, uuid.New(), model.JobTriggerManual)
			assert.Error(t, err)
			// SQLite FK constraint error usually wraps as system error in our service layer currently
			// or specifically ErrSystem if not handled explicitly as ConstraintError in CreateJob
			assert.ErrorIs(t, err, errs.ErrSystem)
		})
	})

	t.Run("UpdateJobStatus", func(t *testing.T) {
		taskID := createTask(t)
		j, _ := service.CreateJob(ctx, taskID, model.JobTriggerManual)

		t.Run("Success_Running", func(t *testing.T) {
			updated, err := service.UpdateJobStatus(ctx, j.ID, string(model.JobStatusRunning), "")
			assert.NoError(t, err)
			assert.Equal(t, model.JobStatusRunning, updated.Status)
			assert.True(t, updated.EndTime.IsZero())
		})

		t.Run("Success_Terminal", func(t *testing.T) {
			updated, err := service.UpdateJobStatus(ctx, j.ID, string(model.JobStatusSuccess), "")
			assert.NoError(t, err)
			assert.Equal(t, model.JobStatusSuccess, updated.Status)
			assert.NotNil(t, updated.EndTime)
		})

		t.Run("NotFound", func(t *testing.T) {
			_, err := service.UpdateJobStatus(ctx, uuid.New(), string(model.JobStatusRunning), "")
			assert.Error(t, err)
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})
	})

	t.Run("UpdateJobStats", func(t *testing.T) {
		taskID := createTask(t)
		j, _ := service.CreateJob(ctx, taskID, model.JobTriggerManual)

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
		j, _ := service.CreateJob(ctx, taskID, model.JobTriggerManual)

		t.Run("AddJobLog", func(t *testing.T) {
			l, err := service.AddJobLog(ctx, j.ID, string(model.LogLevelInfo), string(model.LogActionMove), "/path", 1024)
			assert.NoError(t, err)

			fetchedJob, err := l.QueryJob().Only(ctx)
			assert.NoError(t, err)
			assert.Equal(t, j.ID, fetchedJob.ID)

			assert.Equal(t, model.LogActionMove, l.What)
			assert.Equal(t, int64(1024), l.Size)
		})

		t.Run("AddJobLogsBatch", func(t *testing.T) {
			logs := []*ent.JobLog{
				{
					Level: model.LogLevelInfo,
					What:  model.LogActionMove,
					Path:  "/path/1",
					Size:  2048,
					Time:  time.Now(),
				},
				{
					Level: model.LogLevelError,
					What:  model.LogActionError,
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
		j1, _ := service.CreateJob(ctx, taskID, model.JobTriggerManual)
		time.Sleep(10 * time.Millisecond) // Ensure time difference
		j2, _ := service.CreateJob(ctx, taskID, model.JobTriggerSchedule)

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
		j, _ := service.CreateJob(ctx, taskID, model.JobTriggerManual)
		_, _ = service.UpdateJobStatus(ctx, j.ID, string(model.JobStatusRunning), "")

		err := service.ResetStuckJobs(ctx)
		assert.NoError(t, err)

		// Verify it is now failed
		updated, _ := service.GetJob(ctx, j.ID)
		assert.Equal(t, model.JobStatusCancelled, updated.Status)
		assert.Contains(t, updated.Errors, "System crash")
	})

	t.Run("GetJobWithLogs", func(t *testing.T) {
		taskID := createTask(t)
		j, _ := service.CreateJob(ctx, taskID, model.JobTriggerManual)

		// Add multiple logs with slight delays to ensure different timestamps
		_, err := service.AddJobLog(ctx, j.ID, string(model.LogLevelInfo), string(model.LogActionMove), "", 512)
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // Ensure different timestamps
		_, err = service.AddJobLog(ctx, j.ID, string(model.LogLevelError), string(model.LogActionError), "/path/to/file", 0)
		require.NoError(t, err)
		time.Sleep(time.Millisecond)
		_, err = service.AddJobLog(ctx, j.ID, string(model.LogLevelWarning), string(model.LogActionDelete), "", 0)
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
		assert.Equal(t, model.LogActionMove, got.Edges.Logs[0].What)
		assert.Equal(t, model.LogLevelInfo, got.Edges.Logs[0].Level)
		assert.Equal(t, "", got.Edges.Logs[0].Path)
		assert.Equal(t, int64(512), got.Edges.Logs[0].Size)

		assert.Equal(t, model.LogActionError, got.Edges.Logs[1].What)
		assert.Equal(t, model.LogLevelError, got.Edges.Logs[1].Level)
		assert.Equal(t, "/path/to/file", got.Edges.Logs[1].Path)
		assert.Equal(t, int64(0), got.Edges.Logs[1].Size)

		assert.Equal(t, model.LogActionDelete, got.Edges.Logs[2].What)
		assert.Equal(t, model.LogLevelWarning, got.Edges.Logs[2].Level)
		assert.Equal(t, "", got.Edges.Logs[2].Path)
		assert.Equal(t, int64(0), got.Edges.Logs[2].Size)
	})

	t.Run("GetJobWithLogs_NotFound", func(t *testing.T) {
		_, err := service.GetJobWithLogs(ctx, uuid.New())
		assert.ErrorIs(t, err, errs.ErrNotFound)
	})

	t.Run("CountJobs", func(t *testing.T) {
		newTaskID := createTask(t)
		_, err := service.CreateJob(ctx, newTaskID, model.JobTriggerManual)
		require.NoError(t, err)
		_, err = service.CreateJob(ctx, newTaskID, model.JobTriggerSchedule)
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
			uniqueTask, err := taskService.CreateTask(ctx, "Unique Connection Task", "/l", uniqueConn.ID, "/r", string(model.SyncDirectionBidirectional), "", false, nil)
			require.NoError(t, err)
			_, err = service.CreateJob(ctx, uniqueTask.ID, model.JobTriggerManual)
			require.NoError(t, err)

			count, err := service.CountJobs(ctx, nil, &uniqueConn.ID)
			assert.NoError(t, err)
			assert.Equal(t, 1, count)
		})
	})

	t.Run("JobLogs_ListAndCount", func(t *testing.T) {
		// Setup: One task, One job, Multiple logs
		taskID := createTask(t)
		job, err := service.CreateJob(ctx, taskID, model.JobTriggerManual)
		require.NoError(t, err)
		jobID := job.ID

		_, err = service.AddJobLog(ctx, jobID, string(model.LogLevelInfo), string(model.LogActionUpload), "", 1024)
		require.NoError(t, err)
		_, err = service.AddJobLog(ctx, jobID, string(model.LogLevelError), string(model.LogLevelError), "", 0)
		require.NoError(t, err)
		_, err = service.AddJobLog(ctx, jobID, string(model.LogLevelInfo), string(model.LogActionDownload), "", 2048)
		require.NoError(t, err)

		t.Run("ListJobLogs_FilterByLevel", func(t *testing.T) {
			logs, err := service.ListJobLogs(ctx, nil, nil, &jobID, string(model.LogLevelInfo), 10, 0)
			assert.NoError(t, err)
			assert.Len(t, logs, 2)
			for _, l := range logs {
				assert.Equal(t, model.LogLevelInfo, l.Level)
			}
		})

		t.Run("CountJobLogs_FilterByLevel", func(t *testing.T) {
			count, err := service.CountJobLogs(ctx, nil, nil, &jobID, string(model.LogLevelError))
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
		j, err := service.CreateJob(ctx, taskID, model.JobTriggerManual)
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
		uniqueTask, err := taskService.CreateTask(ctx, "List Jobs Task "+uuid.NewString(), "/l", uniqueConn.ID, "/r", string(model.SyncDirectionBidirectional), "", false, nil)
		require.NoError(t, err)

		// Create jobs for this task
		job1, err := service.CreateJob(ctx, uniqueTask.ID, model.JobTriggerManual)
		require.NoError(t, err)
		job2, err := service.CreateJob(ctx, uniqueTask.ID, model.JobTriggerSchedule)
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
			j, err := service.CreateJob(ctx, taskID, model.JobTriggerManual)
			require.NoError(t, err)

			updated, err := service.UpdateJobStatus(ctx, j.ID, string(model.JobStatusFailed), "Test error message")
			assert.NoError(t, err)
			assert.Equal(t, model.JobStatusFailed, updated.Status)
			assert.NotNil(t, updated.EndTime)
			assert.Equal(t, "Test error message", updated.Errors)
		})

		// Test Cancelled status
		t.Run("Cancelled", func(t *testing.T) {
			j, err := service.CreateJob(ctx, taskID, model.JobTriggerManual)
			require.NoError(t, err)

			updated, err := service.UpdateJobStatus(ctx, j.ID, string(model.JobStatusCancelled), "User cancelled")
			assert.NoError(t, err)
			assert.Equal(t, model.JobStatusCancelled, updated.Status)
			assert.NotNil(t, updated.EndTime)
			assert.Equal(t, "User cancelled", updated.Errors)
		})

		// Test Success with error string (edge case)
		t.Run("Success_WithErrorString", func(t *testing.T) {
			j, err := service.CreateJob(ctx, taskID, model.JobTriggerManual)
			require.NoError(t, err)

			updated, err := service.UpdateJobStatus(ctx, j.ID, string(model.JobStatusSuccess), "Warning: some files skipped")
			assert.NoError(t, err)
			assert.Equal(t, model.JobStatusSuccess, updated.Status)
			assert.NotNil(t, updated.EndTime)
			assert.Equal(t, "Warning: some files skipped", updated.Errors)
		})

		// Test Running -> Failed transition
		t.Run("Running_ToFailed", func(t *testing.T) {
			j, err := service.CreateJob(ctx, taskID, model.JobTriggerManual)
			require.NoError(t, err)

			// First set to running
			_, err = service.UpdateJobStatus(ctx, j.ID, string(model.JobStatusRunning), "")
			require.NoError(t, err)

			// Then fail it
			updated, err := service.UpdateJobStatus(ctx, j.ID, string(model.JobStatusFailed), "Connection lost")
			assert.NoError(t, err)
			assert.Equal(t, model.JobStatusFailed, updated.Status)
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
	task, err := taskService.CreateTask(ctx, "Transition Test Task", "/l", testConn.ID, "/r", string(model.SyncDirectionBidirectional), "", false, nil)
	require.NoError(t, err)

	// Create job
	j, err := service.CreateJob(ctx, task.ID, model.JobTriggerManual)
	require.NoError(t, err)
	assert.Equal(t, model.JobStatusPending, j.Status)

	// Pending -> Running
	updated, err := service.UpdateJobStatus(ctx, j.ID, string(model.JobStatusRunning), "")
	require.NoError(t, err)
	assert.Equal(t, model.JobStatusRunning, updated.Status)
	assert.True(t, updated.EndTime.IsZero())
	assert.Empty(t, updated.Errors)

	// Running -> Success
	updated, err = service.UpdateJobStatus(ctx, j.ID, string(model.JobStatusSuccess), "")
	require.NoError(t, err)
	assert.Equal(t, model.JobStatusSuccess, updated.Status)
	assert.False(t, updated.EndTime.IsZero())
	assert.Empty(t, updated.Errors)
}

// Test for ListJobLogsByJobPaginated
func TestJobService_ListJobLogsByJobPaginated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	service := NewJobService(client)
	taskService := NewTaskService(client)
	ctx := context.Background()

	// Create test connection
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connService := NewConnectionService(client, encryptor)
	testConn, err := connService.CreateConnection(ctx, "test-paginated-logs", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	// Create task
	task, err := taskService.CreateTask(ctx, "Paginated Logs Task", "/l", testConn.ID, "/r", string(model.SyncDirectionBidirectional), "", false, nil)
	require.NoError(t, err)

	// Create job
	j, err := service.CreateJob(ctx, task.ID, model.JobTriggerManual)
	require.NoError(t, err)

	// Add multiple logs
	for i := 0; i < 5; i++ {
		_, err := service.AddJobLog(ctx, j.ID, string(model.LogLevelInfo), string(model.LogActionUpload), "/file"+string(rune('0'+i)), int64(i*100))
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	t.Run("FirstPage", func(t *testing.T) {
		logs, total, err := service.ListJobLogsByJobPaginated(ctx, j.ID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, logs, 2)
		assert.Equal(t, 5, total)
	})

	t.Run("SecondPage", func(t *testing.T) {
		logs, total, err := service.ListJobLogsByJobPaginated(ctx, j.ID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, logs, 2)
		assert.Equal(t, 5, total)
	})

	t.Run("LastPage", func(t *testing.T) {
		logs, total, err := service.ListJobLogsByJobPaginated(ctx, j.ID, 2, 4)
		require.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, 5, total)
	})

	t.Run("OffsetBeyondTotal", func(t *testing.T) {
		logs, total, err := service.ListJobLogsByJobPaginated(ctx, j.ID, 10, 100)
		require.NoError(t, err)
		assert.Empty(t, logs)
		assert.Equal(t, 5, total)
	})

	t.Run("LargeLimit", func(t *testing.T) {
		logs, total, err := service.ListJobLogsByJobPaginated(ctx, j.ID, 100, 0)
		require.NoError(t, err)
		assert.Len(t, logs, 5)
		assert.Equal(t, 5, total)
	})

	t.Run("EmptyJob", func(t *testing.T) {
		// Create a job with no logs
		emptyJob, err := service.CreateJob(ctx, task.ID, model.JobTriggerManual)
		require.NoError(t, err)

		logs, total, err := service.ListJobLogsByJobPaginated(ctx, emptyJob.ID, 10, 0)
		require.NoError(t, err)
		assert.Empty(t, logs)
		assert.Equal(t, 0, total)
	})

	t.Run("OrderedByTimeAsc", func(t *testing.T) {
		logs, _, err := service.ListJobLogsByJobPaginated(ctx, j.ID, 5, 0)
		require.NoError(t, err)
		require.Len(t, logs, 5)

		// Verify logs are ordered by time ascending
		for i := 1; i < len(logs); i++ {
			assert.True(t, logs[i].Time.After(logs[i-1].Time) || logs[i].Time.Equal(logs[i-1].Time),
				"Logs should be ordered by time ascending")
		}
	})
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
	task, err := taskService.CreateTask(ctx, "Count Logs Task", "/l", testConn.ID, "/r", string(model.SyncDirectionBidirectional), "", false, nil)
	require.NoError(t, err)

	// Create job
	j, err := service.CreateJob(ctx, task.ID, model.JobTriggerManual)
	require.NoError(t, err)

	// Add various logs
	_, err = service.AddJobLog(ctx, j.ID, string(model.LogLevelInfo), string(model.LogActionUpload), "/file1", 100)
	require.NoError(t, err)
	_, err = service.AddJobLog(ctx, j.ID, string(model.LogLevelInfo), string(model.LogActionDownload), "/file2", 200)
	require.NoError(t, err)
	_, err = service.AddJobLog(ctx, j.ID, string(model.LogLevelError), string(model.LogLevelError), "/file3", 0)
	require.NoError(t, err)
	_, err = service.AddJobLog(ctx, j.ID, string(model.LogLevelWarning), string(model.LogActionDelete), "/file4", 0)
	require.NoError(t, err)

	t.Run("CountByConnectionAndLevel", func(t *testing.T) {
		count, err := service.CountJobLogs(ctx, &testConn.ID, nil, nil, string(model.LogLevelInfo))
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("CountByTaskAndLevel", func(t *testing.T) {
		count, err := service.CountJobLogs(ctx, nil, &task.ID, nil, string(model.LogLevelError))
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("CountByJobOnly", func(t *testing.T) {
		count, err := service.CountJobLogs(ctx, nil, nil, &j.ID, "")
		assert.NoError(t, err)
		assert.Equal(t, 4, count)
	})
}
