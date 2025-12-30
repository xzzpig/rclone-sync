// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// PaginationTestCase represents a pagination test case.
type PaginationTestCase struct {
	EntityName string
	Query      string
	DataPath   string                          // e.g., "connection.list"
	CreateFunc func(*TestEnv, *testing.T, int) // Function to create test items, receives index
	PageSize   int
	TotalItems int
}

// testPagination is a generic pagination test helper.
func testPagination(env *TestEnv, t *testing.T, tc PaginationTestCase) {
	// Create test items
	for i := 0; i < tc.TotalItems; i++ {
		tc.CreateFunc(env, t, i)
	}

	// Test first page
	resp := env.ExecuteGraphQLWithVars(t, tc.Query, map[string]interface{}{
		"pagination": map[string]interface{}{
			"limit":  tc.PageSize,
			"offset": 0,
		},
	})
	require.Empty(t, resp.Errors)

	data := string(resp.Data)
	actualTotalCount := int(gjson.Get(data, tc.DataPath+".totalCount").Int())

	// Only check items count for the current page
	items := gjson.Get(data, tc.DataPath+".items").Array()
	require.Equal(t, tc.PageSize, len(items))
	require.True(t, actualTotalCount >= tc.PageSize, "total count should be at least page size")
	assert.True(t, gjson.Get(data, tc.DataPath+".pageInfo.hasNextPage").Bool())
	assert.False(t, gjson.Get(data, tc.DataPath+".pageInfo.hasPreviousPage").Bool())

	// Test middle page (only if we have more than 2 pages)
	if actualTotalCount > tc.PageSize*2 {
		middleOffset := tc.PageSize
		resp = env.ExecuteGraphQLWithVars(t, tc.Query, map[string]interface{}{
			"pagination": map[string]interface{}{
				"limit":  tc.PageSize,
				"offset": middleOffset,
			},
		})
		require.Empty(t, resp.Errors)

		data = string(resp.Data)
		assert.True(t, gjson.Get(data, tc.DataPath+".pageInfo.hasNextPage").Bool())
		assert.True(t, gjson.Get(data, tc.DataPath+".pageInfo.hasPreviousPage").Bool())
	}

	// Test last page
	lastPageOffset := ((actualTotalCount - 1) / tc.PageSize) * tc.PageSize
	expectedLastPageSize := actualTotalCount - lastPageOffset

	resp = env.ExecuteGraphQLWithVars(t, tc.Query, map[string]interface{}{
		"pagination": map[string]interface{}{
			"limit":  tc.PageSize,
			"offset": lastPageOffset,
		},
	})
	require.Empty(t, resp.Errors)

	data = string(resp.Data)
	actualLastPageSize := len(gjson.Get(data, tc.DataPath+".items").Array())
	assert.Equal(t, expectedLastPageSize, actualLastPageSize)
	assert.False(t, gjson.Get(data, tc.DataPath+".pageInfo.hasNextPage").Bool())
	assert.True(t, gjson.Get(data, tc.DataPath+".pageInfo.hasPreviousPage").Bool())
}

// GetNotFoundTestCase represents a test case for get operations with non-existent IDs.
type GetNotFoundTestCase struct {
	Name        string
	Entity      string
	Query       string
	VariableKey string
	Variable    func(*testing.T) interface{}
	DataPath    string // e.g., "connection.get"
}

// testGetNotFound is a generic test helper for testing get operations with non-existent IDs.
func testGetNotFound(env *TestEnv, t *testing.T, tc GetNotFoundTestCase) {
	variables := map[string]interface{}{
		tc.VariableKey: tc.Variable(t),
	}

	resp := env.ExecuteGraphQLWithVars(t, tc.Query, variables)
	require.Empty(t, resp.Errors)

	data := string(resp.Data)
	result := gjson.Get(data, tc.DataPath)
	assert.True(t, !result.Exists() || result.Type == gjson.Null,
		tc.Entity+".get should be null for non-existent ID")
}

// CommonGraphQLQueries contains reusable GraphQL query templates for testing.
// These queries are shared across multiple test files to avoid duplication.
type CommonGraphQLQueries struct {
	ConnectionListPagination string
	TaskListPagination       string
	JobListPagination        string
	LogListPagination        string
	ConnectionGet            string
	TaskGet                  string
}

// GetCommonQueries returns common GraphQL query templates.
func GetCommonQueries() CommonGraphQLQueries {
	return CommonGraphQLQueries{
		ConnectionListPagination: `
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
		`,
		TaskListPagination: `
			query($pagination: PaginationInput) {
				task {
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
		`,
		JobListPagination: `
			query($pagination: PaginationInput) {
				job {
					list(pagination: $pagination) {
						items {
							id
							status
						}
						totalCount
						pageInfo {
							hasNextPage
							hasPreviousPage
						}
					}
				}
			}
		`,
		LogListPagination: `
			query($connectionId: ID!, $jobId: ID, $pagination: PaginationInput) {
				log {
					list(connectionId: $connectionId, jobId: $jobId, pagination: $pagination) {
						items {
							id
							level
						}
						totalCount
						pageInfo {
							hasNextPage
							hasPreviousPage
						}
					}
				}
			}
		`,
		ConnectionGet: `
			query($id: ID!) {
				connection {
					get(id: $id) {
						id
						name
					}
				}
			}
		`,
		TaskGet: `
			query($id: ID!) {
				task {
					get(id: $id) {
						id
						name
					}
				}
			}
		`,
	}
}
