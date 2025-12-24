package rclone_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/rclone/rclone/backend/alias"
	_ "github.com/rclone/rclone/backend/combine"
	_ "github.com/rclone/rclone/backend/local"
	_ "github.com/rclone/rclone/backend/union"
)

// TestDBStorage_Integration_FsNewFs tests that fs.NewFs works correctly with DBStorage
func TestDBStorage_Integration_FsNewFs(t *testing.T) {
	_, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Create a local connection in the database
	tempDir := t.TempDir()
	_, err := connSvc.CreateConnection(ctx, "test-local", "local", map[string]string{})
	require.NoError(t, err)

	// Create a test file in tempDir
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("hello"), 0644)
	require.NoError(t, err)

	// Use fs.NewFs to create a filesystem from DBStorage config
	remotePath := "test-local:" + tempDir
	f, err := fs.NewFs(ctx, remotePath)
	require.NoError(t, err, "fs.NewFs should succeed with DBStorage config")

	// Verify we can list files
	entries, err := f.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, entries, 1, "Should find one file")
	assert.Equal(t, "test.txt", entries[0].Remote())
}

// TestDBStorage_Integration_ConfigGetRemoteNames tests that config.GetRemoteNames works with DBStorage
func TestDBStorage_Integration_ConfigGetRemoteNames(t *testing.T) {
	_, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Initially no remotes
	names := config.GetRemoteNames()
	assert.Empty(t, names, "Should have no remotes initially")

	// Create some connections
	_, err := connSvc.CreateConnection(ctx, "remote-a", "local", map[string]string{})
	require.NoError(t, err)
	_, err = connSvc.CreateConnection(ctx, "remote-b", "local", map[string]string{})
	require.NoError(t, err)
	_, err = connSvc.CreateConnection(ctx, "remote-c", "local", map[string]string{})
	require.NoError(t, err)

	// Verify config.GetRemoteNames returns all connections
	names = config.GetRemoteNames()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "remote-a")
	assert.Contains(t, names, "remote-b")
	assert.Contains(t, names, "remote-c")
}

// TestDBStorage_Integration_ConfigFileGet tests that config.FileGetValue works with DBStorage
func TestDBStorage_Integration_ConfigFileGet(t *testing.T) {
	_, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Create a connection with specific config
	_, err := connSvc.CreateConnection(ctx, "config-test", "onedrive", map[string]string{
		"token":    `{"access_token":"test-token"}`,
		"drive_id": "abc123",
	})
	require.NoError(t, err)

	// Verify config.FileGetValue reads from DBStorage
	val, ok := config.FileGetValue("config-test", "type")
	assert.True(t, ok)
	assert.Equal(t, "onedrive", val)

	val, ok = config.FileGetValue("config-test", "token")
	assert.True(t, ok)
	assert.Equal(t, `{"access_token":"test-token"}`, val)

	val, ok = config.FileGetValue("config-test", "drive_id")
	assert.True(t, ok)
	assert.Equal(t, "abc123", val)

	// Non-existing key
	_, ok = config.FileGetValue("config-test", "non-existing")
	assert.False(t, ok)
}

// TestDBStorage_Integration_ConfigFileSet tests that config.FileSetValue writes to DBStorage
func TestDBStorage_Integration_ConfigFileSet(t *testing.T) {
	_, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Create a connection
	_, err := connSvc.CreateConnection(ctx, "set-test", "s3", map[string]string{
		"token": "old-token",
	})
	require.NoError(t, err)

	// Update value via config.FileSetValue (simulating rclone token refresh)
	config.FileSetValue("set-test", "token", "new-refreshed-token")

	// Verify the update via DBStorage (through ConnectionService)
	cfg, err := connSvc.GetConnectionConfig(ctx, "set-test")
	require.NoError(t, err)
	assert.Equal(t, "new-refreshed-token", cfg["token"], "Token should be updated in database")

	// Also verify via config.FileGetValue
	val, ok := config.FileGetValue("set-test", "token")
	assert.True(t, ok)
	assert.Equal(t, "new-refreshed-token", val)
}

// TestDBStorage_Integration_TokenRefresh_Persistence simulates OAuth token refresh scenario
func TestDBStorage_Integration_TokenRefresh_Persistence(t *testing.T) {
	_, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Create an OAuth connection with initial token
	initialToken := `{"access_token":"old_access","refresh_token":"xxx","expiry":"2024-01-01T00:00:00Z"}`
	_, err := connSvc.CreateConnection(ctx, "oauth-remote", "onedrive", map[string]string{
		"token": initialToken,
	})
	require.NoError(t, err)

	// Simulate rclone refreshing the token (this is what rclone does internally)
	newToken := `{"access_token":"new_access","refresh_token":"xxx","expiry":"2025-01-01T00:00:00Z"}`
	config.FileSetValue("oauth-remote", "token", newToken)

	// Verify the token was persisted to database
	cfg, err := connSvc.GetConnectionConfig(ctx, "oauth-remote")
	require.NoError(t, err)
	assert.Equal(t, newToken, cfg["token"], "Refreshed token should be persisted")

	// Verify reading back via rclone config API
	val, ok := config.FileGetValue("oauth-remote", "token")
	assert.True(t, ok)
	assert.Equal(t, newToken, val)
}

