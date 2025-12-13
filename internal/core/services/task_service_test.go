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

	t.Run("ListTasksByRemote", func(t *testing.T) {
		// Create a task for testing
		testTask, err := service.CreateTask(ctx, "Task For Remote Test", "/local", "test-remote", "/remote", "bidirectional", "", false, nil)
		require.NoError(t, err)

		// Create a job service to create jobs
		jobService := NewJobService(client)

		// Create multiple jobs for the task with different start times
		job1, err := jobService.CreateJob(ctx, testTask.ID, "manual")
		require.NoError(t, err)

		// Update job1 to have a specific start time
		_, err = client.Job.UpdateOneID(job1.ID).
			SetStatus(job.StatusSuccess).
			Save(ctx)
		require.NoError(t, err)

		// Create a second job (which should have a later start time)
		job2, err := jobService.CreateJob(ctx, testTask.ID, "schedule")
		require.NoError(t, err)

		t.Run("ReturnsLatestJobForEachTask", func(t *testing.T) {
			tasks, err := service.ListTasksByRemote(ctx, "test-remote")
			assert.NoError(t, err)
			assert.NotEmpty(t, tasks)

			// Find our test task
			var foundTask *ent.Task
			for _, task := range tasks {
				if task.ID == testTask.ID {
					foundTask = task
					break
				}
			}
			require.NotNil(t, foundTask, "Test task should be found")

			// Verify it has exactly one job (the latest one)
			assert.Len(t, foundTask.Edges.Jobs, 1)

			// Verify it's the latest job (job2)
			assert.Equal(t, job2.ID, foundTask.Edges.Jobs[0].ID)
		})

		t.Run("FiltersByRemoteName", func(t *testing.T) {
			// Create another task with a different remote
			otherTask, err := service.CreateTask(ctx, "Task For Other Remote", "/local2", "other-remote", "/remote2", "bidirectional", "", false, nil)
			require.NoError(t, err)

			// Query for test-remote only
			tasks, err := service.ListTasksByRemote(ctx, "test-remote")
			assert.NoError(t, err)

			// Verify otherTask is not in the results
			for _, task := range tasks {
				assert.NotEqual(t, otherTask.ID, task.ID)
			}

			// Verify testTask is in the results
			foundTestTask := false
			for _, task := range tasks {
				if task.ID == testTask.ID {
					foundTestTask = true
					break
				}
			}
			assert.True(t, foundTestTask)
		})

		t.Run("ReturnsAllTasksWhenRemoteNameIsEmpty", func(t *testing.T) {
			tasks, err := service.ListTasksByRemote(ctx, "")
			assert.NoError(t, err)
			// Should return all tasks
			assert.NotEmpty(t, tasks)
		})
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

	t.Run("GetTaskWithJobs", func(t *testing.T) {
		// Setup: Task with multiple jobs
		taskWithJobs, err := service.CreateTask(ctx, "Task With Jobs", "/l", "r", "/r", "bidirectional", "", false, nil)
		require.NoError(t, err)

		jobService := NewJobService(client)
		// Old job
		j1, err := jobService.CreateJob(ctx, taskWithJobs.ID, "manual")
		require.NoError(t, err)
		// Manually update StartTime to be older
		_, err = client.Job.UpdateOne(j1).SetStartTime(j1.StartTime.Add(-2 * time.Hour)).Save(ctx)
		require.NoError(t, err)

		// Newer job
		time.Sleep(10 * time.Millisecond)
		j2, err := jobService.CreateJob(ctx, taskWithJobs.ID, "schedule")
		require.NoError(t, err)

		// Test
		gotTask, err := service.GetTaskWithJobs(ctx, taskWithJobs.ID)
		assert.NoError(t, err)
		require.NotNil(t, gotTask)

		require.Len(t, gotTask.Edges.Jobs, 1) // Should only have one job (the latest)
		assert.Equal(t, j2.ID, gotTask.Edges.Jobs[0].ID)

		t.Run("NotFound", func(t *testing.T) {
			_, err := service.GetTaskWithJobs(ctx, uuid.New())
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})
	})
}
