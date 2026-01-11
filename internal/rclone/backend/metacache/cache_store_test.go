package metacache

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/core/db"
)

func TestNewCacheStore_CreateNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Verify schema was initialized
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM cache_meta WHERE key = 'schema_version'").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestNewCacheStore_OpenExisting(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create initial store
	store1, err := NewCacheStore(dbPath)
	require.NoError(t, err)

	// Insert a test entry
	entry := &CacheEntry{
		Path:     "test/file.txt",
		Parent:   "test",
		ModTime:  time.Now(),
		IsDir:    false,
		CachedAt: time.Now(),
	}
	err = store1.Set(entry)
	require.NoError(t, err)
	store1.Close()

	// Reopen store
	store2, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store2.Close()

	// Verify data persisted
	retrieved, err := store2.Get("test/file.txt")
	assert.NoError(t, err)
	assert.Equal(t, entry.Path, retrieved.Path)
}

func TestNewCacheStore_SchemaVersionMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create database with old schema version
	dsn := db.FileSDN(dbPath)
	sqlDB, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)

	_, err = sqlDB.Exec(`
		CREATE TABLE cache_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO cache_meta (key, value) VALUES ('schema_version', '0');
	`)
	require.NoError(t, err)
	sqlDB.Close()

	// Open with NewCacheStore - should recreate database
	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify schema version is now current
	var version int
	err = store.db.QueryRow("SELECT value FROM cache_meta WHERE key = 'schema_version'").Scan(&version)
	assert.NoError(t, err)
	assert.Equal(t, CurrentCacheSchemaVersion, version)
}

func TestNewCacheStore_WALModeEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify WAL mode is enabled
	var journalMode string
	err = store.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	assert.NoError(t, err)
	assert.Equal(t, "wal", journalMode)
}

func TestCacheStore_GetSet(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	size := int64(1024)
	hash := "md5:abc123"
	entry := &CacheEntry{
		Path:      "documents/report.docx",
		Parent:    "documents",
		ModTime:   time.Now().Truncate(time.Second),
		IsDir:     false,
		Size:      &size,
		Hash:      &hash,
		DirLoaded: false,
		CachedAt:  time.Now().Truncate(time.Second),
	}

	// Set entry
	err = store.Set(entry)
	require.NoError(t, err)

	// Get entry
	retrieved, err := store.Get(entry.Path)
	require.NoError(t, err)

	assert.Equal(t, entry.Path, retrieved.Path)
	assert.Equal(t, entry.Parent, retrieved.Parent)
	assert.Equal(t, entry.IsDir, retrieved.IsDir)
	assert.Equal(t, *entry.Size, *retrieved.Size)
	assert.Equal(t, *entry.Hash, *retrieved.Hash)
	assert.Equal(t, entry.DirLoaded, retrieved.DirLoaded)
	// ModTime stored as nanoseconds, compare with some tolerance
	assert.WithinDuration(t, entry.ModTime, retrieved.ModTime, time.Second)
	assert.WithinDuration(t, entry.CachedAt, retrieved.CachedAt, time.Second)
}

func TestCacheStore_GetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	_, err = store.Get("nonexistent/path")
	assert.ErrorIs(t, err, ErrCacheNotFound)
}

func TestCacheStore_SetBatch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	entries := []*CacheEntry{
		{Path: "dir/file1.txt", Parent: "dir", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "dir/file2.txt", Parent: "dir", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "dir/subdir", Parent: "dir", ModTime: now, IsDir: true, CachedAt: now},
	}

	err = store.SetBatch(entries)
	require.NoError(t, err)

	// Verify all entries were inserted
	for _, entry := range entries {
		retrieved, err := store.Get(entry.Path)
		assert.NoError(t, err)
		assert.Equal(t, entry.Path, retrieved.Path)
	}
}

func TestCacheStore_ListChildren(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	entries := []*CacheEntry{
		{Path: "root", Parent: "", ModTime: now, IsDir: true, CachedAt: now},
		{Path: "root/file1.txt", Parent: "root", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "root/file2.txt", Parent: "root", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "root/subdir", Parent: "root", ModTime: now, IsDir: true, CachedAt: now},
		{Path: "root/subdir/nested.txt", Parent: "root/subdir", ModTime: now, IsDir: false, CachedAt: now},
	}

	err = store.SetBatch(entries)
	require.NoError(t, err)

	// List children of root
	children, err := store.ListChildren("root")
	require.NoError(t, err)
	assert.Len(t, children, 3) // file1.txt, file2.txt, subdir

	// Verify paths
	paths := make([]string, len(children))
	for i, c := range children {
		paths[i] = c.Path
	}
	assert.Contains(t, paths, "root/file1.txt")
	assert.Contains(t, paths, "root/file2.txt")
	assert.Contains(t, paths, "root/subdir")
	assert.NotContains(t, paths, "root/subdir/nested.txt")
}

