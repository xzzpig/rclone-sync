package rclone_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

func TestGetRemoteQuota(t *testing.T) {
	setupTestConfig(t)

	ctx := context.Background()

	t.Run("success with local filesystem", func(t *testing.T) {
		// Create a local remote
		remoteName := "test-quota-local"
		err := createRemote(remoteName, map[string]string{
			"type": "local",
		})
		require.NoError(t, err)
		defer deleteRemote(remoteName)

		// Get quota information
		quota, err := rclone.GetRemoteQuota(ctx, remoteName)
		require.NoError(t, err)
		assert.NotNil(t, quota)

		// Local filesystem should report some usage information
		// We can't assert exact values, but we can check the structure
		assert.NotNil(t, quota)
	})

	t.Run("error with non-existent remote", func(t *testing.T) {
		// Try to get quota for non-existent remote
		_, err := rclone.GetRemoteQuota(ctx, "non-existent-remote")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create fs")
	})

	t.Run("error with backend that doesn't support About", func(t *testing.T) {
		// Memory backend doesn't support the About interface
		remoteName := "test-quota-memory"
		err := createRemote(remoteName, map[string]string{
			"type": "memory",
		})
		require.NoError(t, err)
		defer deleteRemote(remoteName)
		// Memory backend doesn't implement Abouter interface
		_, err = rclone.GetRemoteQuota(ctx, remoteName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not support quota information")
	})

	t.Run("success case structure validation", func(t *testing.T) {
		// Create a local remote
		remoteName := "test-quota-structure"
		err := createRemote(remoteName, map[string]string{
			"type": "local",
		})
		require.NoError(t, err)
		defer deleteRemote(remoteName)
		// Get quota information
		quota, err := rclone.GetRemoteQuota(ctx, remoteName)
		require.NoError(t, err)

		// Verify the AboutInfo structure is returned correctly
		// The exact values depend on the filesystem, but we can verify it's not nil
		assert.NotNil(t, quota)

		// For local filesystem, some fields might be populated
		// We just verify the function returns without panic and has correct structure
	})

	t.Run("uses cached Fs instance", func(t *testing.T) {
		// Create a local remote for testing cache behavior
		remoteName := "test-quota-cache"
		err := createRemote(remoteName, map[string]string{
			"type": "local",
		})
		require.NoError(t, err)
		defer func() {
			rclone.ClearFsCache(remoteName)
			deleteRemote(remoteName)
		}()

		// Clear any existing cache first
		rclone.ClearFsCache(remoteName)

		// Verify that the Fs is not yet loaded in cache
		assert.False(t, rclone.IsConnectionLoaded(remoteName, ""),
			"Fs should not be cached before GetRemoteQuota call")

		// Call GetRemoteQuota which should cache the Fs
		quota, err := rclone.GetRemoteQuota(ctx, remoteName)
		require.NoError(t, err)
		assert.NotNil(t, quota)

		// After GetRemoteQuota call, the Fs should be cached
		assert.True(t, rclone.IsConnectionLoaded(remoteName, ""),
			"Fs should be cached after GetRemoteQuota call")

		// Call GetRemoteQuota again - should reuse cached Fs
		quota2, err := rclone.GetRemoteQuota(ctx, remoteName)
		require.NoError(t, err)
		assert.NotNil(t, quota2)

		// Fs should still be cached
		assert.True(t, rclone.IsConnectionLoaded(remoteName, ""),
			"Fs should remain cached after second GetRemoteQuota call")
	})
}
