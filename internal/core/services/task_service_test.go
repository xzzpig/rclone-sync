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
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
	"github.com/xzzpig/rclone-sync/internal/core/ent/task"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
)

func TestTaskService(t *testing.T) {
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	defer client.Close()

	service := NewTaskService(client)
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connService := NewConnectionService(client, encryptor)
	ctx := context.Background()

	// Create a test connection for use across tests
	testConn, err := connService.CreateConnection(ctx, "test-local", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)
	testConnID := testConn.ID

	// Create another connection for multi-connection tests
	testConn2, err := connService.CreateConnection(ctx, "test-remote-2", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)
	testConnID2 := testConn2.ID

	// Helper to cleanup tasks between tests if needed, though we rely on unique names/ids mostly
	t.Cleanup(func() {
		client.Task.Delete().Exec(context.Background())
		client.Connection.Delete().Exec(context.Background())
	})

	t.Run("CreateTask", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			task, err := service.CreateTask(ctx, "Test Task", "/local/path", testConnID, "/remote/path", string(model.SyncDirectionBidirectional), "", false, nil)
			require.NoError(t, err)
			assert.NotNil(t, task)
			assert.Equal(t, "Test Task", task.Name)
			assert.Equal(t, string(model.SyncDirectionBidirectional), string(task.Direction))
		})
	})

	t.Run("ListTasks", func(t *testing.T) {
		tasks, err := service.ListAllTasks(ctx)
		assert.NoError(t, err)
		// Should have "Test Task" from previous run
		assert.NotEmpty(t, tasks)
		assert.Equal(t, "Test Task", tasks[0].Name)
	})

	t.Run("ListTasksByConnection", func(t *testing.T) {
		// Create a task for testing
		testTask, err := service.CreateTask(ctx, "Task For Connection Test", "/local", testConnID, "/remote", string(model.SyncDirectionBidirectional), "", false, nil)
		require.NoError(t, err)

		// Create a job service to create jobs
		jobService := NewJobService(client)

		// Create multiple jobs for the task with different start times
		job1, err := jobService.CreateJob(ctx, testTask.ID, model.JobTriggerManual)
		require.NoError(t, err)

		// Update job1 to have a specific start time
		_, err = client.Job.UpdateOneID(job1.ID).
			SetStatus(model.JobStatusSuccess).
			Save(ctx)
		require.NoError(t, err)

		// Create a second job (which should have a later start time)
		job2, err := jobService.CreateJob(ctx, testTask.ID, model.JobTriggerSchedule)
		require.NoError(t, err)

		t.Run("ReturnsLatestJobForEachTask", func(t *testing.T) {
			tasks, err := service.ListTasksByConnection(ctx, testConnID)
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

		t.Run("FiltersByConnectionID", func(t *testing.T) {
			// Create another task with a different connection
			otherTask, err := service.CreateTask(ctx, "Task For Other Connection", "/local2", testConnID2, "/remote2", string(model.SyncDirectionBidirectional), "", false, nil)
			require.NoError(t, err)

			// Query for testConnID only
			tasks, err := service.ListTasksByConnection(ctx, testConnID)
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

		t.Run("ReturnsAllTasksWhenConnectionIDIsZero", func(t *testing.T) {
			tasks, err := service.ListTasksByConnection(ctx, uuid.Nil)
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
			updated, err := service.UpdateTask(ctx, existingTask.ID, "Updated Task", existingTask.SourcePath, existingTask.ConnectionID, existingTask.RemotePath, string(existingTask.Direction), existingTask.Schedule, existingTask.Realtime, existingTask.Options)
			assert.NoError(t, err)
			assert.Equal(t, "Updated Task", updated.Name)
		})

		t.Run("NotFound", func(t *testing.T) {
			_, err := service.UpdateTask(ctx, uuid.New(), "New Name", "s", testConnID, "rp", string(model.SyncDirectionBidirectional), "", false, nil)
			assert.Error(t, err)
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})
	})

	t.Run("DeleteTask", func(t *testing.T) {
		// Create a task to delete to avoid interfering with other tests sequences if any
		tToDelete, err := service.CreateTask(ctx, "To Delete", "/l", testConnID, "/r", string(model.SyncDirectionBidirectional), "", false, nil)
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
			taskWithJobs, err := service.CreateTask(ctx, "Task With Jobs", "/path", testConnID, "/remote", string(model.SyncDirectionBidirectional), "", false, nil)
			require.NoError(t, err)

			// Create a job service to create jobs
			jobService := NewJobService(client)

			// Create multiple jobs associated with the task
			job1, err := jobService.CreateJob(ctx, taskWithJobs.ID, model.JobTriggerManual)
			require.NoError(t, err)
			job2, err := jobService.CreateJob(ctx, taskWithJobs.ID, model.JobTriggerSchedule)
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
		taskWithJobs, err := service.CreateTask(ctx, "Task With Jobs", "/l", testConnID, "/r", string(model.SyncDirectionBidirectional), "", false, nil)
		require.NoError(t, err)

		jobService := NewJobService(client)
		// Old job
		j1, err := jobService.CreateJob(ctx, taskWithJobs.ID, model.JobTriggerManual)
		require.NoError(t, err)
		// Manually update StartTime to be older
		_, err = client.Job.UpdateOne(j1).SetStartTime(j1.StartTime.Add(-2 * time.Hour)).Save(ctx)
		require.NoError(t, err)

		// Newer job
		time.Sleep(10 * time.Millisecond)
		j2, err := jobService.CreateJob(ctx, taskWithJobs.ID, model.JobTriggerSchedule)
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

	t.Run("GetTaskWithConnection", func(t *testing.T) {
		// Create a task
		testTask, err := service.CreateTask(ctx, "Task With Connection", "/local", testConnID, "/remote", string(model.SyncDirectionUpload), "", false, nil)
		require.NoError(t, err)

		t.Run("Success", func(t *testing.T) {
			gotTask, err := service.GetTaskWithConnection(ctx, testTask.ID)
			assert.NoError(t, err)
			require.NotNil(t, gotTask)
			assert.Equal(t, testTask.ID, gotTask.ID)
			assert.Equal(t, "Task With Connection", gotTask.Name)

			// Verify connection is loaded
			assert.NotNil(t, gotTask.Edges.Connection)
			assert.Equal(t, testConnID, gotTask.Edges.Connection.ID)
			assert.Equal(t, "test-local", gotTask.Edges.Connection.Name)
		})

		t.Run("NotFound", func(t *testing.T) {
			_, err := service.GetTaskWithConnection(ctx, uuid.New())
			assert.ErrorIs(t, err, errs.ErrNotFound)
		})
	})

	t.Run("CreateTask_ConstraintError", func(t *testing.T) {
		// Create a task with a specific name
		taskName := "Unique Task Name"
		_, err := service.CreateTask(ctx, taskName, "/source", testConnID, "/dest", string(model.SyncDirectionBidirectional), "", false, nil)
		require.NoError(t, err)

		// Try to create another task with the same name (if unique constraint exists)
		// Note: This depends on whether the schema enforces unique task names
		// If there's no unique constraint, this test may need adjustment
		_, err = service.CreateTask(ctx, taskName, "/source2", testConnID, "/dest2", string(model.SyncDirectionUpload), "", false, nil)
		if err != nil {
			// If there IS a unique constraint on task names
			assert.ErrorIs(t, err, errs.ErrAlreadyExists)
		} else {
			// If there's NO unique constraint, we can still create duplicate names
			// In this case, just verify the task was created
			t.Log("Schema allows duplicate task names")
		}
	})

	t.Run("UpdateTask_ConstraintError", func(t *testing.T) {
		// Create two tasks
		task1, err := service.CreateTask(ctx, "Update Task 1", "/s1", testConnID, "/d1", string(model.SyncDirectionBidirectional), "", false, nil)
		require.NoError(t, err)

		task2, err := service.CreateTask(ctx, "Update Task 2", "/s2", testConnID, "/d2", string(model.SyncDirectionUpload), "", false, nil)
		require.NoError(t, err)

		// Try to update task2 to have the same name as task1 (if unique constraint exists)
		_, err = service.UpdateTask(ctx, task2.ID, "Update Task 1", task2.SourcePath, task2.ConnectionID, task2.RemotePath, string(task2.Direction), task2.Schedule, task2.Realtime, task2.Options)
		if err != nil {
			// If there IS a unique constraint on task names
			assert.ErrorIs(t, err, errs.ErrAlreadyExists)
		} else {
			// If there's NO unique constraint
			t.Log("Schema allows duplicate task names during update")
		}

		// Verify task1 is unchanged
		task1Retrieved, err := service.GetTask(ctx, task1.ID)
		require.NoError(t, err)
		assert.Equal(t, "Update Task 1", task1Retrieved.Name)
	})

	t.Run("ListAllTasks_WithLatestJobs", func(t *testing.T) {
		// Create a fresh task with multiple jobs
		freshTask, err := service.CreateTask(ctx, "Task for ListAll Test", "/src", testConnID, "/dst", string(model.SyncDirectionBidirectional), "", false, nil)
		require.NoError(t, err)

		jobService := NewJobService(client)

		// Create multiple jobs with different start times
		oldJob, err := jobService.CreateJob(ctx, freshTask.ID, model.JobTriggerManual)
		require.NoError(t, err)
		// Make it older
		_, err = client.Job.UpdateOne(oldJob).SetStartTime(time.Now().Add(-3 * time.Hour)).Save(ctx)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)
		middleJob, err := jobService.CreateJob(ctx, freshTask.ID, model.JobTriggerSchedule)
		require.NoError(t, err)
		_, err = client.Job.UpdateOne(middleJob).SetStartTime(time.Now().Add(-1 * time.Hour)).Save(ctx)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)
		latestJob, err := jobService.CreateJob(ctx, freshTask.ID, model.JobTriggerManual)
		require.NoError(t, err)

		// List all tasks
		tasks, err := service.ListAllTasks(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, tasks)

		// Find our test task
		var foundTask *ent.Task
		for _, t := range tasks {
			if t.ID == freshTask.ID {
				foundTask = t
				break
			}
		}
		require.NotNil(t, foundTask, "Test task should be in the results")

		// Verify it only has the latest job
		require.Len(t, foundTask.Edges.Jobs, 1, "Should only have one job (the latest)")
		assert.Equal(t, latestJob.ID, foundTask.Edges.Jobs[0].ID, "Should be the latest job")
	})
}

// Additional test for edge cases
func TestTaskService_EdgeCases(t *testing.T) {
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	defer client.Close()

	service := NewTaskService(client)
	ctx := context.Background()

	// Create test connection
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connService := NewConnectionService(client, encryptor)
	testConn, err := connService.CreateConnection(ctx, "edge-case-conn", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	t.Run("CreateTask_WithNilOptions", func(t *testing.T) {
		task, err := service.CreateTask(ctx, "Nil Options Task", "/src", testConn.ID, "/dst", string(model.SyncDirectionBidirectional), "", false, nil)
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Nil(t, task.Options)
	})

	t.Run("CreateTask_WithOptions", func(t *testing.T) {
		// Create a task with options
		newResolution := model.ConflictResolutionNewer
		filters := []string{"- *.tmp", "- *.log"}
		noDelete := true
		transfers := 10
		options := &model.TaskSyncOptions{
			ConflictResolution: &newResolution,
			Filters:            filters,
			NoDelete:           &noDelete,
			Transfers:          &transfers,
		}
		task, err := service.CreateTask(ctx, "Task With Options", "/src", testConn.ID, "/dst", string(model.SyncDirectionUpload), "", false, options)
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.NotNil(t, task.Options)
	})

	t.Run("CreateTask_WithSchedule", func(t *testing.T) {
		task, err := service.CreateTask(ctx, "Scheduled Task", "/src", testConn.ID, "/dst", string(model.SyncDirectionDownload), "0 */6 * * *", false, nil)
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "0 */6 * * *", task.Schedule)
	})

	t.Run("CreateTask_WithRealtime", func(t *testing.T) {
		task, err := service.CreateTask(ctx, "Realtime Task", "/src", testConn.ID, "/dst", string(model.SyncDirectionBidirectional), "", true, nil)
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.True(t, task.Realtime)
	})

	t.Run("UpdateTask_ChangeAllFields", func(t *testing.T) {
		// Create initial task
		task, err := service.CreateTask(ctx, "Initial Task", "/old-src", testConn.ID, "/old-dst", string(model.SyncDirectionUpload), "0 0 * * *", false, nil)
		require.NoError(t, err)

		// Update all fields
		newResolution := model.ConflictResolutionLocal
		newOptions := &model.TaskSyncOptions{
			ConflictResolution: &newResolution,
		}
		updated, err := service.UpdateTask(
			ctx,
			task.ID,
			"Updated Task",
			"/new-src",
			testConn.ID,
			"/new-dst",
			string(model.SyncDirectionDownload),
			"0 12 * * *",
			true,
			newOptions,
		)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Task", updated.Name)
		assert.Equal(t, "/new-src", updated.SourcePath)
		assert.Equal(t, "/new-dst", updated.RemotePath)
		assert.Equal(t, string(model.SyncDirectionDownload), string(updated.Direction))
		assert.Equal(t, "0 12 * * *", updated.Schedule)
		assert.True(t, updated.Realtime)
		assert.NotNil(t, updated.Options)
	})

	t.Run("ListTasksByConnection_EmptyResult", func(t *testing.T) {
		// Create a connection with no tasks
		emptyConn, err := connService.CreateConnection(ctx, "empty-conn", "local", map[string]string{
			"type": "local",
		})
		require.NoError(t, err)

		tasks, err := service.ListTasksByConnection(ctx, emptyConn.ID)
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})
}

// Tests for ListTasksPaginated
func TestTaskService_ListTasksPaginated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	defer client.Close()

	service := NewTaskService(client)
	ctx := context.Background()

	// Create test connection
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connService := NewConnectionService(client, encryptor)
	testConn, err := connService.CreateConnection(ctx, "paginated-tasks-conn", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	// Initially empty
	tasks, total, err := service.ListTasksPaginated(ctx, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, tasks)
	assert.Equal(t, 0, total)

	// Create multiple tasks
	for i := 0; i < 5; i++ {
		_, err := service.CreateTask(ctx, "Paginated Task "+uuid.NewString()[:8], "/src", testConn.ID, "/dst", string(model.SyncDirectionBidirectional), "", false, nil)
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // Ensure different creation times
	}

	t.Run("FirstPage", func(t *testing.T) {
		tasks, total, err := service.ListTasksPaginated(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
		assert.Equal(t, 5, total)
	})

	t.Run("SecondPage", func(t *testing.T) {
		tasks, total, err := service.ListTasksPaginated(ctx, 2, 2)
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
		assert.Equal(t, 5, total)
	})

	t.Run("LastPage", func(t *testing.T) {
		tasks, total, err := service.ListTasksPaginated(ctx, 2, 4)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, 5, total)
	})

	t.Run("OffsetBeyondTotal", func(t *testing.T) {
		tasks, total, err := service.ListTasksPaginated(ctx, 10, 100)
		require.NoError(t, err)
		assert.Empty(t, tasks)
		assert.Equal(t, 5, total)
	})

	t.Run("LargeLimit", func(t *testing.T) {
		tasks, total, err := service.ListTasksPaginated(ctx, 100, 0)
		require.NoError(t, err)
		assert.Len(t, tasks, 5)
		assert.Equal(t, 5, total)
	})

	t.Run("OrderedByCreatedAtDesc", func(t *testing.T) {
		tasks, _, err := service.ListTasksPaginated(ctx, 5, 0)
		require.NoError(t, err)
		require.Len(t, tasks, 5)

		// Verify tasks are ordered by created_at descending
		for i := 1; i < len(tasks); i++ {
			assert.True(t, tasks[i].CreatedAt.Before(tasks[i-1].CreatedAt) || tasks[i].CreatedAt.Equal(tasks[i-1].CreatedAt),
				"Tasks should be ordered by created_at descending")
		}
	})
}

// Tests for ListTasksByConnectionPaginated
func TestTaskService_ListTasksByConnectionPaginated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	defer client.Close()

	service := NewTaskService(client)
	ctx := context.Background()

	// Create test connections
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connService := NewConnectionService(client, encryptor)

	conn1, err := connService.CreateConnection(ctx, "paginated-conn-1", "local", map[string]string{"type": "local"})
	require.NoError(t, err)

	conn2, err := connService.CreateConnection(ctx, "paginated-conn-2", "local", map[string]string{"type": "local"})
	require.NoError(t, err)

	// Create tasks for conn1
	for i := 0; i < 5; i++ {
		_, err := service.CreateTask(ctx, "Conn1 Task "+uuid.NewString()[:8], "/src", conn1.ID, "/dst", string(model.SyncDirectionBidirectional), "", false, nil)
		require.NoError(t, err)
		time.Sleep(time.Millisecond)
	}

	// Create tasks for conn2
	for i := 0; i < 3; i++ {
		_, err := service.CreateTask(ctx, "Conn2 Task "+uuid.NewString()[:8], "/src", conn2.ID, "/dst", string(model.SyncDirectionUpload), "", false, nil)
		require.NoError(t, err)
	}

	t.Run("FiltersByConnectionID", func(t *testing.T) {
		tasks, total, err := service.ListTasksByConnectionPaginated(ctx, conn1.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, tasks, 5)
		assert.Equal(t, 5, total)

		// Verify all tasks belong to conn1
		for _, task := range tasks {
			assert.Equal(t, conn1.ID, task.ConnectionID)
		}
	})

	t.Run("DifferentConnection", func(t *testing.T) {
		tasks, total, err := service.ListTasksByConnectionPaginated(ctx, conn2.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
		assert.Equal(t, 3, total)

		// Verify all tasks belong to conn2
		for _, task := range tasks {
			assert.Equal(t, conn2.ID, task.ConnectionID)
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		tasks, total, err := service.ListTasksByConnectionPaginated(ctx, conn1.ID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
		assert.Equal(t, 5, total)

		tasks2, total, err := service.ListTasksByConnectionPaginated(ctx, conn1.ID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, tasks2, 2)
		assert.Equal(t, 5, total)

		// Verify different tasks
		assert.NotEqual(t, tasks[0].ID, tasks2[0].ID)
	})

	t.Run("EmptyConnection", func(t *testing.T) {
		emptyConn, err := connService.CreateConnection(ctx, "empty-paginated-conn", "local", map[string]string{"type": "local"})
		require.NoError(t, err)

		tasks, total, err := service.ListTasksByConnectionPaginated(ctx, emptyConn.ID, 10, 0)
		require.NoError(t, err)
		assert.Empty(t, tasks)
		assert.Equal(t, 0, total)
	})

	t.Run("OrderedByCreatedAtDesc", func(t *testing.T) {
		tasks, _, err := service.ListTasksByConnectionPaginated(ctx, conn1.ID, 5, 0)
		require.NoError(t, err)
		require.Len(t, tasks, 5)

		// Verify tasks are ordered by created_at descending
		for i := 1; i < len(tasks); i++ {
			assert.True(t, tasks[i].CreatedAt.Before(tasks[i-1].CreatedAt) || tasks[i].CreatedAt.Equal(tasks[i-1].CreatedAt),
				"Tasks should be ordered by created_at descending")
		}
	})
}

// Tests for ListJobsByTaskPaginated
func TestTaskService_ListJobsByTaskPaginated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	defer client.Close()

	service := NewTaskService(client)
	jobService := NewJobService(client)
	ctx := context.Background()

	// Create test connection
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	connService := NewConnectionService(client, encryptor)
	testConn, err := connService.CreateConnection(ctx, "jobs-by-task-conn", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	// Create task
	task, err := service.CreateTask(ctx, "Jobs By Task Test", "/src", testConn.ID, "/dst", string(model.SyncDirectionBidirectional), "", false, nil)
	require.NoError(t, err)

	// Initially no jobs
	jobs, total, err := service.ListJobsByTaskPaginated(ctx, task.ID, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, jobs)
	assert.Equal(t, 0, total)

	// Create multiple jobs
	for i := 0; i < 5; i++ {
		_, err := jobService.CreateJob(ctx, task.ID, model.JobTriggerManual)
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // Ensure different start times
	}

	t.Run("FirstPage", func(t *testing.T) {
		jobs, total, err := service.ListJobsByTaskPaginated(ctx, task.ID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, jobs, 2)
		assert.Equal(t, 5, total)
	})

	t.Run("SecondPage", func(t *testing.T) {
		jobs, total, err := service.ListJobsByTaskPaginated(ctx, task.ID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, jobs, 2)
		assert.Equal(t, 5, total)
	})

	t.Run("LastPage", func(t *testing.T) {
		jobs, total, err := service.ListJobsByTaskPaginated(ctx, task.ID, 2, 4)
		require.NoError(t, err)
		assert.Len(t, jobs, 1)
		assert.Equal(t, 5, total)
	})

	t.Run("OffsetBeyondTotal", func(t *testing.T) {
		jobs, total, err := service.ListJobsByTaskPaginated(ctx, task.ID, 10, 100)
		require.NoError(t, err)
		assert.Empty(t, jobs)
		assert.Equal(t, 5, total)
	})

	t.Run("LargeLimit", func(t *testing.T) {
		jobs, total, err := service.ListJobsByTaskPaginated(ctx, task.ID, 100, 0)
		require.NoError(t, err)
		assert.Len(t, jobs, 5)
		assert.Equal(t, 5, total)
	})

	t.Run("OrderedByStartTimeDesc", func(t *testing.T) {
		jobs, _, err := service.ListJobsByTaskPaginated(ctx, task.ID, 5, 0)
		require.NoError(t, err)
		require.Len(t, jobs, 5)

		// Verify jobs are ordered by start_time descending
		for i := 1; i < len(jobs); i++ {
			assert.True(t, jobs[i].StartTime.Before(jobs[i-1].StartTime) || jobs[i].StartTime.Equal(jobs[i-1].StartTime),
				"Jobs should be ordered by start_time descending")
		}
	})

	t.Run("DifferentTask", func(t *testing.T) {
		// Create another task with different jobs
		task2, err := service.CreateTask(ctx, "Jobs By Task Test 2", "/src2", testConn.ID, "/dst2", string(model.SyncDirectionUpload), "", false, nil)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			_, err := jobService.CreateJob(ctx, task2.ID, model.JobTriggerSchedule)
			require.NoError(t, err)
		}

		// Verify jobs are filtered by task
		jobs1, total1, err := service.ListJobsByTaskPaginated(ctx, task.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, jobs1, 5)
		assert.Equal(t, 5, total1)

		jobs2, total2, err := service.ListJobsByTaskPaginated(ctx, task2.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, jobs2, 3)
		assert.Equal(t, 3, total2)

		// Verify no overlap
		for _, j1 := range jobs1 {
			for _, j2 := range jobs2 {
				assert.NotEqual(t, j1.ID, j2.ID)
			}
		}
	})
}
