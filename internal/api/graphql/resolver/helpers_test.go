// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
)

// HelpersTestSuite tests helper functions indirectly through GraphQL queries.
type HelpersTestSuite struct {
	ResolverTestSuite
}

func TestHelpersSuite(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}

// TestEntConnectionToModel tests connection model conversion.
func (s *HelpersTestSuite) TestEntConnectionToModel() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")

	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					name
					type
					createdAt
					updatedAt
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), connID.String(), gjson.Get(data, "connection.get.id").String())
	assert.Equal(s.T(), "test-conn", gjson.Get(data, "connection.get.name").String())
	assert.Equal(s.T(), "local", gjson.Get(data, "connection.get.type").String())
	assert.True(s.T(), gjson.Get(data, "connection.get.createdAt").Exists())
	assert.True(s.T(), gjson.Get(data, "connection.get.updatedAt").Exists())
}

// TestEntTaskToModel tests task model conversion.
func (s *HelpersTestSuite) TestEntTaskToModel() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)

	query := `
		query($id: ID!) {
			task {
				get(id: $id) {
					id
					name
					sourcePath
					remotePath
					direction
					schedule
					realtime
					createdAt
					updatedAt
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), task.ID.String(), gjson.Get(data, "task.get.id").String())
	assert.Equal(s.T(), "test-task", gjson.Get(data, "task.get.name").String())
	assert.Equal(s.T(), "/tmp/source", gjson.Get(data, "task.get.sourcePath").String())
	assert.Equal(s.T(), "/remote", gjson.Get(data, "task.get.remotePath").String())
	assert.Equal(s.T(), "UPLOAD", gjson.Get(data, "task.get.direction").String())
	assert.False(s.T(), gjson.Get(data, "task.get.realtime").Bool())
	assert.True(s.T(), gjson.Get(data, "task.get.createdAt").Exists())
	assert.True(s.T(), gjson.Get(data, "task.get.updatedAt").Exists())
}

// TestEntTaskToModelWithSchedule tests task model conversion with schedule.
func (s *HelpersTestSuite) TestEntTaskToModelWithSchedule() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")

	mutation := `
		mutation($input: CreateTaskInput!) {
			task {
				create(input: $input) {
					id
					name
					schedule
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name":         "scheduled-task",
			"sourcePath":   "/local",
			"connectionId": connID.String(),
			"remotePath":   "/remote",
			"direction":    "UPLOAD",
			"schedule":     "*/5 * * * *",
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), "scheduled-task", gjson.Get(data, "task.create.name").String())
	assert.Equal(s.T(), "*/5 * * * *", gjson.Get(data, "task.create.schedule").String())
}

// TestEntJobToModel tests job model conversion.
func (s *HelpersTestSuite) TestEntJobToModel() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)

	// Create a job
	job, err := s.Env.JobService.CreateJob(s.T().Context(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
						status
						trigger
						startTime
						endTime
						filesTransferred
						bytesTransferred
						errors
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"taskId": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	items := gjson.Get(data, "job.list.items").Array()
	require.Len(s.T(), items, 1)

	assert.Equal(s.T(), job.ID.String(), items[0].Get("id").String())
	assert.Equal(s.T(), "PENDING", items[0].Get("status").String())
	assert.Equal(s.T(), "MANUAL", items[0].Get("trigger").String())
	assert.True(s.T(), items[0].Get("startTime").Exists())
	// endTime should be null for pending jobs
	assert.True(s.T(), items[0].Get("endTime").Type == gjson.Null)
	assert.Equal(s.T(), int64(0), items[0].Get("filesTransferred").Int())
	assert.Equal(s.T(), int64(0), items[0].Get("bytesTransferred").Int())
}

// TestEntJobToModelWithErrors tests job model conversion with errors field.
func (s *HelpersTestSuite) TestEntJobToModelWithErrors() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)

	// Create a job and update it with error
	job, err := s.Env.JobService.CreateJob(s.T().Context(), task.ID, model.JobTriggerManual)
	require.NoError(s.T(), err)

	_, err = s.Env.JobService.UpdateJobStatus(s.T().Context(), job.ID, "FAILED", "Test error message")
	require.NoError(s.T(), err)

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
						status
						errors
						endTime
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"taskId": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	items := gjson.Get(data, "job.list.items").Array()
	require.Len(s.T(), items, 1)

	assert.Equal(s.T(), "FAILED", items[0].Get("status").String())
	assert.Equal(s.T(), "Test error message", items[0].Get("errors").String())
	assert.True(s.T(), items[0].Get("endTime").Exists())
	assert.False(s.T(), items[0].Get("endTime").Type == gjson.Null)
}

