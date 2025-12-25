// Package resolver provides GraphQL resolver tests.
// This file contains integration tests that simulate complete user workflows.
package resolver_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

// IntegrationTestSuite tests complete user workflows through GraphQL API.
// These tests simulate real user journeys as performed by the frontend.
// Unlike unit tests that test individual resolvers, integration tests verify
// that multiple components work together correctly in realistic scenarios.
type IntegrationTestSuite struct {
	ResolverTestSuite
	testDir string
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

// SetupTest runs before each test.
func (s *IntegrationTestSuite) SetupTest() {
	s.ResolverTestSuite.SetupTest()

	// Create test directories for file browsing
	s.testDir = s.T().TempDir()
	require.NoError(s.T(), os.MkdirAll(filepath.Join(s.testDir, "source"), 0755))
	require.NoError(s.T(), os.MkdirAll(filepath.Join(s.testDir, "remote"), 0755))
}

// =============================================================================
// Integration Tests - Complete User Workflows
// =============================================================================

// TestUserJourney_CompleteWorkflow tests the complete user workflow:
// 1. Create a connection
// 2. Create a task for that connection
// 3. Run the task manually
// 4. View job history and logs
// 5. Update the task
// 6. Delete the task
// 7. Delete the connection
func (s *IntegrationTestSuite) TestUserJourney_CompleteWorkflow() {
	// Phase 1: Create a connection
	connID := s.Env.CreateTestConnection(s.T(), "my-workflow-connection")

	// Phase 2: Create a task for the connection
	task := s.Env.CreateTestTask(s.T(), "my-sync-task", connID)

	// Phase 3: Run the task manually and verify job is created
	ctx := context.Background()
	job, err := s.Env.JobService.CreateJob(ctx, task.ID, "MANUAL")
	require.NoError(s.T(), err)

	// Verify job appears in the job list
	jobQuery := `
        query($taskId: ID) {
            job {
                list(taskId: $taskId) {
                    items {
                        id
                        status
                        trigger
                    }
                }
            }
        }
    `
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), jobQuery, map[string]interface{}{
		"taskId": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)
	items := gjson.Get(string(resp.Data), "job.list.items").Array()
	require.Len(s.T(), items, 1)
	assert.Equal(s.T(), job.ID.String(), items[0].Get("id").String())
	assert.Equal(s.T(), "PENDING", items[0].Get("status").String())
	assert.Equal(s.T(), "MANUAL", items[0].Get("trigger").String())

	// Phase 4: Add job logs and verify they appear
	_, err = s.Env.JobService.AddJobLog(ctx, job.ID, "INFO", "UPLOAD", "/test/file.txt", 1024)
	require.NoError(s.T(), err)

	// Query task with latestJob
	taskQuery := `
        query($id: ID!) {
            task {
                get(id: $id) {
                    id
                    name
                    latestJob {
                        id
                        status
                    }
                }
            }
        }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), taskQuery, map[string]interface{}{
		"id": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)
	latestJob := gjson.Get(string(resp.Data), "task.get.latestJob")
	assert.True(s.T(), latestJob.Exists())
	assert.Equal(s.T(), job.ID.String(), latestJob.Get("id").String())

	// Phase 5: Update the task
	updateMutation := `
        mutation($id: ID!, $input: UpdateTaskInput!) {
            task {
                update(id: $id, input: $input) {
                    id
                    name
                    sourcePath
                }
            }
        }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), updateMutation, map[string]interface{}{
		"id": task.ID.String(),
		"input": map[string]interface{}{
			"name":       "updated-sync-task",
			"sourcePath": "/new/source/path",
		},
	})
	require.Empty(s.T(), resp.Errors)
	assert.Equal(s.T(), "updated-sync-task", gjson.Get(string(resp.Data), "task.update.name").String())

	// Phase 6: Delete the task
	deleteTaskMutation := `
        mutation($id: ID!) {
            task {
                delete(id: $id) {
                    id
                }
            }
        }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), deleteTaskMutation, map[string]interface{}{
		"id": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)
	assert.Equal(s.T(), task.ID.String(), gjson.Get(string(resp.Data), "task.delete.id").String())

	// Verify task is deleted
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), taskQuery, map[string]interface{}{
		"id": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)
	taskGet := gjson.Get(string(resp.Data), "task.get")
	assert.True(s.T(), taskGet.Type == gjson.Null || !taskGet.Exists())

	// Phase 7: Delete the connection (this should work since task is deleted)
	deleteConnMutation := `
        mutation($id: ID!) {
            connection {
                delete(id: $id) {
                    id
                }
            }
        }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), deleteConnMutation, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)
	assert.Equal(s.T(), connID.String(), gjson.Get(string(resp.Data), "connection.delete.id").String())
}

