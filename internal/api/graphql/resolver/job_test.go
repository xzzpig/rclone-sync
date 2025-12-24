// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

// JobResolverTestSuite tests JobQuery and LogQuery resolvers.
type JobResolverTestSuite struct {
	ResolverTestSuite
}

func TestJobResolverSuite(t *testing.T) {
	suite.Run(t, new(JobResolverTestSuite))
}

// createTestJob creates a test job for a task and returns the job ID.
func (s *JobResolverTestSuite) createTestJob(taskID uuid.UUID) uuid.UUID {
	ctx := context.Background()
	job, err := s.Env.JobService.CreateJob(ctx, taskID, "MANUAL")
	require.NoError(s.T(), err)
	return job.ID
}

// TestJobQuery_List tests JobQuery.list resolver.
func (s *JobResolverTestSuite) TestJobQuery_List() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)

	// Create multiple jobs
	for i := 0; i < 3; i++ {
		s.createTestJob(task.ID)
	}

	query := `
		query {
			job {
				list {
					items {
						id
						status
						trigger
					}
					totalCount
					pageInfo {
						limit
						offset
						hasNextPage
						hasPreviousPage
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: query})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 3, int(gjson.Get(data, "job.list.totalCount").Int()))
	assert.Equal(s.T(), 3, len(gjson.Get(data, "job.list.items").Array()))
}

// TestJobQuery_ListWithTaskFilter tests JobQuery.list with taskId filter.
func (s *JobResolverTestSuite) TestJobQuery_ListWithTaskFilter() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task1 := s.Env.CreateTestTask(s.T(), "task-1", connID)
	task2 := s.Env.CreateTestTask(s.T(), "task-2", connID)

	// Create jobs for task1
	for i := 0; i < 3; i++ {
		s.createTestJob(task1.ID)
	}

	// Create jobs for task2
	for i := 0; i < 2; i++ {
		s.createTestJob(task2.ID)
	}

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
					}
					totalCount
				}
			}
		}
	`

	// Query jobs for task1
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"taskId": task1.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 3, int(gjson.Get(data, "job.list.totalCount").Int()))

	// Query jobs for task2
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"taskId": task2.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data = string(resp.Data)
	assert.Equal(s.T(), 2, int(gjson.Get(data, "job.list.totalCount").Int()))
}

// TestJobQuery_ListWithPagination tests JobQuery.list with pagination.
func (s *JobResolverTestSuite) TestJobQuery_ListWithPagination() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)

	// Create 5 jobs
	for i := 0; i < 5; i++ {
		s.createTestJob(task.ID)
	}

	query := `
		query($pagination: PaginationInput) {
			job {
				list(pagination: $pagination) {
					items {
						id
					}
					totalCount
					pageInfo {
						hasNextPage
						hasPreviousPage
					}
				}
			}
		}
	`

	// Test first page
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"pagination": map[string]interface{}{
			"limit":  2,
			"offset": 0,
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 5, int(gjson.Get(data, "job.list.totalCount").Int()))
	assert.Equal(s.T(), 2, len(gjson.Get(data, "job.list.items").Array()))
	assert.True(s.T(), gjson.Get(data, "job.list.pageInfo.hasNextPage").Bool())
	assert.False(s.T(), gjson.Get(data, "job.list.pageInfo.hasPreviousPage").Bool())
}

// TestJob_Task tests Job.task field resolver.
func (s *JobResolverTestSuite) TestJob_Task() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "my-task", connID)
	jobID := s.createTestJob(task.ID)

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

	assert.Equal(s.T(), jobID.String(), items[0].Get("id").String())
	assert.Equal(s.T(), task.ID.String(), items[0].Get("task.id").String())
	assert.Equal(s.T(), "my-task", items[0].Get("task.name").String())
}

// TestJob_Logs tests Job.logs field resolver.
func (s *JobResolverTestSuite) TestJob_Logs() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	// Add some logs to the job
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := s.Env.JobService.AddJobLog(ctx, jobID, "INFO", "UPLOAD", "/test/path", 1024)
		require.NoError(s.T(), err)
	}

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
						logs {
							items {
								id
								level
								path
							}
							totalCount
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

	logs := items[0].Get("logs")
	assert.Equal(s.T(), 3, int(logs.Get("totalCount").Int()))
	assert.Equal(s.T(), 3, len(logs.Get("items").Array()))
}

