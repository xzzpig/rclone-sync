package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/handlers"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

func TestJobAPI_ListJobs(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create a test connection first
	connID := createTestConnection(t, ts, "test-local")

	// 1. Create a task and a job
	ctx := context.Background()

	task, err := ts.TaskService.CreateTask(ctx, "Job Test Task", "/src", connID, "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job1, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	// 2. List jobs
	resp, err := http.Get(ts.Server.URL + "/api/jobs")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.Job]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(page.Data), 1)
	assert.Equal(t, 1, page.Total)
	assert.Equal(t, job1.ID, page.Data[0].ID)
}

func TestJobAPI_GetJob(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create a test connection first
	connID := createTestConnection(t, ts, "test-local")

	ctx := context.Background()
	task, err := ts.TaskService.CreateTask(ctx, "Job Detail Task", "/src", connID, "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	// Get job details
	resp, err := http.Get(ts.Server.URL + "/api/jobs/" + job.ID.String())
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var fetchedJob ent.Job
	err = json.NewDecoder(resp.Body).Decode(&fetchedJob)
	require.NoError(t, err)
	assert.Equal(t, job.ID, fetchedJob.ID)
}

func TestJobAPI_GetJobProgress_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create a test connection first
	connID := createTestConnection(t, ts, "test-local")

	ctx := context.Background()
	task, err := ts.TaskService.CreateTask(ctx, "Progress Task", "/src", connID, "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	// Get progress for a job that isn't running in memory
	resp, err := http.Get(ts.Server.URL + "/api/jobs/" + job.ID.String() + "/progress")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Expect 404 because the job is not active in the sync engine
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestJobAPI_GetJobProgress_InvalidID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/jobs/invalid-uuid/progress")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestJobAPI_ListJobs_WithPagination(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	connID := createTestConnection(t, ts, "test-local")

	ctx := context.Background()
	task, err := ts.TaskService.CreateTask(ctx, "Pagination Task", "/src", connID, "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	// Create multiple jobs
	for i := 0; i < 5; i++ {
		_, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
		require.NoError(t, err)
	}

	// Test pagination with limit
	resp, err := http.Get(ts.Server.URL + "/api/jobs?limit=2&offset=0")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.Job]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)
	assert.Len(t, page.Data, 2)
	assert.Equal(t, 5, page.Total)
}

func TestJobAPI_ListJobs_ByTaskID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	connID := createTestConnection(t, ts, "test-local")

	ctx := context.Background()
	task1, err := ts.TaskService.CreateTask(ctx, "Task 1", "/src1", connID, "/dst1", "upload", "", false, nil)
	require.NoError(t, err)
	task2, err := ts.TaskService.CreateTask(ctx, "Task 2", "/src2", connID, "/dst2", "upload", "", false, nil)
	require.NoError(t, err)

	// Create jobs for both tasks
	_, err = ts.JobService.CreateJob(ctx, task1.ID, "manual")
	require.NoError(t, err)
	_, err = ts.JobService.CreateJob(ctx, task2.ID, "manual")
	require.NoError(t, err)

	// List jobs for task 1 only
	resp, err := http.Get(ts.Server.URL + "/api/jobs?task_id=" + task1.ID.String())
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.Job]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)
	assert.Len(t, page.Data, 1)
	assert.Equal(t, 1, page.Total)
}

func TestJobAPI_ListJobs_ByConnectionID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	conn1ID := createTestConnection(t, ts, "test-local-1")
	conn2ID := createTestConnection(t, ts, "test-local-2")

	ctx := context.Background()
	task1, err := ts.TaskService.CreateTask(ctx, "Task 1", "/src1", conn1ID, "/dst1", "upload", "", false, nil)
	require.NoError(t, err)
	task2, err := ts.TaskService.CreateTask(ctx, "Task 2", "/src2", conn2ID, "/dst2", "upload", "", false, nil)
	require.NoError(t, err)

	// Create jobs for both tasks
	_, err = ts.JobService.CreateJob(ctx, task1.ID, "manual")
	require.NoError(t, err)
	_, err = ts.JobService.CreateJob(ctx, task2.ID, "manual")
	require.NoError(t, err)

	// List jobs for connection 1 only
	resp, err := http.Get(ts.Server.URL + "/api/jobs?connection_id=" + conn1ID.String())
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.Job]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)
	assert.Len(t, page.Data, 1)
	assert.Equal(t, 1, page.Total)
}

func TestJobAPI_ListJobs_InvalidConnectionID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/jobs?connection_id=invalid-uuid")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestJobAPI_GetJob_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/jobs/00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestJobAPI_GetJob_InvalidID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/jobs/invalid-uuid")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
