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

	// Create a test connection first
	connID := createTestConnection(t, ts, "test-local")

	// 1. Create a task via POST /tasks
	taskPayload := map[string]interface{}{
		"name":          "Test Task",
		"source_path":   "/tmp/source",
		"connection_id": connID.String(),
		"remote_path":   "/tmp/dest",
		"direction":     "bidirectional",
		"schedule":      "",
		"realtime":      false,
		"options":       map[string]interface{}{},
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

	// Create a test connection first
	connID := createTestConnection(t, ts, "test-local")

	// 1. Pre-create a task in the database using the service
	task, err := ts.TaskService.CreateTask(context.Background(),
		"Task to Run",
		"/tmp/source",
		connID,
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

	jobs, err := ts.JobService.ListJobs(context.Background(), &task.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, task.ID, jobs[0].Edges.Task.ID)
}

func TestTaskAPI_Get(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create a test connection first
	connID := createTestConnection(t, ts, "test-local")

	task, err := ts.TaskService.CreateTask(context.Background(), "Get Task", "/src", connID, "/dst", "upload", "", false, nil)
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

	// Create a test connection first
	connID := createTestConnection(t, ts, "test-local")

	task, err := ts.TaskService.CreateTask(context.Background(), "Update Task", "/src", connID, "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	updatePayload := map[string]interface{}{
		"name":          "Updated Task",
		"source_path":   "/src/new",
		"connection_id": connID.String(),
		"remote_path":   "/dst/new",
		"direction":     "download",
		"schedule":      "@daily",
		"realtime":      true,
		"options":       map[string]interface{}{},
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

	// Create a test connection first
	connID := createTestConnection(t, ts, "test-local")

	task, err := ts.TaskService.CreateTask(context.Background(), "Delete Task", "/src", connID, "/dst", "upload", "", false, nil)
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

func TestTaskAPI_Create_InvalidSchedule(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	connID := createTestConnection(t, ts, "test-local")

	taskPayload := map[string]interface{}{
		"name":          "Test Task",
		"source_path":   "/tmp/source",
		"connection_id": connID.String(),
		"remote_path":   "/tmp/dest",
		"direction":     "upload",
		"schedule":      "invalid cron expression",
		"realtime":      false,
	}
	payloadBytes, _ := json.Marshal(taskPayload)

	resp, err := http.Post(ts.Server.URL+"/api/tasks", "application/json", bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTaskAPI_Create_InvalidConnectionID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	taskPayload := map[string]interface{}{
		"name":          "Test Task",
		"source_path":   "/tmp/source",
		"connection_id": "invalid-uuid",
		"remote_path":   "/tmp/dest",
		"direction":     "upload",
	}
	payloadBytes, _ := json.Marshal(taskPayload)

	resp, err := http.Post(ts.Server.URL+"/api/tasks", "application/json", bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTaskAPI_Create_WithRealtime(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	connID := createTestConnection(t, ts, "test-local")

	taskPayload := map[string]interface{}{
		"name":          "Realtime Task",
		"source_path":   "/tmp/source",
		"connection_id": connID.String(),
		"remote_path":   "/tmp/dest",
		"direction":     "bidirectional",
		"realtime":      true,
	}
	payloadBytes, _ := json.Marshal(taskPayload)

	resp, err := http.Post(ts.Server.URL+"/api/tasks", "application/json", bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdTask ent.Task
	err = json.NewDecoder(resp.Body).Decode(&createdTask)
	require.NoError(t, err)
	assert.True(t, createdTask.Realtime)
}

func TestTaskAPI_Create_WithSchedule(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	connID := createTestConnection(t, ts, "test-local")

	taskPayload := map[string]interface{}{
		"name":          "Scheduled Task",
		"source_path":   "/tmp/source",
		"connection_id": connID.String(),
		"remote_path":   "/tmp/dest",
		"direction":     "upload",
		"schedule":      "@daily",
	}
	payloadBytes, _ := json.Marshal(taskPayload)

	resp, err := http.Post(ts.Server.URL+"/api/tasks", "application/json", bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdTask ent.Task
	err = json.NewDecoder(resp.Body).Decode(&createdTask)
	require.NoError(t, err)
	assert.Equal(t, "@daily", createdTask.Schedule)
}

func TestTaskAPI_List_ByConnectionID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	conn1ID := createTestConnection(t, ts, "test-local-1")
	conn2ID := createTestConnection(t, ts, "test-local-2")

	// Create tasks for different connections
	_, err := ts.TaskService.CreateTask(context.Background(), "Task 1", "/src1", conn1ID, "/dst1", "upload", "", false, nil)
	require.NoError(t, err)
	_, err = ts.TaskService.CreateTask(context.Background(), "Task 2", "/src2", conn2ID, "/dst2", "upload", "", false, nil)
	require.NoError(t, err)

	// List tasks for connection 1
	resp, err := http.Get(ts.Server.URL + "/api/tasks?connection_id=" + conn1ID.String())
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var tasks []*ent.Task
	err = json.NewDecoder(resp.Body).Decode(&tasks)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Task 1", tasks[0].Name)
}

func TestTaskAPI_List_InvalidConnectionID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/tasks?connection_id=invalid-uuid")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTaskAPI_Update_PartialFields(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	connID := createTestConnection(t, ts, "test-local")

	task, err := ts.TaskService.CreateTask(context.Background(), "Original Task", "/src", connID, "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	// Only update name
	updatePayload := map[string]interface{}{
		"name": "Updated Name",
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
	assert.Equal(t, "Updated Name", updatedTask.Name)
	assert.Equal(t, "/src", updatedTask.SourcePath) // Should remain unchanged
}

func TestTaskAPI_Update_ScheduleChange(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	connID := createTestConnection(t, ts, "test-local")

	task, err := ts.TaskService.CreateTask(context.Background(), "Task", "/src", connID, "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	// Add schedule
	schedule := "@daily"
	updatePayload := map[string]interface{}{
		"schedule": schedule,
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
	assert.Equal(t, "@daily", updatedTask.Schedule)
}

func TestTaskAPI_Update_RealtimeChange(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	connID := createTestConnection(t, ts, "test-local")

	task, err := ts.TaskService.CreateTask(context.Background(), "Task", "/src", connID, "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	// Enable realtime
	realtime := true
	updatePayload := map[string]interface{}{
		"realtime": realtime,
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
	assert.True(t, updatedTask.Realtime)
}

func TestTaskAPI_Update_InvalidSchedule(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	connID := createTestConnection(t, ts, "test-local")

	task, err := ts.TaskService.CreateTask(context.Background(), "Task", "/src", connID, "/dst", "upload", "", false, nil)
	require.NoError(t, err)

	updatePayload := map[string]interface{}{
		"schedule": "invalid cron",
	}
	payloadBytes, _ := json.Marshal(updatePayload)

	req, err := http.NewRequest(http.MethodPut, ts.Server.URL+"/api/tasks/"+task.ID.String(), bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTaskAPI_Delete_WithRealtimeAndSchedule(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	connID := createTestConnection(t, ts, "test-local")

	task, err := ts.TaskService.CreateTask(context.Background(), "Task", "/src", connID, "/dst", "upload", "@daily", true, nil)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodDelete, ts.Server.URL+"/api/tasks/"+task.ID.String(), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestTaskAPI_Delete_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	req, err := http.NewRequest(http.MethodDelete, ts.Server.URL+"/api/tasks/00000000-0000-0000-0000-000000000000", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestTaskAPI_Run_InvalidID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Post(ts.Server.URL+"/api/tasks/invalid-uuid/run", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
