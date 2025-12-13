package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/handlers"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

func TestLogAPI_ListLogs_MissingRemoteName(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Request without remote_name should return 400
	resp, err := http.Get(ts.Server.URL + "/api/logs")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var appErr handlers.AppError
	err = json.NewDecoder(resp.Body).Decode(&appErr)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, appErr.Code)
	assert.Contains(t, appErr.Message, "remote_name")
}

func TestLogAPI_ListLogs_Success(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	// Create a task and job for testing
	task, err := ts.TaskService.CreateTask(ctx, "Log Test Task", "/src", "testremote", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	// Add job logs using the service
	_, err = ts.JobService.AddJobLog(ctx, job.ID, "info", "upload", "", 1024)
	require.NoError(t, err)

	_, err = ts.JobService.AddJobLog(ctx, job.ID, "error", "error", "", 0)
	require.NoError(t, err)

	// List logs with remote_name
	resp, err := http.Get(ts.Server.URL + "/api/logs?remote_name=testremote")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(page.Data), 2)
	assert.Equal(t, 2, page.Total)
}

func TestLogAPI_ListLogs_WithTaskID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	// Create two tasks
	task1, err := ts.TaskService.CreateTask(ctx, "Task 1", "/src1", "testremote", "/dst1", "upload", "", false, nil)
	require.NoError(t, err)

	task2, err := ts.TaskService.CreateTask(ctx, "Task 2", "/src2", "testremote", "/dst2", "upload", "", false, nil)
	require.NoError(t, err)

	job1, err := ts.JobService.CreateJob(ctx, task1.ID, "manual")
	require.NoError(t, err)

	job2, err := ts.JobService.CreateJob(ctx, task2.ID, "manual")
	require.NoError(t, err)

	// Add logs for both tasks
	_, err = ts.JobService.AddJobLog(ctx, job1.ID, "info", "upload", "", 2048)
	require.NoError(t, err)

	_, err = ts.JobService.AddJobLog(ctx, job2.ID, "info", "download", "", 4096)
	require.NoError(t, err)

	// Filter by task1 ID
	url := fmt.Sprintf("%s/api/logs?remote_name=testremote&task_id=%s", ts.Server.URL, task1.ID.String())
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)

	assert.Equal(t, 1, page.Total)
	assert.Len(t, page.Data, 1)
}

func TestLogAPI_ListLogs_WithJobID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	task, err := ts.TaskService.CreateTask(ctx, "Task", "/src", "testremote", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job1, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	job2, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	// Add logs for both jobs
	_, err = ts.JobService.AddJobLog(ctx, job1.ID, "info", "upload", "", 512)
	require.NoError(t, err)

	_, err = ts.JobService.AddJobLog(ctx, job2.ID, "info", "download", "", 1024)
	require.NoError(t, err)

	// Filter by job1 ID
	url := fmt.Sprintf("%s/api/logs?remote_name=testremote&job_id=%s", ts.Server.URL, job1.ID.String())
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)

	assert.Equal(t, 1, page.Total)
	assert.Len(t, page.Data, 1)
}

func TestLogAPI_ListLogs_WithLevel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	task, err := ts.TaskService.CreateTask(ctx, "Task", "/src", "testremote", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	// Add logs with different levels
	_, err = ts.JobService.AddJobLog(ctx, job.ID, "info", "upload", "", 1024)
	require.NoError(t, err)

	_, err = ts.JobService.AddJobLog(ctx, job.ID, "error", "error", "", 0)
	require.NoError(t, err)

	_, err = ts.JobService.AddJobLog(ctx, job.ID, "warning", "delete", "", 0)
	require.NoError(t, err)

	// Filter by error level
	url := fmt.Sprintf("%s/api/logs?remote_name=testremote&level=error", ts.Server.URL)
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)

	assert.Equal(t, 1, page.Total)
	assert.Len(t, page.Data, 1)
	assert.Equal(t, "error", string(page.Data[0].Level))
}

func TestLogAPI_ListLogs_WithPagination(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	task, err := ts.TaskService.CreateTask(ctx, "Task", "/src", "testremote", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	// Create 10 logs
	for i := 0; i < 10; i++ {
		_, err = ts.JobService.AddJobLog(ctx, job.ID, "info", "upload", "", int64(i*100))
		require.NoError(t, err)
	}

	// Request first page with limit=5
	resp, err := http.Get(ts.Server.URL + "/api/logs?remote_name=testremote&limit=5&offset=0")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page1 handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page1)
	require.NoError(t, err)

	assert.Equal(t, 10, page1.Total)
	assert.Len(t, page1.Data, 5)

	// Request second page with offset=5
	resp, err = http.Get(ts.Server.URL + "/api/logs?remote_name=testremote&limit=5&offset=5")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page2 handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page2)
	require.NoError(t, err)

	assert.Equal(t, 10, page2.Total)
	assert.Len(t, page2.Data, 5)

	// Ensure pages have different data
	assert.NotEqual(t, page1.Data[0].ID, page2.Data[0].ID)
}