func TestCacheStore_MarkStale(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	entries := []*CacheEntry{
		{Path: "dir", Parent: "", ModTime: now, IsDir: true, CachedAt: now},
		{Path: "dir/file1.txt", Parent: "dir", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "dir/subdir", Parent: "dir", ModTime: now, IsDir: true, CachedAt: now},
		{Path: "dir/subdir/file2.txt", Parent: "dir/subdir", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "other/file3.txt", Parent: "other", ModTime: now, IsDir: false, CachedAt: now},
	}

	err = store.SetBatch(entries)
	require.NoError(t, err)

	// Mark dir as stale (non-recursive)
	err = store.MarkStale("dir")
	require.NoError(t, err)

	// Verify dir is stale
	entry1, _ := store.Get("dir")
	assert.Equal(t, time.Unix(0, 0), entry1.CachedAt)

	// Verify descendants are NOT stale
	entry2, _ := store.Get("dir/file1.txt")
	assert.NotEqual(t, time.Unix(0, 0), entry2.CachedAt)

	entry3, _ := store.Get("dir/subdir/file2.txt")
	assert.NotEqual(t, time.Unix(0, 0), entry3.CachedAt)
}

func TestCacheStore_MarkStaleRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	entries := []*CacheEntry{
		{Path: "dir", Parent: "", ModTime: now, IsDir: true, CachedAt: now},
		{Path: "dir/file1.txt", Parent: "dir", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "dir/subdir", Parent: "dir", ModTime: now, IsDir: true, CachedAt: now},
		{Path: "dir/subdir/file2.txt", Parent: "dir/subdir", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "other/file3.txt", Parent: "other", ModTime: now, IsDir: false, CachedAt: now},
	}

	err = store.SetBatch(entries)
	require.NoError(t, err)

	// Mark dir as stale recursively
	err = store.MarkStaleRecursive("dir")
	require.NoError(t, err)

	// Verify dir and its descendants are stale (cached_at = 0)
	entry1, _ := store.Get("dir")
	assert.Equal(t, time.Unix(0, 0), entry1.CachedAt)

	entry2, _ := store.Get("dir/file1.txt")
	assert.Equal(t, time.Unix(0, 0), entry2.CachedAt)

	entry3, _ := store.Get("dir/subdir/file2.txt")
	assert.Equal(t, time.Unix(0, 0), entry3.CachedAt)

	// Verify other is not affected
	entry4, _ := store.Get("other/file3.txt")
	assert.NotEqual(t, time.Unix(0, 0), entry4.CachedAt)
}

func TestCacheStore_MarkStaleRecursive_RootPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	entries := []*CacheEntry{
		{Path: "", Parent: "", ModTime: now, IsDir: true, CachedAt: now},
		{Path: "file1.txt", Parent: "", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "dir", Parent: "", ModTime: now, IsDir: true, CachedAt: now},
		{Path: "dir/file2.txt", Parent: "dir", ModTime: now, IsDir: false, CachedAt: now},
	}

	err = store.SetBatch(entries)
	require.NoError(t, err)

	err = store.MarkStaleRecursive("")
	require.NoError(t, err)

	rootEntry, _ := store.Get("")
	assert.Equal(t, time.Unix(0, 0), rootEntry.CachedAt)

	entry1, _ := store.Get("file1.txt")
	assert.Equal(t, time.Unix(0, 0), entry1.CachedAt)

	entry2, _ := store.Get("dir")
	assert.Equal(t, time.Unix(0, 0), entry2.CachedAt)

	entry3, _ := store.Get("dir/file2.txt")
	assert.Equal(t, time.Unix(0, 0), entry3.CachedAt)
}

func TestCacheStore_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	entries := []*CacheEntry{
		{Path: "file1.txt", Parent: "", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "file2.txt", Parent: "", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "dir", Parent: "", ModTime: now, IsDir: true, CachedAt: now},
	}

	err = store.SetBatch(entries)
	require.NoError(t, err)

	// Clear all entries
	count, err := store.Clear()
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Verify all entries are gone
	entriesCount, err := store.GetEntriesCount()
	require.NoError(t, err)
	assert.Equal(t, int64(0), entriesCount)
}

func TestCacheStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	entry := &CacheEntry{
		Path:     "test/file.txt",
		Parent:   "test",
		ModTime:  now,
		IsDir:    false,
		CachedAt: now,
	}

	err = store.Set(entry)
	require.NoError(t, err)

	// Delete entry
	err = store.Delete(entry.Path)
	require.NoError(t, err)

	// Verify entry is gone
	_, err = store.Get(entry.Path)
	assert.ErrorIs(t, err, ErrCacheNotFound)
}

func TestCacheStore_DeleteChildren(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	entries := []*CacheEntry{
		{Path: "parent", Parent: "", ModTime: now, IsDir: true, CachedAt: now},
		{Path: "parent/child1.txt", Parent: "parent", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "parent/child2.txt", Parent: "parent", ModTime: now, IsDir: false, CachedAt: now},
	}

	err = store.SetBatch(entries)
	require.NoError(t, err)

	// Delete children of parent
	err = store.DeleteChildren("parent")
	require.NoError(t, err)

	// Verify parent still exists
	_, err = store.Get("parent")
	assert.NoError(t, err)

	// Verify children are gone
	_, err = store.Get("parent/child1.txt")
	assert.ErrorIs(t, err, ErrCacheNotFound)

	_, err = store.Get("parent/child2.txt")
	assert.ErrorIs(t, err, ErrCacheNotFound)
}

func TestCacheStore_GetEntriesCount(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Initially empty
	count, err := store.GetEntriesCount()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Add entries
	now := time.Now()
	entries := []*CacheEntry{
		{Path: "file1.txt", Parent: "", ModTime: now, IsDir: false, CachedAt: now},
		{Path: "file2.txt", Parent: "", ModTime: now, IsDir: false, CachedAt: now},
	}
	err = store.SetBatch(entries)
	require.NoError(t, err)

	count, err = store.GetEntriesCount()
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestCacheStore_GetDBSize(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	size, err := store.GetDBSize()
	require.NoError(t, err)
	assert.Greater(t, size, int64(0))
}

func TestCacheStore_SetDirLoaded(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	entry := &CacheEntry{
		Path:      "mydir",
		Parent:    "",
		ModTime:   now,
		IsDir:     true,
		DirLoaded: false,
		CachedAt:  now,
	}

	err = store.Set(entry)
	require.NoError(t, err)

	// Initially not loaded
	loaded, err := store.IsDirLoaded("mydir", 24*time.Hour)
	require.NoError(t, err)
	assert.False(t, loaded)

	// Mark as loaded
	err = store.SetDirLoaded("mydir", true)
	require.NoError(t, err)

	// Verify it's now loaded
	loaded, err = store.IsDirLoaded("mydir", 24*time.Hour)
	require.NoError(t, err)
	assert.True(t, loaded)
}

func TestCacheStore_IsDirLoaded_Expired(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Create an entry that was cached 2 hours ago
	pastTime := time.Now().Add(-2 * time.Hour)
	entry := &CacheEntry{
		Path:      "mydir",
		Parent:    "",
		ModTime:   pastTime,
		IsDir:     true,
		DirLoaded: true,
		CachedAt:  pastTime,
	}

	err = store.Set(entry)
	require.NoError(t, err)

	// With 24h TTL, should be loaded
	loaded, err := store.IsDirLoaded("mydir", 24*time.Hour)
	require.NoError(t, err)
	assert.True(t, loaded)

	// With 1h TTL, should be expired
	loaded, err = store.IsDirLoaded("mydir", 1*time.Hour)
	require.NoError(t, err)
	assert.False(t, loaded)
}

func TestCacheStore_IsDirLoaded_ChildExpired(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	pastTime := now.Add(-2 * time.Hour) // 2 hours ago

	// Create directory with fresh timestamp
	dirEntry := &CacheEntry{
		Path:      "mydir",
		Parent:    "",
		ModTime:   now,
		IsDir:     true,
		DirLoaded: true,
		CachedAt:  now, // Fresh timestamp
	}
	err = store.Set(dirEntry)
	require.NoError(t, err)

	// Create children with old timestamp (simulating stale children)
	childEntries := []*CacheEntry{
		{Path: "mydir/file1.txt", Parent: "mydir", ModTime: pastTime, IsDir: false, CachedAt: pastTime},
		{Path: "mydir/file2.txt", Parent: "mydir", ModTime: pastTime, IsDir: false, CachedAt: pastTime},
	}
	err = store.SetBatch(childEntries)
	require.NoError(t, err)

	// With 24h TTL, all entries are valid, should be loaded
	loaded, err := store.IsDirLoaded("mydir", 24*time.Hour)
	require.NoError(t, err)
	assert.True(t, loaded)

	// With 1h TTL, children are expired even though directory is fresh
	// FR-009: Directory should NOT be considered loaded if any child is expired
	loaded, err = store.IsDirLoaded("mydir", 1*time.Hour)
	require.NoError(t, err)
	assert.False(t, loaded, "IsDirLoaded should return false when children are expired")
}

func TestCacheStore_IsDirLoaded_MixedChildExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	pastTime := now.Add(-2 * time.Hour) // 2 hours ago

	// Create directory with fresh timestamp
	dirEntry := &CacheEntry{
		Path:      "mydir",
		Parent:    "",
		ModTime:   now,
		IsDir:     true,
		DirLoaded: true,
		CachedAt:  now,
	}
	err = store.Set(dirEntry)
	require.NoError(t, err)

	// Create one fresh child and one stale child
	childEntries := []*CacheEntry{
		{Path: "mydir/fresh.txt", Parent: "mydir", ModTime: now, IsDir: false, CachedAt: now},           // Fresh
		{Path: "mydir/stale.txt", Parent: "mydir", ModTime: pastTime, IsDir: false, CachedAt: pastTime}, // Stale
	}
	err = store.SetBatch(childEntries)
	require.NoError(t, err)

	// With 1h TTL, one child is expired, directory should NOT be loaded
	loaded, err := store.IsDirLoaded("mydir", 1*time.Hour)
	require.NoError(t, err)
	assert.False(t, loaded, "IsDirLoaded should return false when even one child is expired")
}