// TestDBStorage_ConcurrentAccess tests thread safety of DBStorage
func TestDBStorage_ConcurrentAccess(t *testing.T) {
	storage, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Create a connection for testing
	_, err := connSvc.CreateConnection(ctx, "concurrent-test", "local", map[string]string{
		"counter": "0",
	})
	require.NoError(t, err)

	// Run concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 50

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Read operation
				_, _ = storage.GetValue("concurrent-test", "type")

				// Write operation
				storage.SetValue("concurrent-test", "key", "value")

				// Check section
				_ = storage.HasSection("concurrent-test")

				// Get key list
				_ = storage.GetKeyList("concurrent-test")
			}
		}(i)
	}

	wg.Wait()

	// Verify data integrity after concurrent access
	assert.True(t, storage.HasSection("concurrent-test"))
	val, ok := storage.GetValue("concurrent-test", "type")
	assert.True(t, ok)
	assert.Equal(t, "local", val)
}

// TestSyncEngine_WithDBStorage_Integration tests full sync flow using DBStorage
func TestSyncEngine_WithDBStorage_Integration(t *testing.T) {
	// Create test database client
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	// Create encryptor
	encryptor, err := crypto.NewEncryptor("")
	require.NoError(t, err)

	// Create services
	connSvc := services.NewConnectionService(client, encryptor)
	jobSvc := services.NewJobService(client)
	taskSvc := services.NewTaskService(client)

	// Install DBStorage
	storage := rclone.NewDBStorage(connSvc)
	storage.Install()

	ctx := context.Background()

	// Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "sync-test.txt")
	err = os.WriteFile(testFilePath, []byte("sync test content"), 0644)
	require.NoError(t, err)

	// Create local connection via ConnectionService (this goes to database)
	testConn, err := connSvc.CreateConnection(ctx, "db-local", "local", map[string]string{})
	require.NoError(t, err)

	// Create task
	testTask, err := taskSvc.CreateTask(ctx,
		"DBStorageSyncTest",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// Reload task with Connection edge
	testTask, err = taskSvc.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobSvc, nil, dataDir)

	// Run the task - this should use DBStorage to read the connection config
	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	// Verify results
	destFilePath := filepath.Join(destDir, "sync-test.txt")
	content, err := os.ReadFile(destFilePath)
	require.NoError(t, err, "File should exist in destination")
	assert.Equal(t, "sync test content", string(content))

	// Check job was created
	jobs, err := jobSvc.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, string(model.JobStatusSuccess), string(jobs[0].Status))
}

// TestDBStorage_Integration_DeleteConnection tests that deleting a connection clears rclone cache
func TestDBStorage_Integration_DeleteConnection(t *testing.T) {
	storage, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Create a connection
	_, err := connSvc.CreateConnection(ctx, "delete-test", "local", map[string]string{})
	require.NoError(t, err)

	// Verify it exists
	assert.True(t, storage.HasSection("delete-test"))
	assert.Contains(t, config.GetRemoteNames(), "delete-test")

	// Delete the section
	storage.DeleteSection("delete-test")

	// Verify it's gone
	assert.False(t, storage.HasSection("delete-test"))
	assert.NotContains(t, config.GetRemoteNames(), "delete-test")
}

// TestDBStorage_Integration_CreateViaSetValue tests creating connection via SetValue
func TestDBStorage_Integration_CreateViaSetValue(t *testing.T) {
	storage, _ := setupTestConfig(t)

	// Connection doesn't exist yet
	assert.False(t, storage.HasSection("new-via-set"))

	// Create by setting type (this is how rclone config create works)
	storage.SetValue("new-via-set", "type", "local")

	// Verify connection was created
	assert.True(t, storage.HasSection("new-via-set"))
	val, ok := storage.GetValue("new-via-set", "type")
	assert.True(t, ok)
	assert.Equal(t, "local", val)
}

// ============================================================================
// Wrapper Fs Integration Tests (alias, union, combine)
// These tests verify that rclone's wrapper backends work correctly with DBStorage
// ============================================================================

// TestDBStorage_Integration_AliasRemote tests that alias backend works with DBStorage
// Alias backend creates an alias to another remote, with optional path prefix
func TestDBStorage_Integration_AliasRemote(t *testing.T) {
	_, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Create temp directory with test files
	baseDir := t.TempDir()
	subDir := filepath.Join(baseDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file1.txt"), []byte("content1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644))

	// Create base local connection
	_, err := connSvc.CreateConnection(ctx, "base-local", "local", map[string]string{})
	require.NoError(t, err)

	// Create alias connection pointing to base-local:subdir
	_, err = connSvc.CreateConnection(ctx, "my-alias", "alias", map[string]string{
		"remote": "base-local:" + subDir,
	})
	require.NoError(t, err)

	// Use alias remote to list files
	f, err := fs.NewFs(ctx, "my-alias:")
	require.NoError(t, err, "fs.NewFs should succeed with alias remote")

	// Verify we can list files through the alias
	entries, err := f.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, entries, 2, "Should find two files through alias")

	// Collect file names
	fileNames := make([]string, 0, len(entries))
	for _, e := range entries {
		fileNames = append(fileNames, e.Remote())
	}
	assert.Contains(t, fileNames, "file1.txt")
	assert.Contains(t, fileNames, "file2.txt")
}

