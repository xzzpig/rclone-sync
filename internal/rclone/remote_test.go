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
	assert.Contains(t, err.Error(), "error_provider_not_found")
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
	entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
		RemoteName: remoteName,
		Path:       tempDir,
	})
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
	_, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
		RemoteName: "non-existent",
		Path:       "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error_path_not_exist")
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
	_, err = rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
		RemoteName: remoteName,
		Path:       "non-existent-path",
	})
	// Memory backend errors on non-existent paths
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "directory not found")
}

func TestListRemoteDir_Options(t *testing.T) {
	setupTestConfig(t)

	// Create a temporary directory with subdirectories and files
	tempDir := t.TempDir()

	// Create directory structure:
	// tempDir/
	//   dir1/
	//   dir2/
	//   include.txt
	//   exclude.tmp
	//   data.json
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "dir1"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "dir2"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "include.txt"), []byte("content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "exclude.tmp"), []byte("temp"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "data.json"), []byte("{}"), 0644))

	// Create a local remote pointing to tempDir
	remoteName := "test-list-opts"
	err := createRemote(remoteName, map[string]string{
		"type": "local",
	})
	require.NoError(t, err)
	defer deleteRemote(remoteName)

	ctx := context.Background()

	t.Run("directories only (default)", func(t *testing.T) {
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName: remoteName,
			Path:       tempDir,
		})
		require.NoError(t, err)

		// Should only return directories
		assert.Len(t, entries, 2)
		for _, entry := range entries {
			assert.True(t, entry.IsDir, "Entry should be a directory")
		}
	})

	t.Run("include files", func(t *testing.T) {
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName:   remoteName,
			Path:         tempDir,
			IncludeFiles: true,
		})
		require.NoError(t, err)

		// Should return 2 directories + 3 files = 5 entries
		assert.Len(t, entries, 5)

		dirCount := 0
		fileCount := 0
		for _, entry := range entries {
			if entry.IsDir {
				dirCount++
			} else {
				fileCount++
			}
		}
		assert.Equal(t, 2, dirCount, "Should have 2 directories")
		assert.Equal(t, 3, fileCount, "Should have 3 files")
	})

	t.Run("filter exclude .tmp files", func(t *testing.T) {
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName:   remoteName,
			Path:         tempDir,
			IncludeFiles: true,
			Filters:      []string{"- *.tmp", "+ **"},
		})
		require.NoError(t, err)

		// Should return 2 directories + 2 files (exclude.tmp excluded) = 4 entries
		assert.Len(t, entries, 4)

		for _, entry := range entries {
			assert.NotContains(t, entry.Name, ".tmp", "Should not contain .tmp files")
		}
	})

	t.Run("filter include only .txt files", func(t *testing.T) {
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName:   remoteName,
			Path:         tempDir,
			IncludeFiles: true,
			Filters:      []string{"+ *.txt", "+ */", "- **"},
		})
		require.NoError(t, err)

		// Note: `+ */` pattern doesn't match top-level directories because their Remote() names
		// don't end with `/`. This is expected rclone filter behavior.
		// Only .txt files should be included.
		assert.Len(t, entries, 1, "Should return only .txt files")

		for _, entry := range entries {
			assert.False(t, entry.IsDir, "Entry should be a file")
			assert.Contains(t, entry.Name, ".txt", "File should be .txt")
		}
	})

	t.Run("filter include directories and .txt files", func(t *testing.T) {
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName:   remoteName,
			Path:         tempDir,
			IncludeFiles: true,
			// Use ** pattern to include all directories at this level
			Filters: []string{"+ *.txt", "+ dir*", "- **"},
		})
		require.NoError(t, err)

		// Should return 2 directories (dir1, dir2) + 1 .txt file = 3 entries
		assert.Len(t, entries, 3)

		dirCount := 0
		fileCount := 0
		for _, entry := range entries {
			if entry.IsDir {
				dirCount++
				assert.True(t, entry.Name == "dir1" || entry.Name == "dir2", "Directory should be dir1 or dir2")
			} else {
				fileCount++
				assert.Contains(t, entry.Name, ".txt", "File should be .txt")
			}
		}
		assert.Equal(t, 2, dirCount, "Should have 2 directories")
		assert.Equal(t, 1, fileCount, "Should have 1 .txt file")
	})

	t.Run("invalid filter rule", func(t *testing.T) {
		_, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName: remoteName,
			Path:       tempDir,
			Filters:    []string{"invalid rule without prefix"},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error_filter_rule_invalid")
	})

	t.Run("invalid remote", func(t *testing.T) {
		_, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName: "non-existent-remote",
			Path:       tempDir,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error_path_not_exist")
	})
}

