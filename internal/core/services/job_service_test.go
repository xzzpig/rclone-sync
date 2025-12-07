package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
	"github.com/xzzpig/rclone-sync/internal/core/ent/joblog"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"go.uber.org/zap"
)

func init() {
	// Initialize logger for tests
	logger.L = zap.NewNop()
}

func TestJobService(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	service := NewJobService(client)
	taskService := NewTaskService(client)
	ctx := context.Background()

	// Helper to create a task for jobs
	createTask := func(t *testing.T) uuid.UUID {
		task, err := taskService.CreateTask(ctx, "Job Test Task "+uuid.NewString(), "/l", "r", "/r", "bidirectional", "", false, nil)
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
			l, err := service.AddJobLog(ctx, j.ID, string(joblog.LevelInfo), "test message", "/path")
			assert.NoError(t, err)

			fetchedJob, err := l.QueryJob().Only(ctx)
			assert.NoError(t, err)
			assert.Equal(t, j.ID, fetchedJob.ID)

			assert.Equal(t, "test message", l.Message)
		})

		t.Run("AddJobLogsBatch", func(t *testing.T) {
			logs := []*ent.JobLog{
				{
					Level:   joblog.LevelInfo,
					Message: "batch message 1",
					Path:    "/path/1",
					Time:    time.Now(),
				},
				{
					Level:   joblog.LevelError,
					Message: "batch message 2",
					Path:    "/path/2",
					Time:    time.Now(),
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
			list, err := service.ListJobs(ctx, &taskID, 10, 0)
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
		assert.Equal(t, job.StatusFailed, updated.Status)
		assert.Contains(t, updated.Errors, "System crash")
	})

	t.Run("GetJobWithLogs", func(t *testing.T) {
		taskID := createTask(t)
		j, _ := service.CreateJob(ctx, taskID, "manual")

		// Add multiple logs with slight delays to ensure different timestamps
		_, err := service.AddJobLog(ctx, j.ID, string(joblog.LevelInfo), "log message 1", "")
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // Ensure different timestamps
		_, err = service.AddJobLog(ctx, j.ID, string(joblog.LevelError), "log message 2", "/path/to/file")
		require.NoError(t, err)
		time.Sleep(time.Millisecond)
		_, err = service.AddJobLog(ctx, j.ID, string(joblog.LevelWarning), "log message 3", "")
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
		assert.Equal(t, "log message 1", got.Edges.Logs[0].Message)
		assert.Equal(t, joblog.LevelInfo, got.Edges.Logs[0].Level)
		assert.Equal(t, "", got.Edges.Logs[0].Path)

		assert.Equal(t, "log message 2", got.Edges.Logs[1].Message)
		assert.Equal(t, joblog.LevelError, got.Edges.Logs[1].Level)
		assert.Equal(t, "/path/to/file", got.Edges.Logs[1].Path)

		assert.Equal(t, "log message 3", got.Edges.Logs[2].Message)
		assert.Equal(t, joblog.LevelWarning, got.Edges.Logs[2].Level)
		assert.Equal(t, "", got.Edges.Logs[2].Path)
	})

	t.Run("GetJobWithLogs_NotFound", func(t *testing.T) {
		_, err := service.GetJobWithLogs(ctx, uuid.New())
		assert.ErrorIs(t, err, errs.ErrNotFound)
	})
}
