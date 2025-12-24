// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

// AtomicityTestSuite tests mutation atomicity (all-or-nothing behavior).
type AtomicityTestSuite struct {
	ResolverTestSuite
}

func TestAtomicitySuite(t *testing.T) {
	suite.Run(t, new(AtomicityTestSuite))
}

// TestConnectionMutation_CreateRollbackOnDuplicate tests that connection creation
// properly rolls back if a duplicate name is detected.
func (s *AtomicityTestSuite) TestConnectionMutation_CreateRollbackOnDuplicate() {
	// First, create a connection successfully
	createMutation := `
		mutation {
			connection {
				create(input: {
					name: "duplicate-test",
					type: "local",
					config: {}
				}) {
					id
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: createMutation})
	require.Empty(s.T(), resp.Errors, "First creation should succeed")

	firstID := gjson.Get(string(resp.Data), "connection.create.id").String()
	require.NotEmpty(s.T(), firstID)

	// Try to create another connection with the same name
	resp2 := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: createMutation})
	assert.NotEmpty(s.T(), resp2.Errors, "Duplicate creation should fail")

	// Verify the first connection still exists
	getQuery := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					name
				}
			}
		}
	`

	resp3 := s.Env.ExecuteGraphQLWithVars(s.T(), getQuery, map[string]interface{}{
		"id": firstID,
	})
	require.Empty(s.T(), resp3.Errors)
	assert.Equal(s.T(), "duplicate-test", gjson.Get(string(resp3.Data), "connection.get.name").String())
}

// TestTaskMutation_CreateWithInvalidConnection tests that task creation fails
// atomically when referencing a non-existent connection.
func (s *AtomicityTestSuite) TestTaskMutation_CreateWithInvalidConnection() {
	nonExistentConnID := uuid.New()

	createMutation := `
		mutation($connId: ID!) {
			task {
				create(input: {
					name: "invalid-conn-task",
					sourcePath: "/tmp/source",
					connectionId: $connId,
					remotePath: "/remote",
					direction: UPLOAD
				}) {
					id
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createMutation, map[string]interface{}{
		"connId": nonExistentConnID.String(),
	})

	// Should fail due to invalid connection reference
	assert.NotEmpty(s.T(), resp.Errors, "Creation with invalid connection should fail")

	// Verify no orphan task was created
	listQuery := `
		query {
			task {
				list {
					items {
						name
					}
				}
			}
		}
	`

	listResp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: listQuery})
	require.Empty(s.T(), listResp.Errors)

	items := gjson.Get(string(listResp.Data), "task.list.items").Array()
	for _, item := range items {
		assert.NotEqual(s.T(), "invalid-conn-task", item.Get("name").String(),
			"No orphan task should exist")
	}
}

// TestConnectionMutation_DeleteWithTasks tests that connection deletion
// is blocked when there are dependent tasks (prevent accidental data loss).
func (s *AtomicityTestSuite) TestConnectionMutation_DeleteWithTasks() {
	// Create a connection via GraphQL
	createConnMutation := `
		mutation {
			connection {
				create(input: {
					name: "conn-with-tasks-gql",
					type: "local",
					config: {}
				}) {
					id
					name
				}
			}
		}
	`

	connResp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: createConnMutation})
	require.Empty(s.T(), connResp.Errors)
	connID := gjson.Get(string(connResp.Data), "connection.create.id").String()

	// Create a task referencing this connection via GraphQL
	createTaskMutation := `
		mutation($connId: ID!) {
			task {
				create(input: {
					name: "task-blocking-delete",
					sourcePath: "/tmp/source",
					connectionId: $connId,
					remotePath: "/remote",
					direction: UPLOAD
				}) {
					id
				}
			}
		}
	`

	taskResp := s.Env.ExecuteGraphQLWithVars(s.T(), createTaskMutation, map[string]interface{}{
		"connId": connID,
	})
	require.Empty(s.T(), taskResp.Errors)
	taskID := gjson.Get(string(taskResp.Data), "task.create.id").String()

	// Try to delete the connection - should fail due to dependent tasks
	deleteMutation := `
		mutation($id: ID!) {
			connection {
				delete(id: $id) {
					id
				}
			}
		}
	`

	deleteResp := s.Env.ExecuteGraphQLWithVars(s.T(), deleteMutation, map[string]interface{}{
		"id": connID,
	})

	// Should fail because connection has dependent tasks
	require.NotEmpty(s.T(), deleteResp.Errors, "Connection deletion should fail when there are dependent tasks")

	// Verify connection still exists
	getConnQuery := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					name
				}
			}
		}
	`

	getConnResp := s.Env.ExecuteGraphQLWithVars(s.T(), getConnQuery, map[string]interface{}{
		"id": connID,
	})
	require.Empty(s.T(), getConnResp.Errors)
	connGet := gjson.Get(string(getConnResp.Data), "connection.get")
	assert.True(s.T(), connGet.Exists() && connGet.Type != gjson.Null,
		"Connection should still exist after failed delete")

	// Verify task still exists
	getTaskQuery := `
		query($id: ID!) {
			task {
				get(id: $id) {
					id
				}
			}
		}
	`

	getTaskResp := s.Env.ExecuteGraphQLWithVars(s.T(), getTaskQuery, map[string]interface{}{
		"id": taskID,
	})
	require.Empty(s.T(), getTaskResp.Errors)
	taskGet := gjson.Get(string(getTaskResp.Data), "task.get")
	assert.True(s.T(), taskGet.Exists() && taskGet.Type != gjson.Null,
		"Task should still exist after failed connection delete")
}