func TestListRemoteDir_BasePath(t *testing.T) {
	setupTestConfig(t)

	// Create a temporary directory with nested structure to test basePath functionality
	// This simulates the real use case where:
	// - Sync task root path is /root/base (stored as task.remotePath)
	// - User browses subdirectory /root/base/subdir in the preview
	// - Filter rules are written relative to root path (e.g., "- subdir/file1.txt")
	tempDir := t.TempDir()

	// Create directory structure:
	// tempDir/
	//   base/                    <- basePath (sync task root)
	//     file_at_root.txt
	//     subdir/                <- browsing this subdirectory
	//       file1.txt            <- should be filtered by "- subdir/file1.txt"
	//       file2.txt            <- should pass filter
	//       nested/
	//         deep.txt
	//     other/
	//       file3.txt
	basePath := filepath.Join(tempDir, "base")
	subdirPath := filepath.Join(basePath, "subdir")
	nestedPath := filepath.Join(subdirPath, "nested")
	otherPath := filepath.Join(basePath, "other")

	require.NoError(t, os.MkdirAll(nestedPath, 0755))
	require.NoError(t, os.MkdirAll(otherPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(basePath, "file_at_root.txt"), []byte("root"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(subdirPath, "file1.txt"), []byte("file1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(subdirPath, "file2.txt"), []byte("file2"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(nestedPath, "deep.txt"), []byte("deep"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(otherPath, "file3.txt"), []byte("file3"), 0644))

	// Create a local remote
	remoteName := "test-basepath"
	err := createRemote(remoteName, map[string]string{
		"type": "local",
	})
	require.NoError(t, err)
	defer deleteRemote(remoteName)

	ctx := context.Background()

	t.Run("basePath equals path - filter applies directly", func(t *testing.T) {
		// When browsing the root directory (basePath == path), filter applies directly to entries
		// Filter: exclude file_at_root.txt
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName:   remoteName,
			Path:         basePath,
			BasePath:     basePath,
			IncludeFiles: true,
			Filters:      []string{"- file_at_root.txt", "+ **"},
		})
		require.NoError(t, err)

		// Should have: subdir/, other/ (2 dirs), but NOT file_at_root.txt
		assert.Len(t, entries, 2, "Should have 2 entries (2 directories, 0 files)")
		for _, entry := range entries {
			assert.True(t, entry.IsDir, "All entries should be directories")
			assert.NotEqual(t, "file_at_root.txt", entry.Name, "file_at_root.txt should be filtered out")
		}
	})

	t.Run("basePath with subdirectory - filter path prefix calculated", func(t *testing.T) {
		// When browsing a subdirectory, filter paths need prefix calculation
		// BasePath: /base, Path: /base/subdir
		// Filter: "- subdir/file1.txt" should match "subdir/" + "file1.txt"
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName:   remoteName,
			Path:         subdirPath,
			BasePath:     basePath,
			IncludeFiles: true,
			Filters:      []string{"- subdir/file1.txt", "+ **"},
		})
		require.NoError(t, err)

		// Should have: nested/ (1 dir), file2.txt (1 file), but NOT file1.txt
		assert.Len(t, entries, 2, "Should have 2 entries (1 directory, 1 file)")

		names := make(map[string]bool)
		for _, entry := range entries {
			names[entry.Name] = true
		}
		assert.True(t, names["nested"], "Should contain nested directory")
		assert.True(t, names["file2.txt"], "Should contain file2.txt")
		assert.False(t, names["file1.txt"], "file1.txt should be filtered out by subdir/file1.txt rule")
	})

	t.Run("basePath with subdirectory - exclude entire subdirectory", func(t *testing.T) {
		// Filter: "- subdir/**" should exclude all files in subdir when browsing subdir
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName:   remoteName,
			Path:         subdirPath,
			BasePath:     basePath,
			IncludeFiles: true,
			Filters:      []string{"- subdir/**", "+ **"},
		})
		require.NoError(t, err)

		// All files and directories in subdir should be excluded
		assert.Len(t, entries, 0, "All entries should be filtered out by subdir/** rule")
	})

	t.Run("basePath with nested subdirectory - deep path prefix", func(t *testing.T) {
		// When browsing nested directory: Path=/base/subdir/nested, BasePath=/base
		// Filter: "- subdir/nested/deep.txt" should work
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName:   remoteName,
			Path:         nestedPath,
			BasePath:     basePath,
			IncludeFiles: true,
			Filters:      []string{"- subdir/nested/deep.txt", "+ **"},
		})
		require.NoError(t, err)

		// Should have no files (deep.txt excluded), empty result
		assert.Len(t, entries, 0, "deep.txt should be filtered out")
	})

	t.Run("basePath empty - defaults to path behavior", func(t *testing.T) {
		// When basePath is empty, filter applies directly to entries (no prefix)
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName:   remoteName,
			Path:         subdirPath,
			BasePath:     "", // empty basePath
			IncludeFiles: true,
			Filters:      []string{"- file1.txt", "+ **"},
		})
		require.NoError(t, err)

		// Should exclude file1.txt directly (no prefix calculation)
		assert.Len(t, entries, 2, "Should have 2 entries")

		names := make(map[string]bool)
		for _, entry := range entries {
			names[entry.Name] = true
		}
		assert.True(t, names["nested"], "Should contain nested directory")
		assert.True(t, names["file2.txt"], "Should contain file2.txt")
		assert.False(t, names["file1.txt"], "file1.txt should be filtered out")
	})

	t.Run("basePath with trailing slash - normalized correctly", func(t *testing.T) {
		// Trailing slashes should be handled correctly
		entries, err := rclone.ListRemoteDir(ctx, rclone.ListRemoteDirOptions{
			RemoteName:   remoteName,
			Path:         subdirPath + "/",
			BasePath:     basePath + "/",
			IncludeFiles: true,
			Filters:      []string{"- subdir/file1.txt", "+ **"},
		})
		require.NoError(t, err)

		// Same result as without trailing slashes
		assert.Len(t, entries, 2, "Should have 2 entries")

		names := make(map[string]bool)
		for _, entry := range entries {
			names[entry.Name] = true
		}
		assert.False(t, names["file1.txt"], "file1.txt should be filtered out")
	})
}