// TestEntJobToModelWithEndTime tests job model conversion with end time.
func (s *HelpersTestSuite) TestEntJobToModelWithEndTime() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)

	// Create a job and complete it
	job, err := s.Env.JobService.CreateJob(s.T().Context(), task.ID, model.JobTriggerSchedule)
	require.NoError(s.T(), err)

	_, err = s.Env.JobService.UpdateJobStatus(s.T().Context(), job.ID, "SUCCESS", "")
	require.NoError(s.T(), err)

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
						status
						trigger
						endTime
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"taskId": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	items := gjson.Get(data, "job.list.items").Array()
	require.Len(s.T(), items, 1)

	assert.Equal(s.T(), "SUCCESS", items[0].Get("status").String())
	assert.Equal(s.T(), "SCHEDULE", items[0].Get("trigger").String())
	// endTime should be set for completed jobs
	assert.True(s.T(), items[0].Get("endTime").Exists())
	assert.False(s.T(), items[0].Get("endTime").Type == gjson.Null)
}

// TestEntJobLogToModel tests job log model conversion.
func (s *HelpersTestSuite) TestEntJobLogToModel() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)

	// Create a job and add logs
	job, err := s.Env.JobService.CreateJob(s.T().Context(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	_, err = s.Env.JobService.AddJobLog(s.T().Context(), job.ID, "INFO", "UPLOAD", "/test/path/file.txt", 1024)
	require.NoError(s.T(), err)

	query := `
		query($connectionId: ID!, $jobId: ID) {
			log {
				list(connectionId: $connectionId, jobId: $jobId) {
					items {
						id
						level
						time
						path
						what
						size
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID.String(),
		"jobId":        job.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	items := gjson.Get(data, "log.list.items").Array()
	require.Len(s.T(), items, 1)

	assert.True(s.T(), items[0].Get("id").Exists())
	assert.Equal(s.T(), "INFO", items[0].Get("level").String())
	assert.True(s.T(), items[0].Get("time").Exists())
	assert.Equal(s.T(), "/test/path/file.txt", items[0].Get("path").String())
	assert.Equal(s.T(), "UPLOAD", items[0].Get("what").String())
	assert.Equal(s.T(), int64(1024), items[0].Get("size").Int())
}

// TestEntJobLogToModelWithJobReference tests that JobLog.job field works.
func (s *HelpersTestSuite) TestEntJobLogToModelWithJobReference() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)

	// Create a job and add logs
	job, err := s.Env.JobService.CreateJob(s.T().Context(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	_, err = s.Env.JobService.AddJobLog(s.T().Context(), job.ID, "ERROR", "DOWNLOAD", "/error/path", 0)
	require.NoError(s.T(), err)

	query := `
		query($connectionId: ID!, $jobId: ID) {
			log {
				list(connectionId: $connectionId, jobId: $jobId) {
					items {
						id
						level
						job {
							id
							status
						}
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID.String(),
		"jobId":        job.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	items := gjson.Get(data, "log.list.items").Array()
	require.Len(s.T(), items, 1)

	assert.Equal(s.T(), "ERROR", items[0].Get("level").String())
	assert.Equal(s.T(), job.ID.String(), items[0].Get("job.id").String())
	assert.Equal(s.T(), "PENDING", items[0].Get("job.status").String())
}

// TestJobTriggersConversion tests all job trigger types.
func (s *HelpersTestSuite) TestJobTriggersConversion() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)

	triggers := []model.JobTrigger{model.JobTriggerManual, model.JobTriggerSchedule, model.JobTriggerRealtime}

	for _, trigger := range triggers {
		job, err := s.Env.JobService.CreateJob(s.T().Context(), task.ID, trigger)
		require.NoError(s.T(), err)

		query := `
			query($taskId: ID) {
				job {
					list(taskId: $taskId) {
						items {
							id
							trigger
						}
					}
				}
			}
		`

		resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
			"taskId": task.ID.String(),
		})
		require.Empty(s.T(), resp.Errors)

		data := string(resp.Data)
		items := gjson.Get(data, "job.list.items").Array()

		// Find the job we just created
		found := false
		for _, item := range items {
			if item.Get("id").String() == job.ID.String() {
				assert.Equal(s.T(), string(trigger), item.Get("trigger").String())
				found = true
				break
			}
		}
		assert.True(s.T(), found, "Job with trigger %s should be found", trigger)
	}
}

// TestTimestampConversion tests that timestamps are properly converted.
func (s *HelpersTestSuite) TestTimestampConversion() {
	beforeCreate := time.Now().Add(-time.Second)
	connID := s.Env.CreateTestConnection(s.T(), "timestamp-test")
	afterCreate := time.Now().Add(time.Second)

	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					createdAt
					updatedAt
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	createdAtStr := gjson.Get(data, "connection.get.createdAt").String()
	updatedAtStr := gjson.Get(data, "connection.get.updatedAt").String()

	// Parse the timestamps
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtStr)
	require.NoError(s.T(), err)
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtStr)
	require.NoError(s.T(), err)

	assert.True(s.T(), createdAt.After(beforeCreate), "createdAt should be after beforeCreate")
	assert.True(s.T(), createdAt.Before(afterCreate), "createdAt should be before afterCreate")
	assert.True(s.T(), updatedAt.After(beforeCreate), "updatedAt should be after beforeCreate")
	assert.True(s.T(), updatedAt.Before(afterCreate), "updatedAt should be before afterCreate")
}