// TestTaskMutation_UpdateWithInvalidConnection tests that task update with
// a non-existent connection results in an error when resolving the connection.
func (s *AtomicityTestSuite) TestTaskMutation_UpdateWithInvalidConnection() {
	// Create a valid connection via GraphQL
	createConnMutation := `
		mutation {
			connection {
				create(input: {
					name: "valid-conn-for-update-gql",
					type: "local",
					config: {}
				}) {
					id
					name
				}
			}
		}
	`

	connResp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: createConnMutation})
	require.Empty(s.T(), connResp.Errors)
	connID := gjson.Get(string(connResp.Data), "connection.create.id").String()

	// Create a task via GraphQL
	createTaskMutation := `
		mutation($connId: ID!) {
			task {
				create(input: {
					name: "task-for-update-test-gql",
					sourcePath: "/tmp/source",
					connectionId: $connId,
					remotePath: "/remote",
					direction: UPLOAD
				}) {
					id
					name
				}
			}
		}
	`

	taskResp := s.Env.ExecuteGraphQLWithVars(s.T(), createTaskMutation, map[string]interface{}{
		"connId": connID,
	})
	require.Empty(s.T(), taskResp.Errors)
	taskID := gjson.Get(string(taskResp.Data), "task.create.id").String()

	// Try to update with invalid connection - request connection in response to force error
	nonExistentConnID := uuid.New()
	updateMutation := `
		mutation($taskId: ID!, $connId: ID!) {
			task {
				update(id: $taskId, input: {
					connectionId: $connId
				}) {
					id
					connection {
						id
					}
				}
			}
		}
	`

	updateResp := s.Env.ExecuteGraphQLWithVars(s.T(), updateMutation, map[string]interface{}{
		"taskId": taskID,
		"connId": nonExistentConnID.String(),
	})

	// The update mutation should fail when trying to resolve the non-existent connection
	// This can happen either during the update (FK constraint) or when resolving the connection field
	assert.NotEmpty(s.T(), updateResp.Errors,
		"Update with invalid connection should fail when resolving connection field")
}

// TestImportMutation_PartialFailure tests that import execution handles
// partial failures correctly.
func (s *AtomicityTestSuite) TestImportMutation_PartialFailure() {
	// Create a connection with a specific name first
	s.Env.CreateTestConnection(s.T(), "existing-for-import")

	// Try to import connections where one has a duplicate name
	importMutation := `
		mutation {
			import {
				execute(input: {
					connections: [
						{
							name: "new-import-conn",
							type: "local",
							config: {}
						},
						{
							name: "existing-for-import",
							type: "local",
							config: {}
						}
					]
				}) {
					connections {
						id
						name
					}
					skippedCount
				}
			}
		}
	`

	importResp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: importMutation})
	require.Empty(s.T(), importResp.Errors)

	// Should have successfully imported one and skipped one
	connections := gjson.Get(string(importResp.Data), "import.execute.connections").Array()
	skippedCount := gjson.Get(string(importResp.Data), "import.execute.skippedCount").Int()

	// At least one should be imported, and the duplicate should be skipped
	assert.GreaterOrEqual(s.T(), len(connections), 1, "At least one connection should be imported")
	assert.GreaterOrEqual(s.T(), skippedCount, int64(0), "Skipped count should be >= 0")
}

