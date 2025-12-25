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
	assert.Equal(t, "cloud-sync.db", cfg.Database.Path)
	assert.Equal(t, "versioned", cfg.Database.MigrationMode)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, 1000, cfg.Log.MaxLogsPerConnection)
	assert.Equal(t, "0 * * * *", cfg.Log.CleanupSchedule)
	assert.Equal(t, "./app_data", cfg.App.DataDir)
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
max_logs_per_connection = 500
cleanup_schedule = "*/30 * * * *"

[app]
data_dir = "/custom/data"
environment = "development"

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
	assert.Equal(t, 500, cfg.Log.MaxLogsPerConnection)
	assert.Equal(t, "*/30 * * * *", cfg.Log.CleanupSchedule)
	assert.Equal(t, "/custom/data", cfg.App.DataDir)
	assert.Equal(t, "development", cfg.App.Environment)
	assert.Equal(t, "secret-key", cfg.Security.EncryptionKey)
}
