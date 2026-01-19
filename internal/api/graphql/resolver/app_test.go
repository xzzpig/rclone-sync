// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/resolver"
	"github.com/xzzpig/rclone-sync/internal/core/config"
)

// AppResolverTestSuite tests AppConfig query resolvers.
type AppResolverTestSuite struct {
	ResolverTestSuite
}

func TestAppResolverSuite(t *testing.T) {
	suite.Run(t, new(AppResolverTestSuite))
}

// TestAppConfig_DefaultHookEnabled tests AppConfig when deps.Config is nil (defaults to enabled=true).
func (s *AppResolverTestSuite) TestAppConfig_DefaultHookEnabled() {
	// The default test env has nil config, so hook.enabled should be true
	query := `
		query {
			appConfig {
				hook {
					enabled
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{Query: query})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.True(s.T(), gjson.Get(data, "appConfig.hook.enabled").Bool())
}

// TestAppConfig_WithConfig tests AppConfig with explicit config.
func TestAppConfig_WithConfig(t *testing.T) {
	// Create a test environment with explicit config
	env := NewTestEnv(t)
	t.Cleanup(env.Cleanup)

	// Create a config with hook disabled
	cfg := &config.Config{}
	cfg.App.Hook.Enabled = false
	env.Deps.Config = cfg

	query := `
		query {
			appConfig {
				hook {
					enabled
				}
			}
		}
	`

	resp := env.ExecuteGraphQL(t, GraphQLRequest{Query: query})
	require.Empty(t, resp.Errors)

	data := string(resp.Data)
	assert.False(t, gjson.Get(data, "appConfig.hook.enabled").Bool())
}

// TestAppConfig_WithHookEnabled tests AppConfig with hook enabled in config.
func TestAppConfig_WithHookEnabled(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(env.Cleanup)

	cfg := &config.Config{}
	cfg.App.Hook.Enabled = true
	env.Deps.Config = cfg

	query := `
		query {
			appConfig {
				hook {
					enabled
				}
			}
		}
	`

	resp := env.ExecuteGraphQL(t, GraphQLRequest{Query: query})
	require.Empty(t, resp.Errors)

	data := string(resp.Data)
	assert.True(t, gjson.Get(data, "appConfig.hook.enabled").Bool())
}

// Test that Dependencies struct can hold a Config.
func TestDependencies_HasConfigField(t *testing.T) {
	deps := &resolver.Dependencies{}
	cfg := &config.Config{}
	cfg.App.Hook.Enabled = true
	deps.Config = cfg
	assert.True(t, deps.Config.App.Hook.Enabled)
}