func TestCacheStore_SetDirLoaded_DoesNotUpdateCachedAt(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	pastTime := time.Now().Add(-2 * time.Hour)
	entry := &CacheEntry{
		Path:      "mydir",
		Parent:    "",
		ModTime:   pastTime,
		IsDir:     true,
		DirLoaded: false,
		CachedAt:  pastTime, // Old timestamp
	}

	err = store.Set(entry)
	require.NoError(t, err)

	// Mark as loaded
	err = store.SetDirLoaded("mydir", true)
	require.NoError(t, err)

	// Get entry and verify cached_at was NOT updated
	retrieved, err := store.Get("mydir")
	require.NoError(t, err)
	assert.True(t, retrieved.DirLoaded)
	// cached_at should still be the old timestamp
	assert.WithinDuration(t, pastTime, retrieved.CachedAt, time.Second)
}

func TestCacheStore_TTLExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Create an entry that was cached 2 hours ago
	pastTime := time.Now().Add(-2 * time.Hour)
	entry := &CacheEntry{
		Path:     "test/file.txt",
		Parent:   "test",
		ModTime:  pastTime,
		IsDir:    false,
		CachedAt: pastTime,
	}

	err = store.Set(entry)
	require.NoError(t, err)

	// Get the entry
	retrieved, err := store.Get(entry.Path)
	require.NoError(t, err)

	// Check expiration
	assert.True(t, retrieved.IsExpired(1*time.Hour))
	assert.False(t, retrieved.IsExpired(3*time.Hour))
}

func TestCacheStore_LastNotifyTime(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCacheStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Initially zero time
	lastTime := store.GetLastNotifyTime()
	assert.True(t, lastTime.IsZero())

	// Set notify time
	now := time.Now()
	store.SetLastNotifyTime(now)

	// Verify it's updated
	lastTime = store.GetLastNotifyTime()
	assert.WithinDuration(t, now, lastTime, time.Second)
}

func TestGetCacheStore_SharedInstance(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	connID := "test-connection"

	// Clean up any existing stores
	ReleaseCacheStore(connID)

	// Get first store
	store1, err := GetCacheStore(connID, dbPath)
	require.NoError(t, err)

	// Get second store with same ID - should return same instance
	store2, err := GetCacheStore(connID, dbPath)
	require.NoError(t, err)

	assert.Same(t, store1, store2)

	// Release both
	ReleaseCacheStore(connID)
	ReleaseCacheStore(connID)
}

