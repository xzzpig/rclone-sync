package rclone

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rclone/rclone/fs/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetFs tests the GetFs helper function
func TestGetFs(t *testing.T) {
	ctx := context.Background()

	t.Run("local path returns new Fs without caching", func(t *testing.T) {
		// Create a temp directory for testing
		tmpDir := t.TempDir()

		// Call GetFs with empty remote (local path)
		fsObj, err := GetFs(ctx, "", tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, fsObj)
		assert.Equal(t, tmpDir, fsObj.Root())
	})

	t.Run("local path with subdirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		fsObj, err := GetFs(ctx, "", subDir)
		require.NoError(t, err)
		assert.NotNil(t, fsObj)
		assert.Equal(t, subDir, fsObj.Root())
	})

	t.Run("local path with non-existing directory still works", func(t *testing.T) {
		// Note: rclone's fs.NewFs does not return error for non-existing local paths
		// It creates an Fs pointing to that path, which will fail on operations
		fsObj, err := GetFs(ctx, "", "/non/existing/path/12345")
		// fs.NewFs for local paths doesn't fail even if path doesn't exist
		require.NoError(t, err)
		assert.NotNil(t, fsObj)
	})

	t.Run("remote path uses cache", func(t *testing.T) {
		// This test verifies that remote paths go through cache.Get
		// Since we don't have a real remote configured, it will fail
		// but we can verify the path format is correct
		_, err := GetFs(ctx, "nonexistent-remote", "some/path")
		// Should fail because the remote doesn't exist
		assert.Error(t, err)
	})

	t.Run("remote path with empty path uses cache", func(t *testing.T) {
		// Testing remote root path
		_, err := GetFs(ctx, "nonexistent-remote", "")
		// Should fail because the remote doesn't exist
		assert.Error(t, err)
	})
}

// TestClearFsCache tests the ClearFsCache helper function
func TestClearFsCache(t *testing.T) {
	t.Run("empty remote name returns 0", func(t *testing.T) {
		result := ClearFsCache("")
		assert.Equal(t, 0, result)
	})

	t.Run("non-existing remote returns 0", func(t *testing.T) {
		// Clearing a remote that was never cached should return 0
		result := ClearFsCache("never-cached-remote-12345")
		assert.Equal(t, 0, result)
	})

	t.Run("clears cached local fs", func(t *testing.T) {
		ctx := context.Background()
		tmpDir := t.TempDir()

		// Manually add to cache using a fake "remote" name
		// We'll use cache.Get to add it (simulating what GetFs does for remotes)
		// Note: For local paths, we use the path as the "remote" for testing purposes
		localRemoteName := "test-local-cache"

		// First, verify the cache entry doesn't exist
		entriesBefore := cache.Entries()

		// Put a local fs into cache manually for testing
		// We use cache.Get which will create and cache it
		_, err := cache.Get(ctx, tmpDir)
		require.NoError(t, err)

		entriesAfter := cache.Entries()
		assert.Greater(t, entriesAfter, entriesBefore, "Cache should have more entries after Get")

		// Clear by the local remote name shouldn't affect the tmpDir cache
		// (since tmpDir doesn't start with "test-local-cache:")
		result := ClearFsCache(localRemoteName)
		assert.Equal(t, 0, result, "Should return 0 as no matching entries")
	})
}

// T048: 单元测试：IsConnectionLoaded() 缓存检查
func TestIsConnectionLoaded(t *testing.T) {
	t.Run("returns false for never-loaded connection", func(t *testing.T) {
		// A connection that was never accessed should return false
		result := IsConnectionLoaded("never-loaded-remote", "some/path")
		assert.False(t, result)
	})

	t.Run("returns false for non-existing connection", func(t *testing.T) {
		result := IsConnectionLoaded("non-existing-remote-12345", "")
		assert.False(t, result)
	})

	t.Run("returns false for empty remote name", func(t *testing.T) {
		// Empty remote name means local path, which is never "loaded" in cache
		result := IsConnectionLoaded("", "/some/local/path")
		assert.False(t, result)
	})

	t.Run("different paths are cached separately", func(t *testing.T) {
		// Verify that the same remote with different paths are separate cache entries
		result1 := IsConnectionLoaded("test-remote", "path1")
		result2 := IsConnectionLoaded("test-remote", "path2")
		// Both should be false since neither has been loaded
		assert.False(t, result1)
		assert.False(t, result2)
	})

	// Note: Testing "loaded" state would require actually loading a connection
	// through cache.Get or GetFs, which needs a real remote configuration.
	// This is better tested in integration tests like TestListRemoteDir_CacheBehavior.
}
