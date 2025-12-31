package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_WithLogLevels(t *testing.T) {
	// Reset viper before each test
	viper.Reset()

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[app]
environment = "test"

[log]
level = "info"

[log.levels]
"core.db" = "debug"
"core.scheduler" = "warn"
"rclone" = "error"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Verify LogLevels are correctly parsed
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Len(t, cfg.Log.Levels, 3)
	assert.Equal(t, "debug", cfg.Log.Levels["core.db"])
	assert.Equal(t, "warn", cfg.Log.Levels["core.scheduler"])
	assert.Equal(t, "error", cfg.Log.Levels["rclone"])
}

func TestLoad_WithNestedLogLevels(t *testing.T) {
	// Reset viper before each test
	viper.Reset()

	// Create a temporary config file with deeply nested structure
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// This tests the case where TOML parser interprets dotted keys as nested maps
	configContent := `
[app]
environment = "test"

[log]
level = "info"

[log.levels]
"a.b.c" = "debug"
"x.y" = "warn"
simple = "error"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Verify deeply nested LogLevels are correctly flattened
	assert.Len(t, cfg.Log.Levels, 3)
	assert.Equal(t, "debug", cfg.Log.Levels["a.b.c"])
	assert.Equal(t, "warn", cfg.Log.Levels["x.y"])
	assert.Equal(t, "error", cfg.Log.Levels["simple"])
}

func TestLoad_WithEmptyLogLevels(t *testing.T) {
	// Reset viper before each test
	viper.Reset()

	// Create a temporary config file without log.levels
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[app]
environment = "test"

[log]
level = "debug"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Empty(t, cfg.Log.Levels)
}

func TestLoad_Defaults(t *testing.T) {
	// Reset viper before each test
	viper.Reset()

	// Create an empty config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	err := os.WriteFile(configPath, []byte(""), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Verify defaults are applied
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, "rclone-sync.db", cfg.Database.Path)
	assert.Equal(t, "versioned", cfg.Database.MigrationMode)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "./app_data", cfg.App.DataDir)
	assert.Equal(t, true, cfg.App.Job.AutoDeleteEmptyJobs)
	assert.Equal(t, 1000, cfg.App.Job.MaxLogsPerConnection)
	assert.Equal(t, "0 * * * *", cfg.App.Job.CleanupSchedule)
	assert.Equal(t, 4, cfg.App.Sync.Transfers)
	assert.Equal(t, "production", cfg.App.Environment)
}

func TestLoad_ConfigFileNotFound(t *testing.T) {
	// Reset viper before each test
	viper.Reset()

	// Load with non-existent file path (empty string triggers default lookup)
	// Using a path that definitely doesn't exist
	cfg, err := Load("/nonexistent/path/config.toml")

	// Should return error for explicit non-existent file
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_WithMixedParentChildLogLevels(t *testing.T) {
	// Reset viper before each test
	viper.Reset()

	// Create a temporary config file with both parent and child keys
	// This tests the case where we have both "api" and "api.graphql"
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// This is the problematic case from config.toml where we have:
	// "api.graphql" = "warn" -> parsed as nested: api: {graphql: "warn"}
	// "api" = "info" -> conflicts with the nested structure
	configContent := `
[app]
environment = "test"

[log]
level = "info"

[log.levels]
"core.db.query" = "debug"
"api.graphql" = "warn"
"api" = "info"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Verify all LogLevels are correctly parsed, including both parent and child
	assert.Len(t, cfg.Log.Levels, 3, "Should have 3 log levels")
	assert.Equal(t, "debug", cfg.Log.Levels["core.db.query"], "core.db.query should be debug")
	assert.Equal(t, "warn", cfg.Log.Levels["api.graphql"], "api.graphql should be warn")
	assert.Equal(t, "info", cfg.Log.Levels["api"], "api should be info")
}