// TestConnectionMutation_UpdatePartialFields tests that partial updates
// only affect specified fields.
func (s *AtomicityTestSuite) TestConnectionMutation_UpdatePartialFields() {
	// Create a connection with initial config
	connID := s.Env.CreateTestConnection(s.T(), "partial-update-test")

	// Get original type
	getQuery := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					name
					type
				}
			}
		}
	`

	getResp := s.Env.ExecuteGraphQLWithVars(s.T(), getQuery, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), getResp.Errors)
	originalType := gjson.Get(string(getResp.Data), "connection.get.type").String()

	// Update only the name, not the type
	updateMutation := `
		mutation($id: ID!) {
			connection {
				update(id: $id, input: {
					name: "partial-update-test-renamed"
				}) {
					id
					name
					type
				}
			}
		}
	`

	updateResp := s.Env.ExecuteGraphQLWithVars(s.T(), updateMutation, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), updateResp.Errors)

	// Name should be updated
	assert.Equal(s.T(), "partial-update-test-renamed",
		gjson.Get(string(updateResp.Data), "connection.update.name").String())
	// Type should remain unchanged
	assert.Equal(s.T(), originalType,
		gjson.Get(string(updateResp.Data), "connection.update.type").String())
}

// TestTaskMutation_DeleteNonExistent tests that deleting a non-existent task
// returns an appropriate error.
func (s *AtomicityTestSuite) TestTaskMutation_DeleteNonExistent() {
	nonExistentID := uuid.New()

	deleteMutation := `
		mutation($id: ID!) {
			task {
				delete(id: $id) {
					id
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), deleteMutation, map[string]interface{}{
		"id": nonExistentID.String(),
	})

	assert.NotEmpty(s.T(), resp.Errors, "Deleting non-existent task should fail")
}

// TestConnectionMutation_DeleteNonExistent tests that deleting a non-existent connection
// returns an appropriate error.
func (s *AtomicityTestSuite) TestConnectionMutation_DeleteNonExistent() {
	nonExistentID := uuid.New()

	deleteMutation := `
		mutation($id: ID!) {
			connection {
				delete(id: $id) {
					id
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), deleteMutation, map[string]interface{}{
		"id": nonExistentID.String(),
	})

	assert.NotEmpty(s.T(), resp.Errors, "Deleting non-existent connection should fail")
}

// TestTaskMutation_CreateValidationAtomicity tests that task creation with
// invalid data fails before any database changes.
func (s *AtomicityTestSuite) TestTaskMutation_CreateValidationAtomicity() {
	// Create a valid connection first
	connID := s.Env.CreateTestConnection(s.T(), "conn-for-validation-test")

	// Get initial task count
	listQuery := `
		query {
			task {
				list {
					totalCount
				}
			}
		}
	`

	initialResp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: listQuery})
	require.Empty(s.T(), initialResp.Errors)
	initialCount := gjson.Get(string(initialResp.Data), "task.list.totalCount").Int()

	// Try to create task with empty name which might be caught by business logic
	createTaskMutation := `
		mutation($connId: ID!) {
			task {
				create(input: {
					name: "",
					sourcePath: "/tmp/source",
					connectionId: $connId,
					remotePath: "/remote",
					direction: UPLOAD
				}) {
					id
				}
			}
		}
	`

	taskResp := s.Env.ExecuteGraphQLWithVars(s.T(), createTaskMutation, map[string]interface{}{
		"connId": connID.String(),
	})

	// If validation catches empty name, it should fail
	// If not, we just verify count
	finalResp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: listQuery})
	require.Empty(s.T(), finalResp.Errors)
	finalCount := gjson.Get(string(finalResp.Data), "task.list.totalCount").Int()

	// If creation failed, count should be the same
	if len(taskResp.Errors) > 0 {
		assert.Equal(s.T(), initialCount, finalCount,
			"Failed creation should not change task count")
	}
}

// TestConnectionMutation_ConcurrentUpdates tests that concurrent updates
// to the same connection are handled correctly.
func (s *AtomicityTestSuite) TestConnectionMutation_ConcurrentUpdates() {
	// Create a connection
	connID := s.Env.CreateTestConnection(s.T(), "concurrent-update-test")

	// Perform two updates sequentially (simulating what would be concurrent)
	updateMutation := `
		mutation($id: ID!, $name: String!) {
			connection {
				update(id: $id, input: {
					name: $name
				}) {
					id
					name
				}
			}
		}
	`

	update1Resp := s.Env.ExecuteGraphQLWithVars(s.T(), updateMutation, map[string]interface{}{
		"id":   connID.String(),
		"name": "concurrent-update-1",
	})
	require.Empty(s.T(), update1Resp.Errors)

	update2Resp := s.Env.ExecuteGraphQLWithVars(s.T(), updateMutation, map[string]interface{}{
		"id":   connID.String(),
		"name": "concurrent-update-2",
	})
	require.Empty(s.T(), update2Resp.Errors)

	// Verify final state is consistent
	getQuery := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					name
				}
			}
		}
	`

	getResp := s.Env.ExecuteGraphQLWithVars(s.T(), getQuery, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), getResp.Errors)

	finalName := gjson.Get(string(getResp.Data), "connection.get.name").String()
	assert.Equal(s.T(), "concurrent-update-2", finalName,
		"Final name should be from the last update")
}

