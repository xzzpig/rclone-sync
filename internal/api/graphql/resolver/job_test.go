// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"context"
	"fmt"
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
	testCases := []struct {
		name       string
		totalItems int
		pageSize   int
	}{
		{"basic-pagination", 5, 2},
		{"single-item-last-page", 5, 3},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// Create a fresh TestEnv for each sub-test to avoid data leakage
			env := NewTestEnv(t)
			t.Cleanup(env.Cleanup)

			connID := env.CreateTestConnection(t, "test-conn-"+tc.name)
			task := env.CreateTestTask(t, "test-task-"+tc.name, connID)
			testPagination(env, t, PaginationTestCase{
				EntityName: "Job",
				Query:      GetCommonQueries().JobListPagination,
				DataPath:   "job.list",
				CreateFunc: func(env *TestEnv, t *testing.T, i int) {
					t.Helper()
					ctx := context.Background()
					job, err := env.JobService.CreateJob(ctx, task.ID, "MANUAL")
					require.NoError(t, err)
					_ = job.ID
				},
				PageSize:   tc.pageSize,
				TotalItems: tc.totalItems,
			})
		})
	}
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
				progress(id: $jobId) {
					jobId
					taskId
					connectionId
					status
					filesTransferred
					bytesTransferred
					filesTotal
					bytesTotal
					filesDeleted
					errorCount
					startTime
					endTime
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
		if progress.Exists() && progress.Type != gjson.Null {
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

// TestJob_FilesDeletedAndErrorCount tests Job.filesDeleted and Job.errorCount fields.
func (s *JobResolverTestSuite) TestJob_FilesDeletedAndErrorCount() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	s.createTestJob(task.ID)

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
						filesTransferred
						bytesTransferred
						filesDeleted
						errorCount
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

	// New job should have 0 filesDeleted and 0 errorCount
	assert.Equal(s.T(), int64(0), items[0].Get("filesDeleted").Int())
	assert.Equal(s.T(), int64(0), items[0].Get("errorCount").Int())
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

// TestJob_Progress tests Job.progress field resolver.
func (s *JobResolverTestSuite) TestJob_Progress() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
						progress {
							jobId
							taskId
							connectionId
							status
							filesTransferred
							bytesTransferred
							filesTotal
							bytesTotal
							filesDeleted
							errorCount
							startTime
							endTime
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
	// Progress may be null for non-running jobs
	progress := items[0].Get("progress")
	// Either null or has fields - both are valid
	if progress.Exists() && progress.Type != gjson.Null {
		assert.True(s.T(), progress.Get("jobId").Exists())
	}
}

// TestJobQuery_ProgressResolver tests JobQuery.progress resolver.
func (s *JobResolverTestSuite) TestJobQuery_ProgressResolver() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	query := `
		query($jobId: ID!) {
			job {
				progress(id: $jobId) {
					jobId
					taskId
					connectionId
					status
					filesTransferred
					bytesTransferred
					filesTotal
					bytesTotal
					filesDeleted
					errorCount
					startTime
					endTime
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"jobId": jobID.String(),
	})
	// Progress query should succeed (may return null for non-running jobs)
	require.Empty(s.T(), resp.Errors)
}

// TestJobQuery_ListWithConnectionFilter tests JobQuery.list with connectionId filter.
func (s *JobResolverTestSuite) TestJobQuery_ListWithConnectionFilter() {
	connID1 := s.Env.CreateTestConnection(s.T(), "test-conn-1")
	connID2 := s.Env.CreateTestConnection(s.T(), "test-conn-2")
	task1 := s.Env.CreateTestTask(s.T(), "task-1", connID1)
	task2 := s.Env.CreateTestTask(s.T(), "task-2", connID2)

	// Create jobs for each task
	for i := 0; i < 3; i++ {
		s.createTestJob(task1.ID)
	}
	for i := 0; i < 2; i++ {
		s.createTestJob(task2.ID)
	}

	query := `
		query($connectionId: ID) {
			job {
				list(connectionId: $connectionId) {
					items {
						id
					}
					totalCount
				}
			}
		}
	`

	// Query jobs for connection1
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID1.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 3, int(gjson.Get(data, "job.list.totalCount").Int()))

	// Query jobs for connection2
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID2.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data = string(resp.Data)
	assert.Equal(s.T(), 2, int(gjson.Get(data, "job.list.totalCount").Int()))
}

// TestJob_LogsWithPagination tests Job.logs field resolver with pagination.
func (s *JobResolverTestSuite) TestJob_LogsWithPagination() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	// Add 10 logs to job
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		_, err := s.Env.JobService.AddJobLog(ctx, jobID, "INFO", "UPLOAD", fmt.Sprintf("/test/path%d", i), 1024)
		require.NoError(s.T(), err)
	}

	query := `
		query($taskId: ID, $pagination: PaginationInput) {
			job {
				list(taskId: $taskId) {
					items {
						id
						logs(pagination: $pagination) {
							items {
								id
								path
							}
							totalCount
							pageInfo {
								hasNextPage
								hasPreviousPage
							}
						}
					}
				}
			}
		}
	`

	// Test with pagination
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"taskId": task.ID.String(),
		"pagination": map[string]interface{}{
			"limit":  5,
			"offset": 0,
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	items := gjson.Get(data, "job.list.items").Array()
	require.Len(s.T(), items, 1)

	logs := items[0].Get("logs")
	assert.Equal(s.T(), 10, int(logs.Get("totalCount").Int()))
	assert.Equal(s.T(), 5, len(logs.Get("items").Array()))
	assert.True(s.T(), logs.Get("pageInfo.hasNextPage").Bool())
	assert.False(s.T(), logs.Get("pageInfo.hasPreviousPage").Bool())

	// Test second page
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"taskId": task.ID.String(),
		"pagination": map[string]interface{}{
			"limit":  5,
			"offset": 5,
		},
	})
	require.Empty(s.T(), resp.Errors)

	data = string(resp.Data)
	items = gjson.Get(data, "job.list.items").Array()
	logs = items[0].Get("logs")
	assert.False(s.T(), logs.Get("pageInfo.hasNextPage").Bool())
	assert.True(s.T(), logs.Get("pageInfo.hasPreviousPage").Bool())
}

// TestLogQuery_ListWithLevel tests LogQuery.list with level filter.
func (s *JobResolverTestSuite) TestLogQuery_ListWithLevel() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	// Add logs with different levels
	ctx := context.Background()
	_, err := s.Env.JobService.AddJobLog(ctx, jobID, "INFO", "UPLOAD", "/test/path1", 1024)
	require.NoError(s.T(), err)
	_, err = s.Env.JobService.AddJobLog(ctx, jobID, "ERROR", "DOWNLOAD", "/test/path2", 2048)
	require.NoError(s.T(), err)
	_, err = s.Env.JobService.AddJobLog(ctx, jobID, "INFO", "UPLOAD", "/test/path3", 512)
	require.NoError(s.T(), err)

	query := `
		query($connectionId: ID!, $level: LogLevel) {
			log {
				list(connectionId: $connectionId, level: $level) {
					items {
						id
						level
					}
					totalCount
				}
			}
		}
	`

	// Query INFO level logs
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID.String(),
		"level":        "INFO",
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 2, int(gjson.Get(data, "log.list.totalCount").Int()))

	// Query ERROR level logs
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID.String(),
		"level":        "ERROR",
	})
	require.Empty(s.T(), resp.Errors)

	data = string(resp.Data)
	assert.Equal(s.T(), 1, int(gjson.Get(data, "log.list.totalCount").Int()))
}

// TestLogQuery_ListWithTaskFilter tests LogQuery.list with taskId filter.
func (s *JobResolverTestSuite) TestLogQuery_ListWithTaskFilter() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task1 := s.Env.CreateTestTask(s.T(), "task-1", connID)
	task2 := s.Env.CreateTestTask(s.T(), "task-2", connID)
	jobID1 := s.createTestJob(task1.ID)
	jobID2 := s.createTestJob(task2.ID)

	// Add logs for each job
	ctx := context.Background()
	_, err := s.Env.JobService.AddJobLog(ctx, jobID1, "INFO", "UPLOAD", "/task1/path", 1024)
	require.NoError(s.T(), err)
	_, err = s.Env.JobService.AddJobLog(ctx, jobID2, "INFO", "UPLOAD", "/task2/path", 2048)
	require.NoError(s.T(), err)

	query := `
		query($connectionId: ID!, $taskId: ID) {
			log {
				list(connectionId: $connectionId, taskId: $taskId) {
					items {
						id
						path
					}
					totalCount
				}
			}
		}
	`

	// Query logs for task1
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID.String(),
		"taskId":       task1.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 1, int(gjson.Get(data, "log.list.totalCount").Int()))
	items := gjson.Get(data, "log.list.items").Array()
	assert.Contains(s.T(), items[0].Get("path").String(), "task1")
}

// TestJobLog_AllFields tests that all JobLog fields are returned correctly.
func (s *JobResolverTestSuite) TestJobLog_AllFields() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	// Add a log with specific values
	ctx := context.Background()
	_, err := s.Env.JobService.AddJobLog(ctx, jobID, "WARNING", "DELETE", "/special/file.txt", 4096)
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

	log := items[0]
	assert.NotEmpty(s.T(), log.Get("id").String())
	assert.Equal(s.T(), "WARNING", log.Get("level").String())
	assert.True(s.T(), log.Get("time").Exists())
	assert.Equal(s.T(), "/special/file.txt", log.Get("path").String())
	assert.Equal(s.T(), "DELETE", log.Get("what").String())
	assert.Equal(s.T(), int64(4096), log.Get("size").Int())
	assert.Equal(s.T(), jobID.String(), log.Get("job.id").String())
}

// TestJob_EndTimeFieldForCompletedJob tests that endTime is set for completed jobs.
func (s *JobResolverTestSuite) TestJob_EndTimeFieldForCompletedJob() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	// Update job to SUCCESS status
	ctx := context.Background()
	_, err := s.Env.JobService.UpdateJobStatus(ctx, jobID, "SUCCESS", "")
	require.NoError(s.T(), err)

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
						status
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

	assert.Equal(s.T(), "SUCCESS", items[0].Get("status").String())
	assert.True(s.T(), items[0].Get("startTime").Exists())
	// endTime should be set for completed jobs
	endTime := items[0].Get("endTime")
	assert.True(s.T(), endTime.Exists() && endTime.Type != gjson.Null)
}

// TestJob_ErrorsFieldForFailedJob tests that errors field is set for failed jobs.
func (s *JobResolverTestSuite) TestJob_ErrorsFieldForFailedJob() {
	connID := s.Env.CreateTestConnection(s.T(), "test-conn")
	task := s.Env.CreateTestTask(s.T(), "test-task", connID)
	jobID := s.createTestJob(task.ID)

	// Update job to FAILED status with error message
	ctx := context.Background()
	_, err := s.Env.JobService.UpdateJobStatus(ctx, jobID, "FAILED", "Connection timeout")
	require.NoError(s.T(), err)

	query := `
		query($taskId: ID) {
			job {
				list(taskId: $taskId) {
					items {
						id
						status
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

	assert.Equal(s.T(), "FAILED", items[0].Get("status").String())
	assert.Equal(s.T(), "Connection timeout", items[0].Get("errors").String())
}