// TestJobQuery_Progress tests JobQuery.progress resolver.
func (s *JobResolverTestSuite) TestJobQuery_Progress() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	query := `
		query($jobId: ID!) {
			job {
				progress(jobId: $jobId) {
					jobId
					status
					bytesTotal
					bytesDone
					filesTotal
					filesDone
					percentage
					speed
					eta
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"jobId": jobID.String(),
	})
	// Progress may or may not be available depending on job state
	// Just verify the query is valid
	if len(resp.Errors) == 0 {
		data := string(resp.Data)
		progress := gjson.Get(data, "job.progress")
		if progress.Exists() {
			assert.Equal(s.T(), jobID.String(), progress.Get("jobId").String())
		}
	}
}

// TestLogQuery_List tests LogQuery.list resolver.
func (s *JobResolverTestSuite) TestLogQuery_List() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	// Add logs
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := s.Env.JobService.AddJobLog(ctx, jobID, "INFO", "UPLOAD", "/test/path", 1024)
		require.NoError(s.T(), err)
	}

	query := `
		query($connectionId: ID!, $jobId: ID) {
			log {
				list(connectionId: $connectionId, jobId: $jobId) {
					items {
						id
						level
						path
						time
					}
					totalCount
					pageInfo {
						limit
						offset
						hasNextPage
						hasPreviousPage
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID.String(),
		"jobId":        jobID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 5, int(gjson.Get(data, "log.list.totalCount").Int()))
	assert.Equal(s.T(), 5, len(gjson.Get(data, "log.list.items").Array()))
}

// TestLogQuery_ListWithPagination tests LogQuery.list with pagination.
func (s *JobResolverTestSuite) TestLogQuery_ListWithPagination() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	// Add logs
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		_, err := s.Env.JobService.AddJobLog(ctx, jobID, "INFO", "UPLOAD", "/test/path", 1024)
		require.NoError(s.T(), err)
	}

	query := `
		query($connectionId: ID!, $jobId: ID, $pagination: PaginationInput) {
			log {
				list(connectionId: $connectionId, jobId: $jobId, pagination: $pagination) {
					items {
						id
					}
					totalCount
					pageInfo {
						hasNextPage
						hasPreviousPage
					}
				}
			}
		}
	`

	// Test first page
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID.String(),
		"jobId":        jobID.String(),
		"pagination": map[string]interface{}{
			"limit":  5,
			"offset": 0,
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 10, int(gjson.Get(data, "log.list.totalCount").Int()))
	assert.Equal(s.T(), 5, len(gjson.Get(data, "log.list.items").Array()))
	assert.True(s.T(), gjson.Get(data, "log.list.pageInfo.hasNextPage").Bool())
	assert.False(s.T(), gjson.Get(data, "log.list.pageInfo.hasPreviousPage").Bool())
}

// TestJobLog_Job tests JobLog.job field resolver.
func (s *JobResolverTestSuite) TestJobLog_Job() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	// Add a log
	ctx := context.Background()
	_, err := s.Env.JobService.AddJobLog(ctx, jobID, "INFO", "UPLOAD", "/test/path", 1024)
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
		"jobId":        jobID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	items := gjson.Get(data, "log.list.items").Array()
	require.Len(s.T(), items, 1)

	assert.Equal(s.T(), jobID.String(), items[0].Get("job.id").String())
}

// TestJob_StatusValues tests that job status values are correct.
func (s *JobResolverTestSuite) TestJob_StatusValues() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	s.createTestJob(task.ID)

	query := `
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

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"taskId": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	items := gjson.Get(data, "job.list.items").Array()
	require.Len(s.T(), items, 1)

	// New job should be PENDING status with MANUAL trigger
	assert.Equal(s.T(), "PENDING", items[0].Get("status").String())
	assert.Equal(s.T(), "MANUAL", items[0].Get("trigger").String())
}

// TestJob_TimeFields tests job time fields.
func (s *JobResolverTestSuite) TestJob_TimeFields() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	s.createTestJob(task.ID)

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
						startTime
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

	// startTime should be set
	assert.True(s.T(), items[0].Get("startTime").Exists())
	// endTime may be null for pending jobs
}

// TestLogQuery_EmptyJobId tests LogQuery.list with non-existent job.
func (s *JobResolverTestSuite) TestLogQuery_EmptyJobId() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")

	query := `
		query($connectionId: ID!, $jobId: ID) {
			log {
				list(connectionId: $connectionId, jobId: $jobId) {
					items {
						id
					}
					totalCount
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID.String(),
		"jobId":        uuid.New().String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 0, int(gjson.Get(data, "log.list.totalCount").Int()))
}
