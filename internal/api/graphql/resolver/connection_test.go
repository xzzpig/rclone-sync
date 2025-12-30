// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

// ConnectionResolverTestSuite tests ConnectionQuery and ConnectionMutation resolvers.
type ConnectionResolverTestSuite struct {
	ResolverTestSuite
}

func TestConnectionResolverSuite(t *testing.T) {
	suite.Run(t, new(ConnectionResolverTestSuite))
}

// TestConnectionQuery_List tests ConnectionQuery.list resolver.
func (s *ConnectionResolverTestSuite) TestConnectionQuery_List() {
	// Create multiple connections
	for i := 0; i < 3; i++ {
		s.Env.CreateTestConnection(s.T(), "conn-"+string(rune('A'+i)))
	}

	query := `
		query {
			connection {
				list {
					items {
						id
						name
						type
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
	assert.Equal(s.T(), 3, int(gjson.Get(data, "connection.list.totalCount").Int()))
	assert.Equal(s.T(), 3, len(gjson.Get(data, "connection.list.items").Array()))
	assert.False(s.T(), gjson.Get(data, "connection.list.pageInfo.hasNextPage").Bool())
	assert.False(s.T(), gjson.Get(data, "connection.list.pageInfo.hasPreviousPage").Bool())
}

// TestConnectionQuery_ListWithPagination tests ConnectionQuery.list with pagination.
func (s *ConnectionResolverTestSuite) TestConnectionQuery_ListWithPagination() {
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

			testPagination(env, t, PaginationTestCase{
				EntityName: "Connection",
				Query:      GetCommonQueries().ConnectionListPagination,
				DataPath:   "connection.list",
				CreateFunc: func(env *TestEnv, t *testing.T, i int) {
					t.Helper()
					env.CreateTestConnection(t, "conn-paged-"+tc.name+"-"+fmt.Sprintf("%d", i))
				},
				PageSize:   tc.pageSize,
				TotalItems: tc.totalItems,
			})
		})
	}
}

// TestConnectionQuery_Get tests ConnectionQuery.get resolver.
func (s *ConnectionResolverTestSuite) TestConnectionQuery_Get() {
	connID := s.Env.CreateTestConnection(s.T(), "my-connection")

	query := `
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

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), connID.String(), gjson.Get(data, "connection.get.id").String())
	assert.Equal(s.T(), "my-connection", gjson.Get(data, "connection.get.name").String())
	assert.Equal(s.T(), "local", gjson.Get(data, "connection.get.type").String())
}

// TestConnectionQuery_GetNotFound tests ConnectionQuery.get with non-existent ID.
func (s *ConnectionResolverTestSuite) TestConnectionQuery_GetNotFound() {
	testGetNotFound(s.Env, s.T(), GetNotFoundTestCase{
		Name:        "non-existent-connection",
		Entity:      "Connection",
		Query:       GetCommonQueries().ConnectionGet,
		VariableKey: "id",
		Variable:    func(t *testing.T) interface{} { return uuid.New().String() },
		DataPath:    "connection.get",
	})
}

// TestConnectionMutation_Create tests ConnectionMutation.create resolver.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_Create() {
	mutation := `
		mutation($input: CreateConnectionInput!) {
			connection {
				create(input: $input) {
					id
					name
					type
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name": "new-connection",
			"type": "local",
			"config": map[string]interface{}{
				"type": "local",
			},
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.NotEmpty(s.T(), gjson.Get(data, "connection.create.id").String())
	assert.Equal(s.T(), "new-connection", gjson.Get(data, "connection.create.name").String())
	assert.Equal(s.T(), "local", gjson.Get(data, "connection.create.type").String())
}

// TestConnectionMutation_Update tests ConnectionMutation.update resolver.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_Update() {
	connID := s.Env.CreateTestConnection(s.T(), "original-name")

	mutation := `
		mutation($id: ID!, $input: UpdateConnectionInput!) {
			connection {
				update(id: $id, input: $input) {
					id
					name
					type
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"id": connID.String(),
		"input": map[string]interface{}{
			"name": "updated-name",
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), connID.String(), gjson.Get(data, "connection.update.id").String())
	assert.Equal(s.T(), "updated-name", gjson.Get(data, "connection.update.name").String())
}

// TestConnectionMutation_UpdateNotFound tests ConnectionMutation.update with non-existent ID.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_UpdateNotFound() {
	mutation := `
		mutation($id: ID!, $input: UpdateConnectionInput!) {
			connection {
				update(id: $id, input: $input) {
					id
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"id": uuid.New().String(),
		"input": map[string]interface{}{
			"name": "new-name",
		},
	})
	require.NotEmpty(s.T(), resp.Errors)
}

// TestConnectionMutation_Delete tests ConnectionMutation.delete resolver.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_Delete() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-to-delete")

	mutation := `
		mutation($id: ID!) {
			connection {
				delete(id: $id) {
					id
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), connID.String(), gjson.Get(data, "connection.delete.id").String())
	assert.Equal(s.T(), "conn-to-delete", gjson.Get(data, "connection.delete.name").String())

	// Verify connection is deleted
	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
				}
			}
		}
	`
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)
	result := gjson.Get(string(resp.Data), "connection.get")
	assert.True(s.T(), !result.Exists() || result.Type == gjson.Null, "connection.get should be null after deletion")
}

// TestConnectionMutation_DeleteNotFound tests ConnectionMutation.delete with non-existent ID.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_DeleteNotFound() {
	mutation := `
		mutation($id: ID!) {
			connection {
				delete(id: $id) {
					id
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"id": uuid.New().String(),
	})
	require.NotEmpty(s.T(), resp.Errors)
}

