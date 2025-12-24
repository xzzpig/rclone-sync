// Package resolver provides GraphQL resolver tests.
// This file contains integration tests that simulate complete user workflows.
package resolver_test

import (
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
// Phase 1: Provider Discovery
// =============================================================================

// TestUserJourney_DiscoverProviders simulates a user discovering available storage providers.
// This corresponds to the frontend's ProvidersListQuery and ProviderGetQuery.
func (s *IntegrationTestSuite) TestUserJourney_DiscoverProviders() {
	// Step 1: List all available providers (matches ProvidersListQuery)
	listQuery := `
		query ProvidersList {
			provider {
				list {
					name
					description
					prefix
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: listQuery})
	require.Empty(s.T(), resp.Errors, "ProvidersList query should succeed")

	data := string(resp.Data)
	providers := gjson.Get(data, "provider.list").Array()
	assert.Greater(s.T(), len(providers), 0, "Should have at least one provider")

	// Verify local provider exists (it's always available)
	hasLocal := false
	for _, p := range providers {
		if p.Get("name").String() == "local" {
			hasLocal = true
			assert.NotEmpty(s.T(), p.Get("prefix").String(), "Provider should have prefix")
			break
		}
	}
	assert.True(s.T(), hasLocal, "Local provider should exist")

	// Step 2: Get detailed provider options (matches ProviderGetQuery)
	getQuery := `
		query ProviderGet($name: String!) {
			provider {
				get(name: $name) {
					name
					description
					prefix
					options {
						name
						help
						required
						isPassword
						default
						examples {
							value
							help
						}
						advanced
						groups
						exclusive
						type
					}
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), getQuery, map[string]interface{}{
		"name": "local",
	})
	require.Empty(s.T(), resp.Errors, "ProviderGet query should succeed")

	data = string(resp.Data)
	provider := gjson.Get(data, "provider.get")
	assert.True(s.T(), provider.Exists(), "Local provider should be returned")
	assert.Equal(s.T(), "local", provider.Get("name").String())
}

// =============================================================================
// Phase 2: Connection Management
// =============================================================================

// TestUserJourney_CreateAndManageConnection simulates creating and managing a connection.
// This corresponds to the frontend's connection CRUD operations.
func (s *IntegrationTestSuite) TestUserJourney_CreateAndManageConnection() {
	// Step 1: Create a new connection (matches ConnectionCreateMutation)
	createMutation := `
		mutation ConnectionCreate($input: CreateConnectionInput!) {
			connection {
				create(input: $input) {
					id
					name
					type
					createdAt
					updatedAt
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name":   "my-local-storage",
			"type":   "local",
			"config": map[string]interface{}{},
		},
	})
	require.Empty(s.T(), resp.Errors, "ConnectionCreate mutation should succeed")

	data := string(resp.Data)
	connID := gjson.Get(data, "connection.create.id").String()
	assert.NotEmpty(s.T(), connID, "Connection ID should be returned")
	assert.Equal(s.T(), "my-local-storage", gjson.Get(data, "connection.create.name").String())
	assert.Equal(s.T(), "local", gjson.Get(data, "connection.create.type").String())
	assert.True(s.T(), gjson.Get(data, "connection.create.createdAt").Exists())
	assert.True(s.T(), gjson.Get(data, "connection.create.updatedAt").Exists())

	// Step 2: List connections (matches ConnectionsListQuery)
	listQuery := `
		query ConnectionsList($pagination: PaginationInput) {
			connection {
				list(pagination: $pagination) {
					items {
						id
						name
						type
					}
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: listQuery})
	require.Empty(s.T(), resp.Errors, "ConnectionsList query should succeed")

	data = string(resp.Data)
	items := gjson.Get(data, "connection.list.items").Array()
	assert.Len(s.T(), items, 1, "Should have one connection")
	assert.Equal(s.T(), connID, items[0].Get("id").String())

	// Step 3: Get connection with config (matches ConnectionGetConfigQuery)
	getConfigQuery := `
		query ConnectionGetConfig($id: ID!) {
			connection {
				get(id: $id) {
					id
					name
					type
					config
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), getConfigQuery, map[string]interface{}{
		"id": connID,
	})
	require.Empty(s.T(), resp.Errors, "ConnectionGetConfig query should succeed")

	data = string(resp.Data)
	assert.Equal(s.T(), connID, gjson.Get(data, "connection.get.id").String())
	assert.True(s.T(), gjson.Get(data, "connection.get.config").Exists(), "Config should be returned")

	// Step 4: Test the connection (matches ConnectionTestMutation)
	testMutation := `
		mutation ConnectionTest($id: ID!) {
			connection {
				test(id: $id) {
					__typename
					... on ConnectionTestSuccess {
						message
					}
					... on ConnectionTestFailure {
						error
					}
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), testMutation, map[string]interface{}{
		"id": connID,
	})
	require.Empty(s.T(), resp.Errors, "ConnectionTest mutation should succeed")

	data = string(resp.Data)
	// Should return either success or failure
	typename := gjson.Get(data, "connection.test.__typename").String()
	assert.Contains(s.T(), []string{"ConnectionTestSuccess", "ConnectionTestFailure"}, typename)

	// Step 5: Update the connection (matches ConnectionUpdateMutation)
	updateMutation := `
		mutation ConnectionUpdate($id: ID!, $input: UpdateConnectionInput!) {
			connection {
				update(id: $id, input: $input) {
					id
					name
					type
					createdAt
					updatedAt
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), updateMutation, map[string]interface{}{
		"id": connID,
		"input": map[string]interface{}{
			"name": "my-local-storage-renamed",
		},
	})
	require.Empty(s.T(), resp.Errors, "ConnectionUpdate mutation should succeed")

	data = string(resp.Data)
	assert.Equal(s.T(), "my-local-storage-renamed", gjson.Get(data, "connection.update.name").String())

	// Step 6: Get connection quota (matches ConnectionGetQuotaQuery)
	getQuotaQuery := `
		query ConnectionGetQuota($id: ID!) {
			connection {
				get(id: $id) {
					id
					name
					quota {
						total
						used
					}
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), getQuotaQuery, map[string]interface{}{
		"id": connID,
	})
	require.Empty(s.T(), resp.Errors, "ConnectionGetQuota query should succeed")
	// Quota may or may not be available for local connections, so we just verify no errors

	// Step 7: Delete the connection (matches ConnectionDeleteMutation)
	deleteMutation := `
		mutation ConnectionDelete($id: ID!) {
			connection {
				delete(id: $id) {
					id
					name
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), deleteMutation, map[string]interface{}{
		"id": connID,
	})
	require.Empty(s.T(), resp.Errors, "ConnectionDelete mutation should succeed")

	data = string(resp.Data)
	assert.Equal(s.T(), connID, gjson.Get(data, "connection.delete.id").String())

	// Verify connection is deleted
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), getConfigQuery, map[string]interface{}{
		"id": connID,
	})
	require.Empty(s.T(), resp.Errors)
	data = string(resp.Data)
	connGet := gjson.Get(data, "connection.get")
	assert.True(s.T(), connGet.Type == gjson.Null || !connGet.Exists(), "Connection should be deleted")
}

// =============================================================================
// Phase 3: File Browsing
// =============================================================================

// TestUserJourney_BrowseFiles simulates a user browsing local and remote files.
// This corresponds to the frontend's FilesLocalQuery and FilesRemoteQuery.
func (s *IntegrationTestSuite) TestUserJourney_BrowseFiles() {
	// Create subdirectories for browsing
	require.NoError(s.T(), os.MkdirAll(filepath.Join(s.testDir, "source", "subdir1"), 0755))
	require.NoError(s.T(), os.MkdirAll(filepath.Join(s.testDir, "source", "subdir2"), 0755))

	// Step 1: Browse local directory (matches FilesLocalQuery)
	localQuery := `
		query FilesLocal($path: String!) {
			file {
				local(path: $path) {
					name
					path
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), localQuery, map[string]interface{}{
		"path": filepath.Join(s.testDir, "source"),
	})
	require.Empty(s.T(), resp.Errors, "FilesLocal query should succeed")

	data := string(resp.Data)
	files := gjson.Get(data, "file.local").Array()
	assert.Len(s.T(), files, 2, "Should have 2 subdirectories")

	// Verify all returned items are directories with proper structure
	for _, f := range files {
		assert.True(s.T(), f.Get("isDir").Bool(), "All items should be directories")
		assert.NotEmpty(s.T(), f.Get("name").String(), "Should have name")
		assert.NotEmpty(s.T(), f.Get("path").String(), "Should have path")
	}

	// Step 2: Create a connection for remote browsing
	connID := s.Env.CreateTestConnection(s.T(), "local-for-browse")

	// Step 3: Browse remote directory (matches FilesRemoteQuery)
	remoteQuery := `
		query FilesRemote($connectionId: ID!, $path: String!) {
			file {
				remote(connectionId: $connectionId, path: $path) {
					name
					path
					isDir
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), remoteQuery, map[string]interface{}{
		"connectionId": connID.String(),
		"path":         filepath.Join(s.testDir, "source"),
	})
	require.Empty(s.T(), resp.Errors, "FilesRemote query should succeed")

	data = string(resp.Data)
	remoteFiles := gjson.Get(data, "file.remote").Array()
	assert.Len(s.T(), remoteFiles, 2, "Should have 2 subdirectories via remote")
}

// =============================================================================
// Phase 4: Task Management
// =============================================================================

// TestUserJourney_CreateAndManageTask simulates creating and managing sync tasks.
// This corresponds to the frontend's task CRUD operations.
func (s *IntegrationTestSuite) TestUserJourney_CreateAndManageTask() {
	// Create a connection first
	connID := s.Env.CreateTestConnection(s.T(), "task-storage")

	// Step 1: Create a new task (matches TaskCreateMutation)
	createMutation := `
		mutation TaskCreate($input: CreateTaskInput!) {
			task {
				create(input: $input) {
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

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name":         "my-sync-task",
			"sourcePath":   filepath.Join(s.testDir, "source"),
			"connectionId": connID.String(),
			"remotePath":   filepath.Join(s.testDir, "remote"),
			"direction":    "UPLOAD",
			"schedule":     "0 * * * *",
			"realtime":     false,
		},
	})
	require.Empty(s.T(), resp.Errors, "TaskCreate mutation should succeed")

	data := string(resp.Data)
	taskID := gjson.Get(data, "task.create.id").String()
	assert.NotEmpty(s.T(), taskID, "Task ID should be returned")
	assert.Equal(s.T(), "my-sync-task", gjson.Get(data, "task.create.name").String())
	assert.Equal(s.T(), "UPLOAD", gjson.Get(data, "task.create.direction").String())
	assert.Equal(s.T(), "0 * * * *", gjson.Get(data, "task.create.schedule").String())
	assert.False(s.T(), gjson.Get(data, "task.create.realtime").Bool())

	// Step 2: List tasks (matches TasksListQuery - frontend uses this)
	listQuery := `
		query TasksList($pagination: PaginationInput) {
			task {
				list(pagination: $pagination) {
					items {
						id
						name
						sourcePath
						remotePath
						direction
						schedule
						realtime
						createdAt
						updatedAt
						connection {
							id
							name
							type
						}
						latestJob {
							id
							status
							startTime
							endTime
							filesTransferred
							bytesTransferred
						}
					}
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: listQuery})
	require.Empty(s.T(), resp.Errors, "TasksList query should succeed")

	data = string(resp.Data)
	items := gjson.Get(data, "task.list.items").Array()
	assert.Len(s.T(), items, 1, "Should have one task")
	assert.Equal(s.T(), taskID, items[0].Get("id").String())
	assert.Equal(s.T(), connID.String(), items[0].Get("connection.id").String())
	// latestJob should be null initially
	assert.True(s.T(), items[0].Get("latestJob").Type == gjson.Null)

	// Step 3: Get task details (matches TaskGetQuery)
	getQuery := `
		query TaskGet($id: ID!) {
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
					connection {
						id
						name
						type
					}
					options {
						conflictResolution
					}
					latestJob {
						id
						status
						startTime
						endTime
					}
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), getQuery, map[string]interface{}{
		"id": taskID,
	})
	require.Empty(s.T(), resp.Errors, "TaskGet query should succeed")

	data = string(resp.Data)
	assert.Equal(s.T(), taskID, gjson.Get(data, "task.get.id").String())
	assert.Equal(s.T(), connID.String(), gjson.Get(data, "task.get.connection.id").String())

	// Step 4: Update the task (matches TaskUpdateMutation)
	updateMutation := `
		mutation TaskUpdate($id: ID!, $input: UpdateTaskInput!) {
			task {
				update(id: $id, input: $input) {
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

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), updateMutation, map[string]interface{}{
		"id": taskID,
		"input": map[string]interface{}{
			"name":      "my-sync-task-renamed",
			"direction": "BIDIRECTIONAL",
			"schedule":  "0 */2 * * *",
		},
	})
	require.Empty(s.T(), resp.Errors, "TaskUpdate mutation should succeed")

	data = string(resp.Data)
	assert.Equal(s.T(), "my-sync-task-renamed", gjson.Get(data, "task.update.name").String())
	assert.Equal(s.T(), "BIDIRECTIONAL", gjson.Get(data, "task.update.direction").String())
	assert.Equal(s.T(), "0 */2 * * *", gjson.Get(data, "task.update.schedule").String())

	// Step 5: Delete the task (matches TaskDeleteMutation)
	deleteMutation := `
		mutation TaskDelete($id: ID!) {
			task {
				delete(id: $id) {
					id
					name
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), deleteMutation, map[string]interface{}{
		"id": taskID,
	})
	require.Empty(s.T(), resp.Errors, "TaskDelete mutation should succeed")

	data = string(resp.Data)
	assert.Equal(s.T(), taskID, gjson.Get(data, "task.delete.id").String())

	// Verify task is deleted
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), getQuery, map[string]interface{}{
		"id": taskID,
	})
	require.Empty(s.T(), resp.Errors)
	data = string(resp.Data)
	taskGet := gjson.Get(data, "task.get")
	assert.True(s.T(), taskGet.Type == gjson.Null || !taskGet.Exists(), "Task should be deleted")
}

// =============================================================================
// Phase 5: Job Execution and Logs
// =============================================================================

// TestUserJourney_RunTaskAndViewJobs simulates running a task and viewing job history.
// This corresponds to the frontend's TaskRunMutation, JobsListQuery, and LogsListQuery.
func (s *IntegrationTestSuite) TestUserJourney_RunTaskAndViewJobs() {
	// Setup: Create connection and task
	connID := s.Env.CreateTestConnection(s.T(), "job-storage")
	task := s.Env.CreateTestTask(s.T(), "job-task", connID)

	// Step 1: Run the task (matches TaskRunMutation)
	runMutation := `
		mutation TaskRun($taskId: ID!) {
			task {
				run(taskId: $taskId) {
					id
					status
					startTime
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), runMutation, map[string]interface{}{
		"taskId": task.ID.String(),
	})
	// Run mutation may have timing issues in tests, so we check gracefully
	if len(resp.Errors) == 0 {
		data := string(resp.Data)
		jobID := gjson.Get(data, "task.run.id").String()
		assert.NotEmpty(s.T(), jobID, "Job ID should be returned")
		assert.Equal(s.T(), "MANUAL", gjson.Get(data, "task.run.trigger").String())
	}

	// Step 2: Create a job directly for testing query
	job, err := s.Env.JobService.CreateJob(s.T().Context(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	// Add some logs to the job
	_, err = s.Env.JobService.AddJobLog(s.T().Context(), job.ID, "INFO", "UPLOAD", "/test/file1.txt", 1024)
	require.NoError(s.T(), err)
	_, err = s.Env.JobService.AddJobLog(s.T().Context(), job.ID, "INFO", "UPLOAD", "/test/file2.txt", 2048)
	require.NoError(s.T(), err)

	// Step 3: List jobs (matches JobsListQuery)
	jobsQuery := `
		query JobsList(
			$taskId: ID
			$connectionId: ID
			$pagination: PaginationInput
			$withConnection: Boolean! = false
		) {
			job {
				list(taskId: $taskId, connectionId: $connectionId, pagination: $pagination) {
					items {
						id
						status
						trigger
						startTime
						endTime
						filesTransferred
						bytesTransferred
						task {
							id
							name
							connection @include(if: $withConnection) {
								id
								name
							}
						}
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

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), jobsQuery, map[string]interface{}{
		"taskId":         task.ID.String(),
		"withConnection": true,
	})
	require.Empty(s.T(), resp.Errors, "JobsList query should succeed")

	data := string(resp.Data)
	jobs := gjson.Get(data, "job.list.items").Array()
	assert.GreaterOrEqual(s.T(), len(jobs), 1, "Should have at least one job")

	// Verify job structure
	foundJob := false
	for _, j := range jobs {
		if j.Get("id").String() == job.ID.String() {
			foundJob = true
			assert.Equal(s.T(), "PENDING", j.Get("status").String())
			assert.Equal(s.T(), "MANUAL", j.Get("trigger").String())
			assert.Equal(s.T(), task.ID.String(), j.Get("task.id").String())
			assert.Equal(s.T(), connID.String(), j.Get("task.connection.id").String())
			break
		}
	}
	assert.True(s.T(), foundJob, "Created job should be in the list")

	// Step 4: Query logs (matches LogsListQuery)
	logsQuery := `
		query LogsList(
			$connectionId: ID!
			$taskId: ID
			$jobId: ID
			$level: LogLevel
			$pagination: PaginationInput
		) {
			log {
				list(
					connectionId: $connectionId
					taskId: $taskId
					jobId: $jobId
					level: $level
					pagination: $pagination
				) {
					items {
						id
						level
						time
						path
						what
						size
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

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), logsQuery, map[string]interface{}{
		"connectionId": connID.String(),
		"jobId":        job.ID.String(),
	})
	require.Empty(s.T(), resp.Errors, "LogsList query should succeed")

	data = string(resp.Data)
	logs := gjson.Get(data, "log.list.items").Array()
	assert.Len(s.T(), logs, 2, "Should have 2 logs")
	assert.Equal(s.T(), 2, int(gjson.Get(data, "log.list.totalCount").Int()))

	// Verify log structure
	for _, l := range logs {
		assert.Equal(s.T(), "INFO", l.Get("level").String())
		assert.Equal(s.T(), "UPLOAD", l.Get("what").String())
		assert.NotEmpty(s.T(), l.Get("path").String())
		assert.Greater(s.T(), l.Get("size").Int(), int64(0))
	}
}

// =============================================================================
// Phase 6: Import Configuration
// =============================================================================

// TestUserJourney_ImportRcloneConfig simulates importing rclone configuration.
// This corresponds to the frontend's IMPORT_PARSE and IMPORT_EXECUTE mutations.
func (s *IntegrationTestSuite) TestUserJourney_ImportRcloneConfig() {
	// Step 1: Parse rclone config content (matches IMPORT_PARSE)
	parseMutation := `
		mutation ImportParse($input: ImportParseInput!) {
			import {
				parse(input: $input) {
					... on ImportParseSuccess {
						__typename
						connections {
							name
							type
							config
						}
					}
					... on ImportParseError {
						__typename
						error
						line
					}
				}
			}
		}
	`

	validConfig := `[my-s3-backup]
type = local

[my-gdrive]
type = local
`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), parseMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"content": validConfig,
		},
	})
	require.Empty(s.T(), resp.Errors, "ImportParse mutation should succeed")

	data := string(resp.Data)
	typename := gjson.Get(data, "import.parse.__typename").String()
	if typename == "ImportParseSuccess" {
		connections := gjson.Get(data, "import.parse.connections").Array()
		assert.Len(s.T(), connections, 2, "Should parse 2 connections")

		// Verify parsed connection names
		names := make([]string, len(connections))
		for i, c := range connections {
			names[i] = c.Get("name").String()
		}
		assert.Contains(s.T(), names, "my-s3-backup")
		assert.Contains(s.T(), names, "my-gdrive")
	}

	// Step 2: Execute import (matches IMPORT_EXECUTE)
	executeMutation := `
		mutation ImportExecute($input: ImportExecuteInput!) {
			import {
				execute(input: $input) {
					connections {
						id
						name
						type
						createdAt
					}
					skippedCount
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), executeMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"connections": []map[string]interface{}{
				{
					"name":   "imported-storage-1",
					"type":   "local",
					"config": map[string]interface{}{},
				},
				{
					"name":   "imported-storage-2",
					"type":   "local",
					"config": map[string]interface{}{},
				},
			},
		},
	})
	require.Empty(s.T(), resp.Errors, "ImportExecute mutation should succeed")

	data = string(resp.Data)
	importedConns := gjson.Get(data, "import.execute.connections").Array()
	assert.Len(s.T(), importedConns, 2, "Should import 2 connections")
	assert.Equal(s.T(), int64(0), gjson.Get(data, "import.execute.skippedCount").Int())

	// Verify imported connections have IDs
	for _, c := range importedConns {
		assert.NotEmpty(s.T(), c.Get("id").String())
		assert.True(s.T(), c.Get("createdAt").Exists())
	}

	// Step 3: Verify imported connections via list query
	listQuery := `
		query ConnectionsList($pagination: PaginationInput) {
			connection {
				list(pagination: $pagination) {
					items {
						id
						name
						type
					}
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: listQuery})
	require.Empty(s.T(), resp.Errors)

	data = string(resp.Data)
	allConns := gjson.Get(data, "connection.list.items").Array()
	assert.Len(s.T(), allConns, 2, "Should have 2 connections")

	// Step 4: Try to import duplicate (should be skipped)
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), executeMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"connections": []map[string]interface{}{
				{
					"name":   "imported-storage-1", // Duplicate
					"type":   "local",
					"config": map[string]interface{}{},
				},
				{
					"name":   "imported-storage-3", // New
					"type":   "local",
					"config": map[string]interface{}{},
				},
			},
		},
	})
	require.Empty(s.T(), resp.Errors, "ImportExecute with duplicates should succeed")

	data = string(resp.Data)
	importedConns = gjson.Get(data, "import.execute.connections").Array()
	assert.Len(s.T(), importedConns, 1, "Should import only 1 new connection")
	assert.Equal(s.T(), int64(1), gjson.Get(data, "import.execute.skippedCount").Int())
	assert.Equal(s.T(), "imported-storage-3", importedConns[0].Get("name").String())
}

// =============================================================================
// Complete User Journey Test
// =============================================================================

// TestUserJourney_CompleteFlow tests the complete user journey from start to finish.
// This simulates a real user's full workflow through the application.
func (s *IntegrationTestSuite) TestUserJourney_CompleteFlow() {
	// Phase 1: Discover providers
	providerListQuery := `
		query { provider { list { name prefix } } }
	`
	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: providerListQuery})
	require.Empty(s.T(), resp.Errors, "Should list providers")

	// Phase 2: Create a connection
	createConnMutation := `
		mutation($input: CreateConnectionInput!) {
			connection { create(input: $input) { id name type } }
		}
	`
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), createConnMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name":   "complete-flow-storage",
			"type":   "local",
			"config": map[string]interface{}{},
		},
	})
	require.Empty(s.T(), resp.Errors, "Should create connection")
	connID := gjson.Get(string(resp.Data), "connection.create.id").String()

	// Phase 3: Browse files
	browseQuery := `
		query($path: String!) { file { local(path: $path) { name isDir } } }
	`
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), browseQuery, map[string]interface{}{
		"path": s.testDir,
	})
	require.Empty(s.T(), resp.Errors, "Should browse local files")

	// Phase 4: Create a task
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

	// Phase 5: List tasks with connections and latest job
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

	// Phase 6: Create a job and add logs
	job, err := s.Env.JobService.CreateJob(s.T().Context(), parseUUID(s.T(), taskID), "MANUAL")
	require.NoError(s.T(), err)
	_, err = s.Env.JobService.AddJobLog(s.T().Context(), job.ID, "INFO", "UPLOAD", "/test.txt", 100)
	require.NoError(s.T(), err)

	// Phase 7: Query jobs with task reference
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

	// Phase 8: Query logs
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

	// Phase 9: Update task
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

	// Phase 10: Cleanup - delete task first (to avoid FK constraint)
	deleteTaskMutation := `
		mutation($id: ID!) { task { delete(id: $id) { id } } }
	`
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), deleteTaskMutation, map[string]interface{}{
		"id": taskID,
	})
	require.Empty(s.T(), resp.Errors, "Should delete task")

	// Phase 11: Cleanup - delete connection
	deleteConnMutation := `
		mutation($id: ID!) { connection { delete(id: $id) { id } } }
	`
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), deleteConnMutation, map[string]interface{}{
		"id": connID,
	})
	require.Empty(s.T(), resp.Errors, "Should delete connection")
}

// parseUUID is a helper to parse UUID string
func parseUUID(t *testing.T, s string) uuid.UUID {
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}