func TestLoad_OverrideDefaults(t *testing.T) {
	// Reset viper before each test
	viper.Reset()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[server]
port = 9000
host = "127.0.0.1"

[database]
path = "custom.db"
migration_mode = "auto"

[log]
level = "debug"

[app]
data_dir = "/custom/data"
environment = "development"

[app.job]
auto_delete_empty_jobs = false
max_logs_per_connection = 500
cleanup_schedule = "*/30 * * * *"

[app.sync]
transfers = 8

[security]
encryption_key = "secret-key"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, 9000, cfg.Server.Port)
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, "custom.db", cfg.Database.Path)
	assert.Equal(t, "auto", cfg.Database.MigrationMode)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "/custom/data", cfg.App.DataDir)
	assert.Equal(t, "development", cfg.App.Environment)
	assert.Equal(t, false, cfg.App.Job.AutoDeleteEmptyJobs)
	assert.Equal(t, 500, cfg.App.Job.MaxLogsPerConnection)
	assert.Equal(t, "*/30 * * * *", cfg.App.Job.CleanupSchedule)
	assert.Equal(t, 8, cfg.App.Sync.Transfers)
	assert.Equal(t, "secret-key", cfg.Security.EncryptionKey)
}

func TestAuthConfig(t *testing.T) {
	tests := []struct {
		name              string
		configContent     string
		expectedUsername  string
		expectedPassword  string
		expectValidateErr bool
		expectAuthEnabled bool
	}{
		{
			name: "Both empty is valid (auth disabled)",
			configContent: `
[auth]
username = ""
password = ""
`,
			expectedUsername:  "",
			expectedPassword:  "",
			expectValidateErr: false,
			expectAuthEnabled: false,
		},
		{
			name: "Both set is valid (auth enabled)",
			configContent: `
[auth]
username = "admin"
password = "secret123"
`,
			expectedUsername:  "admin",
			expectedPassword:  "secret123",
			expectValidateErr: false,
			expectAuthEnabled: true,
		},
		{
			name: "Only username set returns error",
			configContent: `
[auth]
username = "admin"
password = ""
`,
			expectedUsername:  "admin",
			expectedPassword:  "",
			expectValidateErr: true,
			expectAuthEnabled: false,
		},
		{
			name: "Only password set returns error",
			configContent: `
[auth]
username = ""
password = "secret123"
`,
			expectedUsername:  "",
			expectedPassword:  "secret123",
			expectValidateErr: true,
			expectAuthEnabled: false,
		},
		{
			name: "No auth section is valid (defaults to empty, auth disabled)",
			configContent: `
[server]
port = 8080
`,
			expectedUsername:  "",
			expectedPassword:  "",
			expectValidateErr: false,
			expectAuthEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.toml")

			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			require.NoError(t, err)

			cfg, err := Load(configPath)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedUsername, cfg.Auth.Username)
			assert.Equal(t, tt.expectedPassword, cfg.Auth.Password)

			if tt.expectValidateErr {
				assert.Error(t, cfg.ValidateAuth())
				assert.Contains(t, cfg.ValidateAuth().Error(), "username and password must both be set or both be empty")
			} else {
				assert.NoError(t, cfg.ValidateAuth())
			}

			assert.Equal(t, tt.expectAuthEnabled, cfg.IsAuthEnabled())
		})
	}
}

func TestAuthConfig_EnvironmentVariablesOverride(t *testing.T) {
	viper.Reset()

	// Set environment variables
	t.Setenv("RCLONESYNC_AUTH_USERNAME", "envuser")
	t.Setenv("RCLONESYNC_AUTH_PASSWORD", "envpass")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Config file has different values
	configContent := `
[auth]
username = "configuser"
password = "configpass"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Environment variables should override config file
	assert.Equal(t, "envuser", cfg.Auth.Username)
	assert.Equal(t, "envpass", cfg.Auth.Password)
	assert.NoError(t, cfg.ValidateAuth())
	assert.True(t, cfg.IsAuthEnabled())
}
