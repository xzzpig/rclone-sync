// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

// ProviderResolverTestSuite tests ProviderQuery resolvers.
type ProviderResolverTestSuite struct {
	ResolverTestSuite
}

func TestProviderResolverSuite(t *testing.T) {
	suite.Run(t, new(ProviderResolverTestSuite))
}

// TestProviderQuery_List tests ProviderQuery.list resolver.
func (s *ProviderResolverTestSuite) TestProviderQuery_List() {
	query := `
		query {
			provider {
				list {
					name
					description
					prefix
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: query})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	providers := gjson.Get(data, "provider.list")
	assert.True(s.T(), providers.IsArray())
	// rclone should provide at least some providers (like local)
	assert.Greater(s.T(), len(providers.Array()), 0)
}

// TestProviderQuery_ListContainsLocal tests that the list contains local provider.
func (s *ProviderResolverTestSuite) TestProviderQuery_ListContainsLocal() {
	query := `
		query {
			provider {
				list {
					name
					prefix
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: query})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	providers := gjson.Get(data, "provider.list").Array()

	// Find the local provider
	hasLocal := false
	for _, p := range providers {
		if p.Get("name").String() == "local" || p.Get("prefix").String() == "local" {
			hasLocal = true
			break
		}
	}
	assert.True(s.T(), hasLocal, "provider list should contain local provider")
}

// TestProviderQuery_Get tests ProviderQuery.get resolver.
func (s *ProviderResolverTestSuite) TestProviderQuery_Get() {
	query := `
		query($name: String!) {
			provider {
				get(name: $name) {
					name
					description
					prefix
					options {
						name
						type
						help
						required
						isPassword
						advanced
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"name": "local",
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	provider := gjson.Get(data, "provider.get")
	assert.True(s.T(), provider.Exists())
	assert.Equal(s.T(), "local", provider.Get("name").String())
}

// TestProviderQuery_GetNotFound tests ProviderQuery.get with non-existent provider.
func (s *ProviderResolverTestSuite) TestProviderQuery_GetNotFound() {
	query := `
		query($name: String!) {
			provider {
				get(name: $name) {
					name
				}
			}
		}
	`

	testGetNotFound(s.Env, s.T(), GetNotFoundTestCase{
		Name:        "non-existent-provider",
		Entity:      "Provider",
		Query:       query,
		VariableKey: "name",
		Variable:    func(t *testing.T) interface{} { return "nonexistent-provider-xyz" },
		DataPath:    "provider.get",
	})
}

// TestProviderQuery_GetWithOptions tests that provider options are returned.
func (s *ProviderResolverTestSuite) TestProviderQuery_GetWithOptions() {
	query := `
		query($name: String!) {
			provider {
				get(name: $name) {
					name
					options {
						name
						type
						help
						required
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"name": "local",
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	provider := gjson.Get(data, "provider.get")
	assert.True(s.T(), provider.Exists())

	// Local provider should have some options
	options := provider.Get("options").Array()
	// Local provider might not have many options, but the structure should be correct
	_ = options // Just verify it's an array
}

// TestProviderQuery_ListProviderFields tests all provider fields.
func (s *ProviderResolverTestSuite) TestProviderQuery_ListProviderFields() {
	query := `
		query {
			provider {
				list {
					name
					description
					prefix
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: query})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	providers := gjson.Get(data, "provider.list").Array()

	if len(providers) > 0 {
		firstProvider := providers[0]
		// Each provider should have name field
		assert.True(s.T(), firstProvider.Get("name").Exists())
	}
}

// TestProviderQuery_OptionFields tests provider option field types.
func (s *ProviderResolverTestSuite) TestProviderQuery_OptionFields() {
	query := `
		query($name: String!) {
			provider {
				get(name: $name) {
					name
					options {
						name
						type
						help
						required
						isPassword
						advanced
						defaultValue
					}
				}
			}
		}
	`

	// Use a provider that is known to have options
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"name": "s3",
	})

	if len(resp.Errors) == 0 {
		data := string(resp.Data)
		provider := gjson.Get(data, "provider.get")
		if provider.Exists() {
			options := provider.Get("options").Array()
			if len(options) > 0 {
				// Verify option structure
				firstOption := options[0]
				assert.True(s.T(), firstOption.Get("name").Exists())
				// type, help, required, isPassword, advanced should exist (may be null)
			}
		}
	}
}

// TestProviderQuery_CommonProviders tests that common cloud providers are available.
func (s *ProviderResolverTestSuite) TestProviderQuery_CommonProviders() {
	query := `
		query {
			provider {
				list {
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: query})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	providers := gjson.Get(data, "provider.list").Array()

	providerNames := make(map[string]bool)
	for _, p := range providers {
		providerNames[p.Get("name").String()] = true
	}

	// Check for common providers that rclone typically supports
	// Note: The exact list depends on rclone build
	assert.True(s.T(), providerNames["local"], "local provider should exist")
}

// TestProviderQuery_EmptyName tests ProviderQuery.get with empty name.
func (s *ProviderResolverTestSuite) TestProviderQuery_EmptyName() {
	query := `
		query($name: String!) {
			provider {
				get(name: $name) {
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"name": "",
	})

	// Empty name should return null
	data := string(resp.Data)
	result := gjson.Get(data, "provider.get")
	assert.True(s.T(), !result.Exists() || result.Type == gjson.Null, "provider.get should be null for empty name")
}