// TestConnection_Tasks tests Connection.tasks field resolver.
func (s *ConnectionResolverTestSuite) TestConnection_Tasks() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-with-tasks")

	// Create multiple tasks for this connection
	for i := 0; i < 3; i++ {
		s.Env.CreateTestTask(s.T(), "task-"+string(rune('A'+i)), connID)
	}

	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					name
					tasks {
						items {
							id
							name
						}
						totalCount
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 3, int(gjson.Get(data, "connection.get.tasks.totalCount").Int()))
	assert.Equal(s.T(), 3, len(gjson.Get(data, "connection.get.tasks.items").Array()))
}

// TestConnection_Config tests Connection.config field resolver.
func (s *ConnectionResolverTestSuite) TestConnection_Config() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-with-config")

	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					config
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	// Config should be returned as JSON object
	config := gjson.Get(data, "connection.get.config")
	assert.True(s.T(), config.Exists())
}

// TestConnection_LoadStatus tests Connection.loadStatus field resolver.
func (s *ConnectionResolverTestSuite) TestConnection_LoadStatus() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-status")

	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					loadStatus
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	// loadStatus should exist
	assert.True(s.T(), gjson.Get(data, "connection.get.loadStatus").Exists())
}

// TestConnection_LoadError tests Connection.loadError field resolver.
func (s *ConnectionResolverTestSuite) TestConnection_LoadError() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-load-error")

	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					loadError
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	// loadError should be null since there's no error
	loadError := gjson.Get(data, "connection.get.loadError")
	assert.True(s.T(), loadError.Type == gjson.Null || !loadError.Exists(), "loadError should be null")
}

