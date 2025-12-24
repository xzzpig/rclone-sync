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
	// Create 5 connections
	for i := 0; i < 5; i++ {
		s.Env.CreateTestConnection(s.T(), "conn-"+string(rune('A'+i)))
	}

	query := `
		query($pagination: PaginationInput) {
			connection {
				list(pagination: $pagination) {
					items {
						id
						name
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
	assert.Equal(s.T(), 5, int(gjson.Get(data, "connection.list.totalCount").Int()))
	assert.Equal(s.T(), 2, len(gjson.Get(data, "connection.list.items").Array()))
	assert.True(s.T(), gjson.Get(data, "connection.list.pageInfo.hasNextPage").Bool())
	assert.False(s.T(), gjson.Get(data, "connection.list.pageInfo.hasPreviousPage").Bool())

	// Test middle page
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"pagination": map[string]interface{}{
			"limit":  2,
			"offset": 2,
		},
	})
	require.Empty(s.T(), resp.Errors)

	data = string(resp.Data)
	assert.True(s.T(), gjson.Get(data, "connection.list.pageInfo.hasNextPage").Bool())
	assert.True(s.T(), gjson.Get(data, "connection.list.pageInfo.hasPreviousPage").Bool())
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
	query := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"id": uuid.New().String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	result := gjson.Get(data, "connection.get")
	assert.True(s.T(), !result.Exists() || result.Type == gjson.Null, "connection.get should be null for non-existent ID")
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