func TestLogAPI_ListLogs_InvalidTaskID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	task, err := ts.TaskService.CreateTask(ctx, "Task", "/src", "testremote", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	_, err = ts.JobService.AddJobLog(ctx, job.ID, "info", "upload", "", 256)
	require.NoError(t, err)

	// Invalid UUID should be ignored, query should still succeed
	url := fmt.Sprintf("%s/api/logs?remote_name=testremote&task_id=invalid-uuid", ts.Server.URL)
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)

	// Should return all logs since invalid UUID is ignored
	assert.GreaterOrEqual(t, page.Total, 1)
}

func TestLogAPI_ListLogs_InvalidJobID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	task, err := ts.TaskService.CreateTask(ctx, "Task", "/src", "testremote", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	_, err = ts.JobService.AddJobLog(ctx, job.ID, "info", "upload", "", 512)
	require.NoError(t, err)

	// Invalid UUID should be ignored, query should still succeed
	url := fmt.Sprintf("%s/api/logs?remote_name=testremote&job_id=not-a-uuid", ts.Server.URL)
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)

	// Should return all logs since invalid UUID is ignored
	assert.GreaterOrEqual(t, page.Total, 1)
}

func TestLogAPI_ListLogs_NoResults(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Query with remote_name that has no logs
	resp, err := http.Get(ts.Server.URL + "/api/logs?remote_name=nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)

	assert.Equal(t, 0, page.Total)
	assert.Len(t, page.Data, 0)
}

func TestLogAPI_ListLogs_DefaultPagination(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	task, err := ts.TaskService.CreateTask(ctx, "Task", "/src", "testremote", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	// Create 5 logs
	for i := 0; i < 5; i++ {
		_, err = ts.JobService.AddJobLog(ctx, job.ID, "info", "download", "", int64(i*200))
		require.NoError(t, err)
	}

	// Request without pagination parameters (should use defaults)
	resp, err := http.Get(ts.Server.URL + "/api/logs?remote_name=testremote")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)

	assert.Equal(t, 5, page.Total)
	assert.Len(t, page.Data, 5)
}

func TestLogAPI_ListLogs_CombinedFilters(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	task, err := ts.TaskService.CreateTask(ctx, "Task", "/src", "testremote", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	job, err := ts.JobService.CreateJob(ctx, task.ID, "manual")
	require.NoError(t, err)

	// Add logs with different levels
	_, err = ts.JobService.AddJobLog(ctx, job.ID, "info", "upload", "", 512)
	require.NoError(t, err)

	_, err = ts.JobService.AddJobLog(ctx, job.ID, "error", "error", "", 0)
	require.NoError(t, err)

	// Filter by task_id, job_id, and level
	url := fmt.Sprintf("%s/api/logs?remote_name=testremote&task_id=%s&job_id=%s&level=error",
		ts.Server.URL, task.ID.String(), job.ID.String())
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)

	assert.Equal(t, 1, page.Total)
	assert.Len(t, page.Data, 1)
	assert.Equal(t, "error", string(page.Data[0].Level))
}

func TestLogAPI_ListLogs_MultipleRemotes(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	// Create tasks for different remotes
	task1, err := ts.TaskService.CreateTask(ctx, "Task 1", "/src1", "remote1", "/dst1", "upload", "", false, nil)
	require.NoError(t, err)

	task2, err := ts.TaskService.CreateTask(ctx, "Task 2", "/src2", "remote2", "/dst2", "upload", "", false, nil)
	require.NoError(t, err)

	job1, err := ts.JobService.CreateJob(ctx, task1.ID, "manual")
	require.NoError(t, err)

	job2, err := ts.JobService.CreateJob(ctx, task2.ID, "manual")
	require.NoError(t, err)

	_, err = ts.JobService.AddJobLog(ctx, job1.ID, "info", "upload", "", 128)
	require.NoError(t, err)

	_, err = ts.JobService.AddJobLog(ctx, job2.ID, "info", "download", "", 256)
	require.NoError(t, err)

	// Query for remote1 only
	resp, err := http.Get(ts.Server.URL + "/api/logs?remote_name=remote1")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var page handlers.Page[[]*ent.JobLog]
	err = json.NewDecoder(resp.Body).Decode(&page)
	require.NoError(t, err)

	assert.Equal(t, 1, page.Total)
	assert.Len(t, page.Data, 1)
}
