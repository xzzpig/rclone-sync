package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

func TestTaskAPI_CreateAndList(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// 1. Create a task via POST /tasks
	taskPayload := map[string]interface{}{
		"name":        "Test Task",
		"source_path": "/tmp/source",
		"remote_name": "local",
		"remote_path": "/tmp/dest",
		"direction":   "bidirectional",
		"schedule":    "",
		"realtime":    false,
		"options":     map[string]interface{}{},
	}
	payloadBytes, _ := json.Marshal(taskPayload)

	resp, err := http.Post(ts.Server.URL+"/api/tasks", "application/json", bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdTask ent.Task
	err = json.NewDecoder(resp.Body).Decode(&createdTask)
	require.NoError(t, err)
	assert.Equal(t, "Test Task", createdTask.Name)
	assert.NotEmpty(t, createdTask.ID)

	// 2. List tasks via GET /tasks
	resp, err = http.Get(ts.Server.URL + "/api/tasks")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var tasks []*ent.Task
	err = json.NewDecoder(resp.Body).Decode(&tasks)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, createdTask.ID, tasks[0].ID)
	assert.Equal(t, "Test Task", tasks[0].Name)
}

func TestTaskAPI_RunTask(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// 1. Pre-create a task in the database using the service
	task, err := ts.TaskService.CreateTask(context.Background(),
		"Task to Run",
		"/tmp/source",
		"local",
		"/tmp/dest",
		"bidirectional",
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 2. Run the task via POST /tasks/:id/run
	resp, err := http.Post(ts.Server.URL+"/api/tasks/"+task.ID.String()+"/run", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode) // The handler returns 200 OK after successfully starting the task

	// 3. (Optional) Verify a job was created for the task
	// We need to wait a bit for the runner to create the job
	time.Sleep(100 * time.Millisecond)

	jobs, err := ts.JobService.ListJobs(context.Background(), &task.ID, "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, task.ID, jobs[0].Edges.Task.ID)
}

func TestTaskAPI_Get(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	task, err := ts.TaskService.CreateTask(context.Background(), "Get Task", "/src", "local", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	resp, err := http.Get(ts.Server.URL + "/api/tasks/" + task.ID.String())
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var fetchedTask ent.Task
	err = json.NewDecoder(resp.Body).Decode(&fetchedTask)
	require.NoError(t, err)
	assert.Equal(t, task.ID, fetchedTask.ID)
	assert.Equal(t, "Get Task", fetchedTask.Name)
}

func TestTaskAPI_Update(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	task, err := ts.TaskService.CreateTask(context.Background(), "Update Task", "/src", "local", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	updatePayload := map[string]interface{}{
		"name":        "Updated Task",
		"source_path": "/src/new",
		"remote_name": "local",
		"remote_path": "/dst/new",
		"direction":   "download",
		"schedule":    "@daily",
		"realtime":    true,
		"options":     map[string]interface{}{},
	}
	payloadBytes, _ := json.Marshal(updatePayload)

	req, err := http.NewRequest(http.MethodPut, ts.Server.URL+"/api/tasks/"+task.ID.String(), bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var updatedTask ent.Task
	err = json.NewDecoder(resp.Body).Decode(&updatedTask)
	require.NoError(t, err)
	assert.Equal(t, "Updated Task", updatedTask.Name)
	assert.Equal(t, "/src/new", updatedTask.SourcePath)
	assert.Equal(t, "download", string(updatedTask.Direction))
}

func TestTaskAPI_Delete(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	task, err := ts.TaskService.CreateTask(context.Background(), "Delete Task", "/src", "local", "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodDelete, ts.Server.URL+"/api/tasks/"+task.ID.String(), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify deletion
	_, err = ts.TaskService.GetTask(context.Background(), task.ID)
	assert.Error(t, err)
	assert.True(t, ent.IsNotFound(err))
}

func TestTaskAPI_Errors(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Test Get NotFound
	resp, err := http.Get(ts.Server.URL + "/api/tasks/00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	// Test Create Invalid Input
	resp, err = http.Post(ts.Server.URL+"/api/tasks", "application/json", bytes.NewBuffer([]byte(`{"name": ""}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}