// TestUserJourney_MultipleConnections simulates managing multiple connections
// and their associated tasks, ensuring data isolation between different connections.
func (s *IntegrationTestSuite) TestUserJourney_MultipleConnections() {
	// Create multiple connections
	conn1 := s.Env.CreateTestConnection(s.T(), "connection-1")
	conn2 := s.Env.CreateTestConnection(s.T(), "connection-2")
	conn3 := s.Env.CreateTestConnection(s.T(), "connection-3")

	// Create tasks for each connection
	task1 := s.Env.CreateTestTask(s.T(), "task-1", conn1)
	task2 := s.Env.CreateTestTask(s.T(), "task-2", conn2)
	task3 := s.Env.CreateTestTask(s.T(), "task-3", conn3)

	// Create jobs for each task
	ctx := context.Background()
	job1, _ := s.Env.JobService.CreateJob(ctx, task1.ID, "MANUAL")
	job2, _ := s.Env.JobService.CreateJob(ctx, task2.ID, "SCHEDULE")
	job3, _ := s.Env.JobService.CreateJob(ctx, task3.ID, "REALTIME")

	// Query all jobs and verify they're correctly associated
	jobQuery := `
        query {
            job {
                list {
                    items {
                        id
                        status
                        trigger
                        task {
                            id
                            name
                            connection {
                                id
                                name
                            }
                        }
                    }
                }
            }
        }
    `
	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: jobQuery})
	require.Empty(s.T(), resp.Errors)
	items := gjson.Get(string(resp.Data), "job.list.items").Array()
	require.Len(s.T(), items, 3)

	// Verify job associations
	jobMap := make(map[string]*gjson.Result)
	for i := range items {
		jobMap[items[i].Get("id").String()] = &items[i]
	}

	assert.Equal(s.T(), "PENDING", jobMap[job1.ID.String()].Get("status").String())
	assert.Equal(s.T(), "MANUAL", jobMap[job1.ID.String()].Get("trigger").String())
	assert.Equal(s.T(), conn1.String(), jobMap[job1.ID.String()].Get("task.connection.id").String())

	assert.Equal(s.T(), "SCHEDULE", jobMap[job2.ID.String()].Get("trigger").String())
	assert.Equal(s.T(), conn2.String(), jobMap[job2.ID.String()].Get("task.connection.id").String())

	assert.Equal(s.T(), "REALTIME", jobMap[job3.ID.String()].Get("trigger").String())
	assert.Equal(s.T(), conn3.String(), jobMap[job3.ID.String()].Get("task.connection.id").String())
}

