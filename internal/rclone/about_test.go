package rclone_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

func TestGetRemoteQuota_Success(t *testing.T) {
	// Test with local backend which supports About
	// Note: We need to create a configured remote, not just use a path
	setupTestConfig(t)

	remoteName := "test-local-quota"
	err := rclone.CreateRemote(remoteName, map[string]string{
		"type": "local",
	})
	require.NoError(t, err)
	defer rclone.DeleteRemote(remoteName)

	ctx := context.Background()

	// Test: Get quota information for local backend
	aboutInfo, err := rclone.GetRemoteQuota(ctx, remoteName)

	// Local backend supports About interface
	require.NoError(t, err, "Local backend should support About")
	assert.NotNil(t, aboutInfo, "AboutInfo should not be nil")

	// Verify that at least one field is populated
	// Different systems may populate different fields
	hasData := aboutInfo.Total != nil ||
		aboutInfo.Used != nil ||
		aboutInfo.Free != nil ||
		aboutInfo.Objects != nil
	assert.True(t, hasData, "At least one quota field should be populated")
}

func TestGetRemoteQuota_InvalidRemote(t *testing.T) {
	setupTestConfig(t)

	ctx := context.Background()
	_, err := rclone.GetRemoteQuota(ctx, "non-existent-remote")

	// Should return error for non-existent remote
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create fs")
}

func TestGetRemoteQuota_UnsupportedBackend(t *testing.T) {
	// Test with memory backend which doesn't support About
	setupTestConfig(t)

	remoteName := "test-no-about"
	err := rclone.CreateRemote(remoteName, map[string]string{
		"type": "memory",
	})
	require.NoError(t, err)
	defer rclone.DeleteRemote(remoteName)

	ctx := context.Background()
	_, err = rclone.GetRemoteQuota(ctx, remoteName)

	// Memory backend doesn't support About, should return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support quota information")
}