// TestDBStorage_Integration_UnionRemote tests that union backend works with DBStorage
// Union backend merges multiple upstreams into a single readonly view
func TestDBStorage_Integration_UnionRemote(t *testing.T) {
	_, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Create two temp directories with different files
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "from-dir1.txt"), []byte("dir1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "from-dir2.txt"), []byte("dir2"), 0644))

	// Create two local connections
	_, err := connSvc.CreateConnection(ctx, "local-1", "local", map[string]string{})
	require.NoError(t, err)

	_, err = connSvc.CreateConnection(ctx, "local-2", "local", map[string]string{})
	require.NoError(t, err)

	// Create union connection that merges both locals
	_, err = connSvc.CreateConnection(ctx, "my-union", "union", map[string]string{
		"upstreams": "local-1:" + dir1 + " local-2:" + dir2,
	})
	require.NoError(t, err)

	// Use union remote to list files from both directories
	f, err := fs.NewFs(ctx, "my-union:")
	require.NoError(t, err, "fs.NewFs should succeed with union remote")

	// Verify we can see files from both directories
	entries, err := f.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, entries, 2, "Should find files from both upstreams")

	// Collect file names
	fileNames := make([]string, 0, len(entries))
	for _, e := range entries {
		fileNames = append(fileNames, e.Remote())
	}
	assert.Contains(t, fileNames, "from-dir1.txt")
	assert.Contains(t, fileNames, "from-dir2.txt")
}

// TestDBStorage_Integration_CombineRemote tests that combine backend works with DBStorage
// Combine backend presents multiple remotes as subdirectories of a single remote
func TestDBStorage_Integration_CombineRemote(t *testing.T) {
	_, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Create two temp directories with files
	dirA := t.TempDir()
	dirB := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dirA, "fileA.txt"), []byte("A"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dirB, "fileB.txt"), []byte("B"), 0644))

	// Create two local connections
	_, err := connSvc.CreateConnection(ctx, "local-a", "local", map[string]string{})
	require.NoError(t, err)

	_, err = connSvc.CreateConnection(ctx, "local-b", "local", map[string]string{})
	require.NoError(t, err)

	// Create combine connection that maps locals to subdirectories
	_, err = connSvc.CreateConnection(ctx, "my-combine", "combine", map[string]string{
		"upstreams": "folder-a=local-a:" + dirA + " folder-b=local-b:" + dirB,
	})
	require.NoError(t, err)

	// Use combine remote to list root - should see two virtual directories
	f, err := fs.NewFs(ctx, "my-combine:")
	require.NoError(t, err, "fs.NewFs should succeed with combine remote")

	// List root to see virtual folders
	entries, err := f.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, entries, 2, "Should find two virtual directories")

	// Collect folder names
	folderNames := make([]string, 0, len(entries))
	for _, e := range entries {
		folderNames = append(folderNames, e.Remote())
	}
	assert.Contains(t, folderNames, "folder-a")
	assert.Contains(t, folderNames, "folder-b")

	// Access files within folder-a
	fA, err := fs.NewFs(ctx, "my-combine:folder-a")
	require.NoError(t, err)
	entriesA, err := fA.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, entriesA, 1)
	assert.Equal(t, "fileA.txt", entriesA[0].Remote())

	// Access files within folder-b
	fB, err := fs.NewFs(ctx, "my-combine:folder-b")
	require.NoError(t, err)
	entriesB, err := fB.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, entriesB, 1)
	assert.Equal(t, "fileB.txt", entriesB[0].Remote())
}

// TestDBStorage_Integration_NestedWrappers tests nested wrapper remotes
// This tests alias pointing to another wrapper remote (multi-level resolution)
func TestDBStorage_Integration_NestedWrappers(t *testing.T) {
	_, connSvc := setupTestConfig(t)
	ctx := context.Background()

	// Create temp directory with test file
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "nested-file.txt"), []byte("nested"), 0644))

	// Create base local connection
	_, err := connSvc.CreateConnection(ctx, "nested-local", "local", map[string]string{})
	require.NoError(t, err)

	// Create first alias pointing to local
	_, err = connSvc.CreateConnection(ctx, "alias-level1", "alias", map[string]string{
		"remote": "nested-local:" + baseDir,
	})
	require.NoError(t, err)

	// Create second alias pointing to first alias (nested)
	_, err = connSvc.CreateConnection(ctx, "alias-level2", "alias", map[string]string{
		"remote": "alias-level1:",
	})
	require.NoError(t, err)

	// Use the deeply nested alias to access files
	f, err := fs.NewFs(ctx, "alias-level2:")
	require.NoError(t, err, "fs.NewFs should succeed with nested alias")

	// Verify we can list files through nested wrappers
	entries, err := f.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "nested-file.txt", entries[0].Remote())
}