// TestUserJourney_FileBrowsingAndSync tests file browsing through a connection
// and verifies the integration between file operations and sync tasks.
func (s *IntegrationTestSuite) TestUserJourney_FileBrowsingAndSync() {
	// Create test files
	sourceDir := filepath.Join(s.testDir, "source")
	testFile := filepath.Join(sourceDir, "test.txt")
	require.NoError(s.T(), os.WriteFile(testFile, []byte("test content"), 0644))

	// Create connection and task
	connID := s.Env.CreateTestConnection(s.T(), "file-browse-conn")
	task := s.Env.CreateTestTask(s.T(), "file-sync-task", connID)

	// Verify task paths are set correctly
	taskQuery := `
        query($id: ID!) {
            task {
                get(id: $id) {
                    id
                    sourcePath
                    remotePath
                    connection {
                        id
                        name
                    }
                }
            }
        }
    `
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), taskQuery, map[string]interface{}{
		"id": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)
	taskData := gjson.Get(string(resp.Data), "task.get")
	assert.NotEmpty(s.T(), taskData.Get("sourcePath").String())
	assert.NotEmpty(s.T(), taskData.Get("remotePath").String())
	assert.Equal(s.T(), connID.String(), taskData.Get("connection.id").String())

	// Run the task and create a job
	ctx := context.Background()
	job, err := s.Env.JobService.CreateJob(ctx, task.ID, "MANUAL")
	require.NoError(s.T(), err)

	// Add file transfer logs
	_, err = s.Env.JobService.AddJobLog(ctx, job.ID, "INFO", "UPLOAD", testFile, 12)
	require.NoError(s.T(), err)

	// Query job logs to verify file transfer was recorded
	logQuery := `
        query($connectionId: ID!, $jobId: ID) {
            log {
                list(connectionId: $connectionId, jobId: $jobId) {
                    items {
                        id
                        path
                        size
                        what
                    }
                    totalCount
                }
            }
        }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), logQuery, map[string]interface{}{
		"connectionId": connID.String(),
		"jobId":        job.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)
	logItems := gjson.Get(string(resp.Data), "log.list.items").Array()
	require.Len(s.T(), logItems, 1)
	assert.Equal(s.T(), testFile, logItems[0].Get("path").String())
	assert.Equal(s.T(), int64(12), logItems[0].Get("size").Int())
	assert.Equal(s.T(), "UPLOAD", logItems[0].Get("what").String())
}

// TestUserJourney_CompleteFlow tests the complete user journey from start to finish.
// This simulates a real user's full workflow through the application.
func (s *IntegrationTestSuite) TestUserJourney_CompleteFlow() {
	// Phase 1: Create a connection
	createConnMutation := `
        mutation($input: CreateConnectionInput!) {
            connection { create(input: $input) { id name type } }
        }
    `
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createConnMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name":   "complete-flow-storage",
			"type":   "local",
			"config": map[string]interface{}{},
		},
	})
	require.Empty(s.T(), resp.Errors, "Should create connection")
	connID := gjson.Get(string(resp.Data), "connection.create.id").String()

	// Phase 2: Browse local files
	browseQuery := `
        query($path: String!) { file { local(path: $path) { name isDir } } }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), browseQuery, map[string]interface{}{
		"path": s.testDir,
	})
	require.Empty(s.T(), resp.Errors, "Should browse local files")

	// Phase 3: Create a task
	createTaskMutation := `
        mutation($input: CreateTaskInput!) {
            task { create(input: $input) { id name direction } }
        }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), createTaskMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name":         "complete-flow-task",
			"sourcePath":   filepath.Join(s.testDir, "source"),
			"connectionId": connID,
			"remotePath":   filepath.Join(s.testDir, "remote"),
			"direction":    "UPLOAD",
		},
	})
	require.Empty(s.T(), resp.Errors, "Should create task")
	taskID := gjson.Get(string(resp.Data), "task.create.id").String()

	// Phase 4: List tasks with connections and latest job
	listTasksQuery := `
        query {
            task {
                list {
                    items {
                        id
                        name
                        connection { id name }
                        latestJob { id status }
                    }
                }
            }
        }
    `
	resp = s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: listTasksQuery})
	require.Empty(s.T(), resp.Errors, "Should list tasks")
	tasks := gjson.Get(string(resp.Data), "task.list.items").Array()
	assert.Len(s.T(), tasks, 1)
	assert.Equal(s.T(), connID, tasks[0].Get("connection.id").String())

	// Phase 5: Create a job and add logs
	id, err := uuid.Parse(taskID)
	require.NoError(s.T(), err)
	job, err := s.Env.JobService.CreateJob(s.T().Context(), id, "MANUAL")
	require.NoError(s.T(), err)
	_, err = s.Env.JobService.AddJobLog(s.T().Context(), job.ID, "INFO", "UPLOAD", "/test.txt", 100)
	require.NoError(s.T(), err)

	// Phase 6: Query jobs with task reference
	listJobsQuery := `
        query($taskId: ID) {
            job {
                list(taskId: $taskId) {
                    items {
                        id
                        status
                        task { id name }
                    }
                    totalCount
                }
            }
        }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), listJobsQuery, map[string]interface{}{
		"taskId": taskID,
	})
	require.Empty(s.T(), resp.Errors, "Should list jobs")
	jobs := gjson.Get(string(resp.Data), "job.list.items").Array()
	assert.Len(s.T(), jobs, 1)
	assert.Equal(s.T(), taskID, jobs[0].Get("task.id").String())

	// Phase 7: Query logs
	listLogsQuery := `
        query($connectionId: ID!, $jobId: ID) {
            log {
                list(connectionId: $connectionId, jobId: $jobId) {
                    items { id level path }
                    totalCount
                }
            }
        }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), listLogsQuery, map[string]interface{}{
		"connectionId": connID,
		"jobId":        job.ID.String(),
	})
	require.Empty(s.T(), resp.Errors, "Should list logs")
	logs := gjson.Get(string(resp.Data), "log.list.items").Array()
	assert.Len(s.T(), logs, 1)

	// Phase 8: Update task
	updateTaskMutation := `
        mutation($id: ID!, $input: UpdateTaskInput!) {
            task { update(id: $id, input: $input) { id name direction } }
        }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), updateTaskMutation, map[string]interface{}{
		"id": taskID,
		"input": map[string]interface{}{
			"direction": "BIDIRECTIONAL",
		},
	})
	require.Empty(s.T(), resp.Errors, "Should update task")
	assert.Equal(s.T(), "BIDIRECTIONAL", gjson.Get(string(resp.Data), "task.update.direction").String())

	// Phase 9: Cleanup - delete task first (to avoid FK constraint)
	deleteTaskMutation := `
        mutation($id: ID!) { task { delete(id: $id) { id } } }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), deleteTaskMutation, map[string]interface{}{
		"id": taskID,
	})
	require.Empty(s.T(), resp.Errors, "Should delete task")

	// Phase 10: Cleanup - delete connection
	deleteConnMutation := `
        mutation($id: ID!) { connection { delete(id: $id) { id } } }
    `
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), deleteConnMutation, map[string]interface{}{
		"id": connID,
	})
	require.Empty(s.T(), resp.Errors, "Should delete connection")
}