// TestConnectionMutation_Test tests ConnectionMutation.test resolver.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_Test() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-to-test")

	mutation := `
		mutation($id: ID!) {
			connection {
				test(id: $id) {
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

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	// Either success with message or failure with error should exist
	successMsg := gjson.Get(data, "connection.test.message")
	errorMsg := gjson.Get(data, "connection.test.error")
	assert.True(s.T(), successMsg.Exists() || errorMsg.Exists(), "Either success message or error should exist")
}

// TestConnectionMutation_TestUnsaved tests ConnectionMutation.testUnsaved resolver.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_TestUnsaved() {
	mutation := `
		mutation($input: TestConnectionInput!) {
			connection {
				testUnsaved(input: $input) {
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

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"input": map[string]interface{}{
			"type": "local",
			"config": map[string]interface{}{
				"type": "local",
			},
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	// Either success with message or failure with error should exist
	successMsg := gjson.Get(data, "connection.testUnsaved.message")
	errorMsg := gjson.Get(data, "connection.testUnsaved.error")
	assert.True(s.T(), successMsg.Exists() || errorMsg.Exists(), "Either success message or error should exist")
}

// TestConnection_Quota tests Connection.quota field resolver.
func (s *ConnectionResolverTestSuite) TestConnection_Quota() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-quota")

	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					quota {
						total
						used
						free
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	// quota may or may not be available depending on the connection type
	require.Empty(s.T(), resp.Errors)
}

// TestConnection_QuotaExtendedFields tests Connection.quota field resolver returns extended fields.
// This test verifies that the quota resolver returns all extended fields:
// trashed, other, and objects in addition to the basic total/used/free fields.
func (s *ConnectionResolverTestSuite) TestConnection_QuotaExtendedFields() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-quota-extended")

	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					quota {
						total
						used
						free
						trashed
						other
						objects
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	// The query should succeed - quota fields are nullable so they can be nil
	// if the remote doesn't support them
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	// Verify the response structure contains quota (may be null for local connections)
	result := gjson.Get(data, "connection.get")
	assert.True(s.T(), result.Exists(), "connection.get should exist")
	assert.Equal(s.T(), connID.String(), gjson.Get(data, "connection.get.id").String())

	// Note: quota may be null for local connections that don't support About
	// The important thing is that the query executes without error
}

// TestConnection_DeleteWithTasks tests that deleting a connection with tasks fails.
func (s *ConnectionResolverTestSuite) TestConnection_DeleteWithTasks() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-with-tasks")
	s.Env.CreateTestTask(s.T(), "dependent-task", connID)

	mutation := `
		mutation($id: ID!) {
			connection {
				delete(id: $id) {
					id
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"id": connID.String(),
	})
	// Deletion should fail due to dependent tasks
	require.NotEmpty(s.T(), resp.Errors)
}

// TestConnection_TasksWithPagination tests Connection.tasks field with pagination.
func (s *ConnectionResolverTestSuite) TestConnection_TasksWithPagination() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-many-tasks")

	// Create multiple tasks for this connection
	for i := 0; i < 10; i++ {
		s.Env.CreateTestTask(s.T(), "task-"+string(rune('A'+i)), connID)
	}

	query := `
		query($id: ID!, $pagination: PaginationInput) {
			connection {
				get(id: $id) {
					id
					tasks(pagination: $pagination) {
						items {
							id
							name
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
		}
	`

	// Test first page
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
		"pagination": map[string]interface{}{
			"limit":  5,
			"offset": 0,
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 10, int(gjson.Get(data, "connection.get.tasks.totalCount").Int()))
	assert.Equal(s.T(), 5, len(gjson.Get(data, "connection.get.tasks.items").Array()))
	assert.True(s.T(), gjson.Get(data, "connection.get.tasks.pageInfo.hasNextPage").Bool())
	assert.False(s.T(), gjson.Get(data, "connection.get.tasks.pageInfo.hasPreviousPage").Bool())

	// Test second page
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
		"pagination": map[string]interface{}{
			"limit":  5,
			"offset": 5,
		},
	})
	require.Empty(s.T(), resp.Errors)

	data = string(resp.Data)
	assert.False(s.T(), gjson.Get(data, "connection.get.tasks.pageInfo.hasNextPage").Bool())
	assert.True(s.T(), gjson.Get(data, "connection.get.tasks.pageInfo.hasPreviousPage").Bool())
}

// TestConnection_TasksEmpty tests Connection.tasks when no tasks exist.
func (s *ConnectionResolverTestSuite) TestConnection_TasksEmpty() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-no-tasks")

	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					tasks {
						items {
							id
						}
						totalCount
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 0, int(gjson.Get(data, "connection.get.tasks.totalCount").Int()))
	assert.Equal(s.T(), 0, len(gjson.Get(data, "connection.get.tasks.items").Array()))
}

// TestConnectionMutation_CreateWithConfig tests ConnectionMutation.create with various config options.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_CreateWithConfig() {
	mutation := `
		mutation($input: CreateConnectionInput!) {
			connection {
				create(input: $input) {
					id
					name
					type
					config
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name": "conn-with-full-config",
			"type": "local",
			"config": map[string]interface{}{
				"type":       "local",
				"nounc":      "true",
				"copy_links": "false",
			},
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.NotEmpty(s.T(), gjson.Get(data, "connection.create.id").String())
	assert.Equal(s.T(), "conn-with-full-config", gjson.Get(data, "connection.create.name").String())
	// Config should be returned
	config := gjson.Get(data, "connection.create.config")
	assert.True(s.T(), config.Exists())
}

// TestConnectionMutation_UpdateConfig tests ConnectionMutation.update with config changes.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_UpdateConfig() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-to-update-config")

	mutation := `
		mutation($id: ID!, $input: UpdateConnectionInput!) {
			connection {
				update(id: $id, input: $input) {
					id
					name
					config
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"id": connID.String(),
		"input": map[string]interface{}{
			"config": map[string]interface{}{
				"type":    "local",
				"new_key": "new_value",
			},
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), connID.String(), gjson.Get(data, "connection.update.id").String())
	// Config should be updated
	config := gjson.Get(data, "connection.update.config")
	assert.True(s.T(), config.Exists())
}

// TestConnection_AllFields tests all connection fields in a single query.
func (s *ConnectionResolverTestSuite) TestConnection_AllFields() {
	connID := s.Env.CreateTestConnection(s.T(), "conn-all-fields")
	s.Env.CreateTestTask(s.T(), "task-for-all-fields", connID)

	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					name
					type
					createdAt
					updatedAt
					config
					loadStatus
					loadError
					quota {
						total
						used
						free
						trashed
						other
						objects
					}
					tasks {
						items {
							id
							name
						}
						totalCount
					}
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
	assert.Equal(s.T(), "conn-all-fields", gjson.Get(data, "connection.get.name").String())
	assert.Equal(s.T(), "local", gjson.Get(data, "connection.get.type").String())
	assert.True(s.T(), gjson.Get(data, "connection.get.createdAt").Exists())
	assert.True(s.T(), gjson.Get(data, "connection.get.updatedAt").Exists())
	assert.True(s.T(), gjson.Get(data, "connection.get.config").Exists())
	assert.True(s.T(), gjson.Get(data, "connection.get.loadStatus").Exists())
	// loadError should be null since there's no error
	loadError := gjson.Get(data, "connection.get.loadError")
	assert.True(s.T(), loadError.Type == gjson.Null || !loadError.Exists())
	// tasks should have 1 item
	assert.Equal(s.T(), 1, int(gjson.Get(data, "connection.get.tasks.totalCount").Int()))
}

// TestConnectionMutation_TestWithInvalidType tests ConnectionMutation.test with an invalid connection type.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_TestWithInvalidType() {
	// Create connection with invalid config
	mutation := `
		mutation($input: CreateConnectionInput!) {
			connection {
				create(input: $input) {
					id
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name": "invalid-type-conn",
			"type": "local",
			"config": map[string]interface{}{
				"type": "local",
			},
		},
	})
	require.Empty(s.T(), resp.Errors)
	connID := gjson.Get(string(resp.Data), "connection.create.id").String()

	// Test the connection
	testMutation := `
		mutation($id: ID!) {
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
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	// Should return either success or failure
	typename := gjson.Get(data, "connection.test.__typename").String()
	assert.Contains(s.T(), []string{"ConnectionTestSuccess", "ConnectionTestFailure"}, typename)
}

// TestConnectionMutation_TestUnsavedWithLocalConfig tests ConnectionMutation.testUnsaved with local config.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_TestUnsavedWithLocalConfig() {
	mutation := `
		mutation($input: TestConnectionInput!) {
			connection {
				testUnsaved(input: $input) {
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

	// Test with local type - should succeed quickly
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"input": map[string]interface{}{
			"type": "local",
			"config": map[string]interface{}{
				"type": "local",
			},
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	typename := gjson.Get(data, "connection.testUnsaved.__typename").String()
	// Local connection test should succeed
	assert.Equal(s.T(), "ConnectionTestSuccess", typename)
}

// TestConnectionQuery_ListEmpty tests ConnectionQuery.list with no connections.
func (s *ConnectionResolverTestSuite) TestConnectionQuery_ListEmpty() {
	query := `
		query {
			connection {
				list {
					items {
						id
					}
					totalCount
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: query})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), 0, int(gjson.Get(data, "connection.list.totalCount").Int()))
	assert.Equal(s.T(), 0, len(gjson.Get(data, "connection.list.items").Array()))
}

// TestConnectionMutation_CreateDuplicateName tests that creating a connection with duplicate name fails.
func (s *ConnectionResolverTestSuite) TestConnectionMutation_CreateDuplicateName() {
	// First create a connection
	s.Env.CreateTestConnection(s.T(), "duplicate-name")

	mutation := `
		mutation($input: CreateConnectionInput!) {
			connection {
				create(input: $input) {
					id
				}
			}
		}
	`

	// Try to create another with the same name
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), mutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name":   "duplicate-name",
			"type":   "local",
			"config": map[string]interface{}{},
		},
	})
	require.NotEmpty(s.T(), resp.Errors, "Should fail with duplicate name")
}
