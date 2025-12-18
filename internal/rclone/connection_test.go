package rclone_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

func TestTestRemote_Success(t *testing.T) {
	setupTestConfig(t)

	ctx := context.Background()

	// Test with memory provider
	params := map[string]string{}

	err := rclone.TestRemote(ctx, "memory", params)
	require.NoError(t, err)
}

func TestTestRemote_InvalidProvider(t *testing.T) {
	setupTestConfig(t)

	ctx := context.Background()

	err := rclone.TestRemote(ctx, "non-existent-provider", map[string]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTestRemote_InvalidParams(t *testing.T) {
	setupTestConfig(t)

	ctx := context.Background()

	// Test with invalid parameters for a provider that requires specific params
	params := map[string]string{
		"invalid_param": "value",
	}

	err := rclone.TestRemote(ctx, "s3", params)
	// Should error because required parameters are missing
	assert.Error(t, err)
}

func TestListRemoteDir(t *testing.T) {
	setupTestConfig(t)

	// Create a temporary directory with subdirectories and files
	tempDir := t.TempDir()

	// Create directory structure:
	// tempDir/
	//   dir1/
	//   dir2/
	//   dir3/
	//   file1.txt (file, should not be listed as directory)
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "dir1"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "dir2"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "dir3"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content"), 0644))

	// Create a local remote pointing to tempDir
	remoteName := "test-list-dir"
	err := createRemote(remoteName, map[string]string{
		"type": "local",
	})
	require.NoError(t, err)
	defer deleteRemote(remoteName)
	ctx := context.Background()

	// List root directory
	entries, err := rclone.ListRemoteDir(ctx, remoteName, tempDir)
	require.NoError(t, err)

	// Should find 3 directories (dir1, dir2, dir3)
	// Files should not be included (ListRemoteDir filters directories only)
	assert.Len(t, entries, 3, "Should list 3 directories")

	// Verify directory names
	dirNames := make(map[string]bool)
	for _, entry := range entries {
		assert.True(t, entry.IsDir, "Entry should be a directory")
		dirNames[entry.Name] = true
	}

	assert.True(t, dirNames["dir1"], "Should contain dir1")
	assert.True(t, dirNames["dir2"], "Should contain dir2")
	assert.True(t, dirNames["dir3"], "Should contain dir3")
}

func TestListRemoteDir_InvalidRemote(t *testing.T) {
	setupTestConfig(t)

	ctx := context.Background()

	// Test with non-existent remote
	_, err := rclone.ListRemoteDir(ctx, "non-existent", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create filesystem")
}

func TestListRemoteDir_InvalidPath(t *testing.T) {
	setupTestConfig(t)

	// Create a memory remote
	remoteName := "test-invalid-path"
	err := createRemote(remoteName, map[string]string{
		"type": "memory",
	})
	require.NoError(t, err)
	defer deleteRemote(remoteName)
	ctx := context.Background()

	// Memory backend returns error for non-existent paths
	_, err = rclone.ListRemoteDir(ctx, remoteName, "non-existent-path")
	// Memory backend errors on non-existent paths
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "directory not found")
}
