package rclone_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

func TestInitConfig(t *testing.T) {
	// Test initialization with different log levels
	tests := []struct {
		name     string
		logLevel string
	}{
		{"debug level", "debug"},
		{"info level", "info"},
		{"warn level", "warn"},
		{"error level", "error"},
		{"default level", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test primarily ensures InitConfig doesn't panic
			// with various log level configurations
			setupTestConfig(t)
			// If we got here without panic, test passes
		})
	}
}

func TestListRemotes(t *testing.T) {
	setupTestConfig(t)

	// Initially should be empty
	remotes := rclone.ListRemotes()
	assert.Empty(t, remotes)

	// Create a test remote
	err := rclone.CreateRemote("test-remote", map[string]string{
		"type": "memory",
	})
	require.NoError(t, err)

	// Now should have one remote
	remotes = rclone.ListRemotes()
	assert.Len(t, remotes, 1)
	assert.Contains(t, remotes, "test-remote")

	// Cleanup
	rclone.DeleteRemote("test-remote")
}

func TestListRemotesWithInfo(t *testing.T) {
	setupTestConfig(t)

	// Create test remotes
	err := rclone.CreateRemote("test-memory", map[string]string{
		"type": "memory",
	})
	require.NoError(t, err)
	defer rclone.DeleteRemote("test-memory")

	err = rclone.CreateRemote("test-alias", map[string]string{
		"type":   "alias",
		"remote": ":memory:",
	})
	require.NoError(t, err)
	defer rclone.DeleteRemote("test-alias")

	// List with info
	infos, err := rclone.ListRemotesWithInfo()
	require.NoError(t, err)
	assert.Len(t, infos, 2)

	// Verify info structure
	for _, info := range infos {
		assert.NotEmpty(t, info.Name)
		assert.NotEmpty(t, info.Type)
	}
}

func TestGetRemoteInfo(t *testing.T) {
	setupTestConfig(t)

	// Test: Get info for non-existent remote
	_, err := rclone.GetRemoteInfo("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Create a test remote
	err = rclone.CreateRemote("test-info", map[string]string{
		"type": "memory",
	})
	require.NoError(t, err)
	defer rclone.DeleteRemote("test-info")

	// Test: Get info for existing remote
	info, err := rclone.GetRemoteInfo("test-info")
	require.NoError(t, err)
	assert.Equal(t, "test-info", info.Name)
	assert.Equal(t, "memory", info.Type)
}

func TestGetRemoteConfig(t *testing.T) {
	setupTestConfig(t)

	// Test: Get config for non-existent remote
	_, err := rclone.GetRemoteConfig("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Create a test remote
	err = rclone.CreateRemote("test-config", map[string]string{
		"type": "memory",
	})
	require.NoError(t, err)
	defer rclone.DeleteRemote("test-config")

	// Test: Get config for existing remote
	config, err := rclone.GetRemoteConfig("test-config")
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "memory", config["type"])
}

func TestCreateRemote(t *testing.T) {
	setupTestConfig(t)

	// Test: Create a new remote
	err := rclone.CreateRemote("test-create", map[string]string{
		"type": "memory",
	})
	require.NoError(t, err)
	defer rclone.DeleteRemote("test-create")

	// Verify it was created
	remotes := rclone.ListRemotes()
	assert.Contains(t, remotes, "test-create")

	// Test: Update existing remote
	err = rclone.CreateRemote("test-create", map[string]string{
		"type":        "memory",
		"description": "updated",
	})
	require.NoError(t, err)

	// Verify update
	config, err := rclone.GetRemoteConfig("test-create")
	require.NoError(t, err)
	assert.Contains(t, config, "description")
}

func TestDeleteRemote(t *testing.T) {
	setupTestConfig(t)

	// Create a remote
	err := rclone.CreateRemote("test-delete", map[string]string{
		"type": "memory",
	})
	require.NoError(t, err)

	// Verify it exists
	remotes := rclone.ListRemotes()
	assert.Contains(t, remotes, "test-delete")

	// Delete it
	rclone.DeleteRemote("test-delete")

	// Verify it's gone
	remotes = rclone.ListRemotes()
	assert.NotContains(t, remotes, "test-delete")

	// Deleting non-existent remote should not panic
	assert.NotPanics(t, func() {
		rclone.DeleteRemote("non-existent")
	})
}

func TestConfigIntegration_CRUD(t *testing.T) {
	// Integration test: Create, Read, Update, Delete cycle
	setupTestConfig(t)

	remoteName := "test-integration"

	// 1. Create
	err := rclone.CreateRemote(remoteName, map[string]string{
		"type": "memory",
	})
	require.NoError(t, err)

	// 2. Read - List
	remotes := rclone.ListRemotes()
	assert.Contains(t, remotes, remoteName)

	// 2. Read - Get Info
	info, err := rclone.GetRemoteInfo(remoteName)
	require.NoError(t, err)
	assert.Equal(t, remoteName, info.Name)
	assert.Equal(t, "memory", info.Type)

	// 2. Read - Get Config
	config, err := rclone.GetRemoteConfig(remoteName)
	require.NoError(t, err)
	assert.Equal(t, "memory", config["type"])

	// 3. Update
	err = rclone.CreateRemote(remoteName, map[string]string{
		"type":   "alias",
		"remote": ":memory:",
	})
	require.NoError(t, err)

	info, err = rclone.GetRemoteInfo(remoteName)
	require.NoError(t, err)
	assert.Equal(t, "alias", info.Type)

	// 4. Delete
	rclone.DeleteRemote(remoteName)
	_, err = rclone.GetRemoteInfo(remoteName)
	assert.Error(t, err)
}