func TestCalculateFilterPrefix(t *testing.T) {
	tests := []struct {
		name        string
		basePath    string
		currentPath string
		expected    string
	}{
		{
			name:        "empty basePath returns empty",
			basePath:    "",
			currentPath: "a/b/c",
			expected:    "",
		},
		{
			name:        "same path returns empty",
			basePath:    "a/b/c",
			currentPath: "a/b/c",
			expected:    "",
		},
		{
			name:        "same path with trailing slashes",
			basePath:    "a/b/c/",
			currentPath: "a/b/c/",
			expected:    "",
		},
		{
			name:        "currentPath is subdirectory of basePath",
			basePath:    "a/b/c",
			currentPath: "a/b/c/xxx",
			expected:    "xxx/",
		},
		{
			name:        "currentPath is nested subdirectory",
			basePath:    "a/b/c",
			currentPath: "a/b/c/xxx/yyy",
			expected:    "xxx/yyy/",
		},
		{
			name:        "paths with trailing slashes",
			basePath:    "a/b/c/",
			currentPath: "a/b/c/xxx/",
			expected:    "xxx/",
		},
		{
			name:        "currentPath does not start with basePath",
			basePath:    "a/b/c",
			currentPath: "x/y/z",
			expected:    "",
		},
		{
			name:        "basePath is longer than currentPath",
			basePath:    "a/b/c/d",
			currentPath: "a/b/c",
			expected:    "",
		},
		{
			name:        "partial prefix match should not work",
			basePath:    "a/b/c",
			currentPath: "a/b/cd/xxx",
			expected:    "",
		},
		{
			name:        "root basePath with subdirectory",
			basePath:    "/",
			currentPath: "/home/user",
			expected:    "home/user/",
		},
		{
			name:        "absolute paths",
			basePath:    "/home/user/sync",
			currentPath: "/home/user/sync/subfolder",
			expected:    "subfolder/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rclone.CalculateFilterPrefix(tt.basePath, tt.currentPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
