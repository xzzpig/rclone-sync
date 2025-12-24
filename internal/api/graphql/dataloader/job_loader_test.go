package dataloader_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/dataloader"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// setupJobTestDB creates an in-memory database with all required services for testing.
func setupJobTestDB(t *testing.T) (*ent.Client, *services.JobService, *services.TaskService, *services.ConnectionService) {
	t.Helper()

	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)

	client, err := db.InitDB(db.InitDBOptions{
		DSN:           "file:ent?mode=memory&cache=shared&_fk=1",
		MigrationMode: db.MigrationModeAuto,
	})
	require.NoError(t, err)

	encryptor, err := crypto.NewEncryptor("test-encryption-key-32-bytes!!")
	require.NoError(t, err)

	connectionService := services.NewConnectionService(client, encryptor)
	taskService := services.NewTaskService(client)
	jobService := services.NewJobService(client)

	// Install DBStorage for rclone configuration
	storage := rclone.NewDBStorage(connectionService)
	storage.Install()

	t.Cleanup(func() {
		client.Close()
	})

	return client, jobService, taskService, connectionService
}

// createTestTask creates a test task for job tests.
func createTestTask(t *testing.T, taskService *services.TaskService, connectionService *services.ConnectionService) *ent.Task {
	t.Helper()
	ctx := context.Background()

	conn, err := connectionService.CreateConnection(ctx, "test-connection", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	task, err := taskService.CreateTask(
		ctx,
		"test-task",
		"/source",
		conn.ID,
		"/remote",
		"UPLOAD",
		"",
		false,
		nil,
	)
	require.NoError(t, err)
	return task
}

// createTestJob creates a test job for the given task.
func createTestJob(t *testing.T, client *ent.Client, taskID uuid.UUID) *ent.Job {
	t.Helper()
	ctx := context.Background()

	j, err := client.Job.Create().
		SetTaskID(taskID).
		SetStatus(model.JobStatusPending).
		SetTrigger(model.JobTriggerManual).
		SetStartTime(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	return j
}

func TestJobLoader_Load_ExistingJob(t *testing.T) {
	client, _, taskService, connectionService := setupJobTestDB(t)
	ctx := context.Background()

	// Create a task first
	task := createTestTask(t, taskService, connectionService)

	// Create a test job
	testJob := createTestJob(t, client, task.ID)

	// Create loader
	loader := dataloader.NewJobLoader(client)

	// Load the job
	result, err := loader.Load(ctx, testJob.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testJob.ID, result.ID)
	assert.Equal(t, model.JobStatusPending, result.Status)
	assert.Equal(t, model.JobTriggerManual, result.Trigger)
}

func TestJobLoader_Load_NonExistentJob(t *testing.T) {
	client, _, _, _ := setupJobTestDB(t)
	ctx := context.Background()

	// Create loader
	loader := dataloader.NewJobLoader(client)

	// Try to load a non-existent job
	nonExistentID := uuid.New()
	result, err := loader.Load(ctx, nonExistentID)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestJobLoader_LoadAll_MultipleJobs(t *testing.T) {
	client, _, taskService, connectionService := setupJobTestDB(t)
	ctx := context.Background()

	// Create a task first
	task := createTestTask(t, taskService, connectionService)

	// Create multiple test jobs
	job1 := createTestJob(t, client, task.ID)
	job2 := createTestJob(t, client, task.ID)
	job3 := createTestJob(t, client, task.ID)

	// Create loader
	loader := dataloader.NewJobLoader(client)

	// Load all jobs in batch
	ids := []uuid.UUID{job1.ID, job2.ID, job3.ID}
	results, err := loader.LoadAll(ctx, ids)

	require.NoError(t, err)
	require.Len(t, results, 3)

	// Check results are in correct order
	assert.Equal(t, job1.ID, results[0].ID)
	assert.Equal(t, job2.ID, results[1].ID)
	assert.Equal(t, job3.ID, results[2].ID)
}

func TestJobLoader_LoadAll_MixedExistingAndNonExistent(t *testing.T) {
	client, _, taskService, connectionService := setupJobTestDB(t)
	ctx := context.Background()

	// Create a task first
	task := createTestTask(t, taskService, connectionService)

	// Create one test job
	job1 := createTestJob(t, client, task.ID)

	nonExistentID := uuid.New()

	// Create loader
	loader := dataloader.NewJobLoader(client)

	// Load batch with mix of existing and non-existing
	ids := []uuid.UUID{job1.ID, nonExistentID}
	results, err := loader.LoadAll(ctx, ids)

	// LoadAll returns error if any item fails
	assert.Error(t, err)

	// Results will still have the length matching input
	require.Len(t, results, 2)

	// First should be present
	assert.NotNil(t, results[0])
	assert.Equal(t, job1.ID, results[0].ID)
}

func TestJobLoader_LoadAll_PreservesOrder(t *testing.T) {
	client, _, taskService, connectionService := setupJobTestDB(t)
	ctx := context.Background()

	// Create a task first
	task := createTestTask(t, taskService, connectionService)

	// Create jobs
	var jobs []*ent.Job
	for i := 0; i < 5; i++ {
		j := createTestJob(t, client, task.ID)
		jobs = append(jobs, j)
	}

	// Create loader
	loader := dataloader.NewJobLoader(client)

	// Request in reverse order
	ids := []uuid.UUID{
		jobs[4].ID,
		jobs[2].ID,
		jobs[0].ID,
		jobs[3].ID,
		jobs[1].ID,
	}

	results, err := loader.LoadAll(ctx, ids)

	require.NoError(t, err)
	require.Len(t, results, 5)

	// Results should match request order
	assert.Equal(t, jobs[4].ID, results[0].ID)
	assert.Equal(t, jobs[2].ID, results[1].ID)
	assert.Equal(t, jobs[0].ID, results[2].ID)
	assert.Equal(t, jobs[3].ID, results[3].ID)
	assert.Equal(t, jobs[1].ID, results[4].ID)
}

func TestJobLoader_LoadAll_EmptySlice(t *testing.T) {
	client, _, _, _ := setupJobTestDB(t)
	ctx := context.Background()

	// Create loader
	loader := dataloader.NewJobLoader(client)

	// Load with empty slice
	results, err := loader.LoadAll(ctx, []uuid.UUID{})

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestJobLoader_LoadAll_DuplicateIDs(t *testing.T) {
	client, _, taskService, connectionService := setupJobTestDB(t)
	ctx := context.Background()

	// Create a task first
	task := createTestTask(t, taskService, connectionService)

	// Create a test job
	testJob := createTestJob(t, client, task.ID)

	// Create loader
	loader := dataloader.NewJobLoader(client)

	// Load same ID multiple times
	ids := []uuid.UUID{testJob.ID, testJob.ID, testJob.ID}
	results, err := loader.LoadAll(ctx, ids)

	require.NoError(t, err)
	require.Len(t, results, 3)

	// All should return the same job
	for _, result := range results {
		assert.NotNil(t, result)
		assert.Equal(t, testJob.ID, result.ID)
	}
}

func TestJobLoader_Load_JobWithDifferentStatuses(t *testing.T) {
	client, _, taskService, connectionService := setupJobTestDB(t)
	ctx := context.Background()

	// Create a task first
	task := createTestTask(t, taskService, connectionService)

	testCases := []struct {
		name   string
		status model.JobStatus
	}{
		{"pending", model.JobStatusPending},
		{"running", model.JobStatusRunning},
		{"success", model.JobStatusSuccess},
		{"failed", model.JobStatusFailed},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a job with specific status
			j, err := client.Job.Create().
				SetTaskID(task.ID).
				SetStatus(tc.status).
				SetTrigger(model.JobTriggerManual).
				SetStartTime(time.Now()).
				Save(ctx)
			require.NoError(t, err)

			// Create loader
			loader := dataloader.NewJobLoader(client)

			// Load the job
			result, err := loader.Load(ctx, j.ID)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tc.status, result.Status)
		})
	}
}

func TestJobLoader_Load_JobWithTriggerTypes(t *testing.T) {
	client, _, taskService, connectionService := setupJobTestDB(t)
	ctx := context.Background()

	// Create a task first
	task := createTestTask(t, taskService, connectionService)

	triggerTypes := []model.JobTrigger{
		model.JobTriggerManual,
		model.JobTriggerSchedule,
		model.JobTriggerRealtime,
	}

	for _, trigger := range triggerTypes {
		t.Run(string(trigger), func(t *testing.T) {
			// Create a job with specific trigger
			j, err := client.Job.Create().
				SetTaskID(task.ID).
				SetStatus(model.JobStatusPending).
				SetTrigger(trigger).
				SetStartTime(time.Now()).
				Save(ctx)
			require.NoError(t, err)

			// Create loader
			loader := dataloader.NewJobLoader(client)

			// Load the job
			result, err := loader.Load(ctx, j.ID)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, trigger, result.Trigger)
		})
	}
}
