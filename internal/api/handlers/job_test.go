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

	// 1. Create a task and a job
	ctx := context.Background()

	task, err := ts.TaskService.CreateTask(ctx, "Job Test Task", "/src", "local", "/dst", "upload", "", false, nil)
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

	ctx := context.Background()
	task, err := ts.TaskService.CreateTask(ctx, "Job Detail Task", "/src", "local", "/dst", "upload", "", false, nil)
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

	ctx := context.Background()
	task, err := ts.TaskService.CreateTask(ctx, "Progress Task", "/src", "local", "/dst", "upload", "", false, nil)
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