// TestConnectionIDInTask tests that connectionID is properly set in task model.
func (s *HelpersTestSuite) TestConnectionIDInTask() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-for-task")
	task := s.Env.CreateTestTask(s.T(), "task-with-conn-id", connID)

	query := `
		query($id: ID!) {
			task {
				get(id: $id) {
					id
					connection {
						id
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), connID.String(), gjson.Get(data, "task.get.connection.id").String())
}

// TestTaskIDInJob tests that taskID is properly set in job model.
func (s *HelpersTestSuite) TestTaskIDInJob() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-for-job")
	task := s.Env.CreateTestTask(s.T(), "task-for-job", connID)

	job, err := s.Env.JobService.CreateJob(s.T().Context(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
						task {
							id
							name
						}
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"taskId": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	items := gjson.Get(data, "job.list.items").Array()
	require.Len(s.T(), items, 1)

	assert.Equal(s.T(), job.ID.String(), items[0].Get("id").String())
	assert.Equal(s.T(), task.ID.String(), items[0].Get("task.id").String())
	assert.Equal(s.T(), "task-for-job", items[0].Get("task.name").String())
}

// TestJobIDInJobLog tests that jobID is properly set in job log model.
func (s *HelpersTestSuite) TestJobIDInJobLog() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-for-log")
	task := s.Env.CreateTestTask(s.T(), "task-for-log", connID)

	job, err := s.Env.JobService.CreateJob(s.T().Context(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	_, err = s.Env.JobService.AddJobLog(s.T().Context(), job.ID, "INFO", "UPLOAD", "/log/path", 512)
	require.NoError(s.T(), err)

	query := `
		query($connectionId: ID!, $jobId: ID) {
			log {
				list(connectionId: $connectionId, jobId: $jobId) {
					items {
						id
						job {
							id
							status
						}
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID.String(),
		"jobId":        job.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	items := gjson.Get(data, "log.list.items").Array()
	require.Len(s.T(), items, 1)

	assert.Equal(s.T(), job.ID.String(), items[0].Get("job.id").String())
}

// TestEmptyScheduleConversion tests that empty schedule is converted to nil.
func (s *HelpersTestSuite) TestEmptyScheduleConversion() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-no-schedule")
	task := s.Env.CreateTestTask(s.T(), "task-no-schedule", connID)

	query := `
		query($id: ID!) {
			task {
				get(id: $id) {
					id
					schedule
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	schedule := gjson.Get(data, "task.get.schedule")
	// Schedule should be null when empty
	assert.True(s.T(), schedule.Type == gjson.Null || !schedule.Exists() || schedule.String() == "",
		"Empty schedule should be null or empty")
}

// TestNonExistentIDs tests handling of non-existent IDs.
func (s *HelpersTestSuite) TestNonExistentIDs() {
	nonExistentID := uuid.New().String()

	// Test connection
	connQuery := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
				}
			}
		}
	`
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), connQuery, map[string]interface{}{
		"id": nonExistentID,
	})
	require.Empty(s.T(), resp.Errors)
	assert.True(s.T(), gjson.Get(string(resp.Data), "connection.get").Type == gjson.Null)

	// Test task
	taskQuery := `
		query($id: ID!) {
			task {
				get(id: $id) {
					id
				}
			}
		}
	`
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), taskQuery, map[string]interface{}{
		"id": nonExistentID,
	})
	require.Empty(s.T(), resp.Errors)
	assert.True(s.T(), gjson.Get(string(resp.Data), "task.get").Type == gjson.Null)
}
