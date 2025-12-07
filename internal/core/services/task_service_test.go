package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
	"github.com/xzzpig/rclone-sync/internal/core/ent/task"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
)

func TestTaskService(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	service := NewTaskService(client)
	ctx := context.Background()

	// Helper to cleanup tasks between tests if needed, though we rely on unique names/ids mostly
	t.Cleanup(func() {
		client.Task.Delete().Exec(context.Background())
	})

	t.Run("CreateTask", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			task, err := service.CreateTask(ctx, "Test Task", "/local/path", "remote", "/remote/path", "bidirectional", "", false, nil)
			require.NoError(t, err)
			assert.NotNil(t, task)
			assert.Equal(t, "Test Task", task.Name)
			assert.Equal(t, "bidirectional", string(task.Direction))
		})
	})

	t.Run("ListTasks", func(t *testing.T) {
		tasks, err := service.ListAllTasks(ctx)
		assert.NoError(t, err)
		// Should have "Test Task" from previous run
		assert.NotEmpty(t, tasks)
		assert.Equal(t, "Test Task", tasks[0].Name)
	})

	t.Run("GetTask", func(t *testing.T) {
		tasks, _ := service.ListAllTasks(ctx)
		existingTask := tasks[0]

		t.Run("Success", func(t *testing.T) {
			task, err := service.GetTask(ctx, existingTask.ID)
			assert.NoError(t, err)
			assert.Equal(t, existingTask.ID, task.ID)
		})

		t.Run("NotFound", func(t *testing.T) {
			_, err := service.GetTask(ctx, uuid.New())
			assert.Error(t, err)
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})
	})

	t.Run("UpdateTask", func(t *testing.T) {
		tasks, _ := service.ListAllTasks(ctx)
		existingTask := tasks[0]

		t.Run("Success", func(t *testing.T) {
			updated, err := service.UpdateTask(ctx, existingTask.ID, "Updated Task", existingTask.SourcePath, existingTask.RemoteName, existingTask.RemotePath, string(existingTask.Direction), existingTask.Schedule, existingTask.Realtime, existingTask.Options)
			assert.NoError(t, err)
			assert.Equal(t, "Updated Task", updated.Name)
		})

		t.Run("NotFound", func(t *testing.T) {
			_, err := service.UpdateTask(ctx, uuid.New(), "New Name", "s", "r", "rp", "bidirectional", "", false, nil)
			assert.Error(t, err)
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})
	})

	t.Run("DeleteTask", func(t *testing.T) {
		// Create a task to delete to avoid interfering with other tests sequences if any
		tToDelete, err := service.CreateTask(ctx, "To Delete", "/l", "r", "/r", "bidirectional", "", false, nil)
		require.NoError(t, err)

		t.Run("Success", func(t *testing.T) {
			err := service.DeleteTask(ctx, tToDelete.ID)
			assert.NoError(t, err)

			// Verify it's gone
			_, err = service.GetTask(ctx, tToDelete.ID)
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})

		t.Run("NotFound", func(t *testing.T) {
			err := service.DeleteTask(ctx, uuid.New())
			assert.Error(t, err)
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})

		t.Run("WithAssociatedJobs", func(t *testing.T) {
			// Create a new task for this test
			taskWithJobs, err := service.CreateTask(ctx, "Task With Jobs", "/path", "remote", "/remote", "bidirectional", "", false, nil)
			require.NoError(t, err)

			// Create a job service to create jobs
			jobService := NewJobService(client)

			// Create multiple jobs associated with the task
			job1, err := jobService.CreateJob(ctx, taskWithJobs.ID, "manual")
			require.NoError(t, err)
			job2, err := jobService.CreateJob(ctx, taskWithJobs.ID, "schedule")
			require.NoError(t, err)

			// Verify jobs exist before deletion
			jobsBeforeDelete, err := client.Job.Query().
				Where(job.HasTaskWith(task.ID(taskWithJobs.ID))).
				All(ctx)
			require.NoError(t, err)
			assert.Len(t, jobsBeforeDelete, 2)

			// Delete the task - this should also delete all associated jobs due to cascade delete
			err = service.DeleteTask(ctx, taskWithJobs.ID)
			assert.NoError(t, err)

			// Verify task is gone
			_, err = service.GetTask(ctx, taskWithJobs.ID)
			assert.ErrorIs(t, err, errs.ErrNotFound)

			// Verify jobs are also deleted due to cascade delete
			jobsAfterDelete, err := client.Job.Query().
				Where(job.HasTaskWith(task.ID(taskWithJobs.ID))).
				All(ctx)
			require.NoError(t, err)
			assert.Len(t, jobsAfterDelete, 0)

			// Verify individual jobs are also deleted
			_, err = jobService.GetJob(ctx, job1.ID)
			assert.ErrorIs(t, err, errs.ErrNotFound)
			_, err = jobService.GetJob(ctx, job2.ID)
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})
	})
}