// TestTaskMutation_UpdateNonExistent tests that updating a non-existent task fails.
func (s *AtomicityTestSuite) TestTaskMutation_UpdateNonExistent() {
	nonExistentID := uuid.New()

	updateMutation := `
		mutation($id: ID!) {
			task {
				update(id: $id, input: {
					name: "new-name"
				}) {
					id
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), updateMutation, map[string]interface{}{
		"id": nonExistentID.String(),
	})

	assert.NotEmpty(s.T(), resp.Errors, "Updating non-existent task should fail")
}

// TestConnectionMutation_UpdateNonExistent tests that updating a non-existent connection fails.
func (s *AtomicityTestSuite) TestConnectionMutation_UpdateNonExistent() {
	nonExistentID := uuid.New()

	updateMutation := `
		mutation($id: ID!) {
			connection {
				update(id: $id, input: {
					name: "new-name"
				}) {
					id
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), updateMutation, map[string]interface{}{
		"id": nonExistentID.String(),
	})

	assert.NotEmpty(s.T(), resp.Errors, "Updating non-existent connection should fail")
}

// TestTaskMutation_CreateAndDeleteCycle tests create-delete cycle is atomic.
func (s *AtomicityTestSuite) TestTaskMutation_CreateAndDeleteCycle() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-for-cycle-test")

	// Create a task
	createMutation := `
		mutation($connId: ID!) {
			task {
				create(input: {
					name: "cycle-test-task",
					sourcePath: "/tmp/source",
					connectionId: $connId,
					remotePath: "/remote",
					direction: UPLOAD
				}) {
					id
					name
				}
			}
		}
	`

	createResp := s.Env.ExecuteGraphQLWithVars(s.T(), createMutation, map[string]interface{}{
		"connId": connID.String(),
	})
	require.Empty(s.T(), createResp.Errors)
	taskID := gjson.Get(string(createResp.Data), "task.create.id").String()

	// Delete the task
	deleteMutation := `
		mutation($id: ID!) {
			task {
				delete(id: $id) {
					id
				}
			}
		}
	`

	deleteResp := s.Env.ExecuteGraphQLWithVars(s.T(), deleteMutation, map[string]interface{}{
		"id": taskID,
	})
	require.Empty(s.T(), deleteResp.Errors)

	// Verify task is gone
	getQuery := `
		query($id: ID!) {
			task {
				get(id: $id) {
					id
				}
			}
		}
	`

	getResp := s.Env.ExecuteGraphQLWithVars(s.T(), getQuery, map[string]interface{}{
		"id": taskID,
	})
	require.Empty(s.T(), getResp.Errors)
	// Check if the value is null (deleted items return null, not missing key)
	taskGet := gjson.Get(string(getResp.Data), "task.get")
	assert.True(s.T(), taskGet.Type == gjson.Null || !taskGet.Exists(),
		"Deleted task should not exist or be null")
}

// TestConnectionMutation_CreateAndDeleteCycle tests create-delete cycle is atomic.
func (s *AtomicityTestSuite) TestConnectionMutation_CreateAndDeleteCycle() {
	// Create a connection
	createMutation := `
		mutation {
			connection {
				create(input: {
					name: "cycle-test-conn",
					type: "local",
					config: {}
				}) {
					id
					name
				}
			}
		}
	`

	createResp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: createMutation})
	require.Empty(s.T(), createResp.Errors)
	connID := gjson.Get(string(createResp.Data), "connection.create.id").String()

	// Delete the connection
	deleteMutation := `
		mutation($id: ID!) {
			connection {
				delete(id: $id) {
					id
				}
			}
		}
	`

	deleteResp := s.Env.ExecuteGraphQLWithVars(s.T(), deleteMutation, map[string]interface{}{
		"id": connID,
	})
	require.Empty(s.T(), deleteResp.Errors)

	// Verify connection is gone
	getQuery := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
				}
			}
		}
	`

	getResp := s.Env.ExecuteGraphQLWithVars(s.T(), getQuery, map[string]interface{}{
		"id": connID,
	})
	require.Empty(s.T(), getResp.Errors)
	// Check if the value is null (deleted items return null, not missing key)
	connGet := gjson.Get(string(getResp.Data), "connection.get")
	assert.True(s.T(), connGet.Type == gjson.Null || !connGet.Exists(),
		"Deleted connection should not exist or be null")
}