func TestReleaseCacheStore_RefCount(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	connID := "test-connection-refcount"

	// Get store multiple times
	store1, err := GetCacheStore(connID, dbPath)
	require.NoError(t, err)

	_, err = GetCacheStore(connID, dbPath)
	require.NoError(t, err)

	// Release once - should still be available
	ReleaseCacheStore(connID)

	// Store should still be in registry
	cacheStoreMu.Lock()
	_, exists := cacheStores[connID]
	cacheStoreMu.Unlock()
	assert.True(t, exists)

	// Store should still be usable after first release
	_, err = store1.Get("nonexistent")
	assert.ErrorIs(t, err, ErrCacheNotFound) // Query works, just no data

	// Release second time - should close
	ReleaseCacheStore(connID)

	// Store should be removed from registry
	cacheStoreMu.Lock()
	_, exists = cacheStores[connID]
	cacheStoreMu.Unlock()
	assert.False(t, exists)

	// Verify the database was closed by attempting a query
	// After Close(), the underlying *sql.DB should reject queries
	_, err = store1.Get("nonexistent")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrCacheNotFound) // Should be a different error (db closed)
}

func TestCacheStore_RefreshDirectory(t *testing.T) {
	tests := []struct {
		name           string
		initialEntries []*CacheEntry
		refreshEntries []*CacheEntry
		dirPath        string
		assertions     func(t *testing.T, store *CacheStore)
	}{
		{
			name: "PreservesDirLoaded",
			initialEntries: []*CacheEntry{
				{Path: "a", Parent: "", IsDir: true, DirLoaded: true},
				{Path: "a/b", Parent: "a", IsDir: true, DirLoaded: true},
				{Path: "a/file.txt", Parent: "a", IsDir: false},
			},
			refreshEntries: []*CacheEntry{
				{Path: "a/b", Parent: "a", IsDir: true, DirLoaded: false},
				{Path: "a/file.txt", Parent: "a", IsDir: false},
			},
			dirPath: "a",
			assertions: func(t *testing.T, store *CacheStore) {
				entry, err := store.Get("a/b")
				require.NoError(t, err)
				assert.True(t, entry.DirLoaded, "dir_loaded should be preserved")
			},
		},
		{
			name: "DeletesRemovedEntries",
			initialEntries: []*CacheEntry{
				{Path: "a", Parent: "", IsDir: true, DirLoaded: true},
				{Path: "a/b", Parent: "a", IsDir: true, DirLoaded: true},
				{Path: "a/b/deep.txt", Parent: "a/b", IsDir: false},
				{Path: "a/c", Parent: "a", IsDir: true},
				{Path: "a/file.txt", Parent: "a", IsDir: false},
			},
			refreshEntries: []*CacheEntry{
				{Path: "a/c", Parent: "a", IsDir: true},
				{Path: "a/file.txt", Parent: "a", IsDir: false},
			},
			dirPath: "a",
			assertions: func(t *testing.T, store *CacheStore) {
				_, err := store.Get("a/b")
				assert.ErrorIs(t, err, ErrCacheNotFound)
				_, err = store.Get("a/b/deep.txt")
				assert.ErrorIs(t, err, ErrCacheNotFound)
				_, err = store.Get("a/c")
				assert.NoError(t, err)
				_, err = store.Get("a/file.txt")
				assert.NoError(t, err)
			},
		},
		{
			name: "HandlesTypeChange",
			initialEntries: []*CacheEntry{
				{Path: "a", Parent: "", IsDir: true, DirLoaded: true},
				{Path: "a/b", Parent: "a", IsDir: true, DirLoaded: true},
				{Path: "a/b/child.txt", Parent: "a/b", IsDir: false},
			},
			refreshEntries: []*CacheEntry{
				{Path: "a/b", Parent: "a", IsDir: false},
			},
			dirPath: "a",
			assertions: func(t *testing.T, store *CacheStore) {
				entry, err := store.Get("a/b")
				require.NoError(t, err)
				assert.False(t, entry.IsDir)
				_, err = store.Get("a/b/child.txt")
				assert.ErrorIs(t, err, ErrCacheNotFound)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test.db")

			store, err := NewCacheStore(dbPath)
			require.NoError(t, err)
			defer store.Close()

			now := time.Now()
			for _, e := range tc.initialEntries {
				e.ModTime = now
				e.CachedAt = now
			}
			for _, e := range tc.refreshEntries {
				e.ModTime = now
				e.CachedAt = now
			}

			err = store.SetBatch(tc.initialEntries)
			require.NoError(t, err)

			err = store.RefreshDirectory(tc.dirPath, tc.refreshEntries)
			require.NoError(t, err)

			tc.assertions(t, store)
		})
	}
}
