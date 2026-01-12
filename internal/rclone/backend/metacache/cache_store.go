package metacache

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/xzzpig/rclone-sync/internal/core/db"
)

// CurrentCacheSchemaVersion is the current schema version.
// When schema changes are made, increment this version.
// The cache store will automatically delete and recreate the database
// if the stored version doesn't match.
const CurrentCacheSchemaVersion = 1

// ErrCacheNotFound is returned when a cache entry is not found.
var ErrCacheNotFound = errors.New("cache entry not found")

// ErrCacheExpired is returned when a cache entry has expired.
var ErrCacheExpired = errors.New("cache entry expired")

// Global cache store registry for sharing stores across Fs instances.
var (
	cacheStoreMu sync.Mutex
	cacheStores  = map[string]*CacheStore{} // key: fsName
)

// CacheStore manages a SQLite cache database for a single connection.
// It provides CRUD operations for cache entries and handles schema versioning.
type CacheStore struct {
	db     *sql.DB
	dbPath string
	inUse  atomic.Int32

	// lastNotifyTime is updated when ChangeNotify receives a notification.
	// Used for monitoring purposes.
	lastNotifyTime atomic.Value // time.Time

	// onChangeMu protects onChange callback.
	onChangeMu sync.RWMutex
	// onChange is called when cache entries are modified.
	onChange func()
}

// SetOnChange sets the callback to be called when cache entries are modified.
func (s *CacheStore) SetOnChange(fn func()) {
	s.onChangeMu.Lock()
	defer s.onChangeMu.Unlock()
	s.onChange = fn
}

// notifyChange triggers the onChange callback.
func (s *CacheStore) notifyChange() {
	s.onChangeMu.RLock()
	fn := s.onChange
	s.onChangeMu.RUnlock()
	if fn != nil {
		fn()
	}
}

// notifyIfNoError triggers the onChange callback if err is nil.
func (s *CacheStore) notifyIfNoError(err error) error {
	if err == nil {
		s.notifyChange()
	}
	return err
}

// GetCacheStore returns a shared CacheStore for the given fsName.
// If the store doesn't exist, it creates a new one.
// The store is reference-counted; call ReleaseCacheStore when done.
func GetCacheStore(fsName, dbPath string) (*CacheStore, error) {
	cacheStoreMu.Lock()
	defer cacheStoreMu.Unlock()

	// Check if store already exists
	if store, ok := cacheStores[fsName]; ok {
		store.inUse.Add(1)
		return store, nil
	}

	// Create new store
	store, err := NewCacheStore(dbPath)
	if err != nil {
		return nil, err
	}
	store.inUse.Store(1)
	cacheStores[fsName] = store
	return store, nil
}

// ReleaseCacheStore decrements the reference count for the store.
// When the count reaches zero, the store is closed and removed from the registry.
func ReleaseCacheStore(fsName string) {
	cacheStoreMu.Lock()
	store, ok := cacheStores[fsName]
	if !ok {
		cacheStoreMu.Unlock()
		return
	}

	if store.inUse.Add(-1) > 0 {
		cacheStoreMu.Unlock()
		return
	}

	delete(cacheStores, fsName)
	cacheStoreMu.Unlock()

	_ = store.close()
}

// GetCacheStoreIfExists returns the CacheStore for the given fsName if it exists.
// Returns nil if the store doesn't exist. Does not create a new store or modify reference count.
func GetCacheStoreIfExists(fsName string) *CacheStore {
	cacheStoreMu.Lock()
	defer cacheStoreMu.Unlock()
	return cacheStores[fsName]
}

// NewCacheStore creates or opens a CacheStore at the given path.
// It performs schema version checking and recreates the database if needed.
// WAL mode is enabled for better concurrent read performance.
func NewCacheStore(dbPath string) (*CacheStore, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0750); err != nil {
		return nil, err
	}

	// Use db.FileSDN for consistent SQLite connection parameters
	dsn := db.FileSDN(dbPath)

	// Open database with WAL mode
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Check schema version
	version, err := getCacheSchemaVersion(sqlDB)
	if err != nil || version != CurrentCacheSchemaVersion {
		// Version mismatch or error reading version - recreate database
		_ = sqlDB.Close()
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		// Also remove WAL and SHM files
		_ = os.Remove(dbPath + "-wal")
		_ = os.Remove(dbPath + "-shm")

		// Reopen and initialize
		sqlDB, err = sql.Open("sqlite3", dsn)
		if err != nil {
			return nil, err
		}

		if err := initCacheSchema(sqlDB); err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
	}

	store := &CacheStore{db: sqlDB, dbPath: dbPath}
	store.lastNotifyTime.Store(time.Time{})
	return store, nil
}

// getCacheSchemaVersion reads the schema version from the database.
func getCacheSchemaVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow("SELECT value FROM cache_meta WHERE key = 'schema_version'").Scan(&version)
	return version, err
}

// initCacheSchema creates the initial schema for the cache database.
func initCacheSchema(db *sql.DB) error {
	// Create tables and indexes first
	schema := `
		-- Metadata table for schema versioning
		CREATE TABLE cache_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		-- Cache entries table
		-- Stores file and directory metadata for caching
		CREATE TABLE cache_entries (
			path TEXT PRIMARY KEY,           -- Full path relative to connection root
			parent TEXT NOT NULL,            -- Parent directory path for List() queries
			mod_time INTEGER NOT NULL,       -- Modification time (Unix nanoseconds)
			is_dir BOOLEAN NOT NULL,         -- Whether this is a directory
			size INTEGER,                    -- File size in bytes (NULL for directories)
			hash TEXT,                       -- Content hash in format "algorithm:value"
			dir_loaded BOOLEAN DEFAULT FALSE,-- Whether directory children are fully loaded
			cached_at INTEGER NOT NULL       -- Cache timestamp (Unix seconds)
		);

		-- Index for efficient List() queries (find all children of a directory)
		CREATE INDEX idx_parent ON cache_entries(parent);

		-- Index for TTL cleanup
		CREATE INDEX idx_cached_at ON cache_entries(cached_at);

		-- Index for directory queries
		CREATE INDEX idx_is_dir ON cache_entries(is_dir);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Insert schema version using parameterized query for safety
	_, err := db.Exec(
		"INSERT INTO cache_meta (key, value) VALUES (?, ?)",
		"schema_version",
		strconv.Itoa(CurrentCacheSchemaVersion),
	)
	return err
}

// Get retrieves a cache entry by path.
// Returns ErrCacheNotFound if the entry doesn't exist.
func (s *CacheStore) Get(path string) (*CacheEntry, error) {
	var entry CacheEntry
	var modTimeNano, cachedAtUnix int64
	var size sql.NullInt64
	var hash sql.NullString

	if err := s.db.QueryRow(`
		SELECT path, parent, mod_time, is_dir, size, hash, dir_loaded, cached_at
		FROM cache_entries WHERE path = ?
	`, path).Scan(
		&entry.Path, &entry.Parent, &modTimeNano, &entry.IsDir,
		&size, &hash, &entry.DirLoaded, &cachedAtUnix,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCacheNotFound
		}
		return nil, err
	}

	entry.ModTime = time.Unix(0, modTimeNano)
	entry.CachedAt = time.Unix(cachedAtUnix, 0)
	if size.Valid {
		entry.Size = &size.Int64
	}
	if hash.Valid {
		entry.Hash = &hash.String
	}
	return &entry, nil
}

// Set stores or updates a cache entry.
func (s *CacheStore) Set(entry *CacheEntry) error {
	var size, hash any
	if entry.Size != nil {
		size = *entry.Size
	}
	if entry.Hash != nil {
		hash = *entry.Hash
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO cache_entries
		(path, parent, mod_time, is_dir, size, hash, dir_loaded, cached_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		entry.Path,
		entry.Parent,
		entry.ModTime.UnixNano(),
		entry.IsDir,
		size,
		hash,
		entry.DirLoaded,
		entry.CachedAt.Unix(),
	)
	return s.notifyIfNoError(err)
}

// SetBatch stores multiple cache entries in a single transaction.
// This is more efficient than calling Set() multiple times.
func (s *CacheStore) SetBatch(entries []*CacheEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO cache_entries
		(path, parent, mod_time, is_dir, size, hash, dir_loaded, cached_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, entry := range entries {
		var size, hash any
		if entry.Size != nil {
			size = *entry.Size
		}
		if entry.Hash != nil {
			hash = *entry.Hash
		}

		if _, err := stmt.Exec(
			entry.Path,
			entry.Parent,
			entry.ModTime.UnixNano(),
			entry.IsDir,
			size,
			hash,
			entry.DirLoaded,
			entry.CachedAt.Unix(),
		); err != nil {
			return err
		}
	}

	return s.notifyIfNoError(tx.Commit())
}

// ListChildren returns all direct children of the given directory path.
// This is used by Fs.List() to return cached directory contents.
// Note: This excludes the directory itself to avoid returning the parent
// when listing root directory (where parent path = path = "").
func (s *CacheStore) ListChildren(parent string) ([]*CacheEntry, error) {
	rows, err := s.db.Query(`
		SELECT path, parent, mod_time, is_dir, size, hash, dir_loaded, cached_at
		FROM cache_entries WHERE parent = ? AND path != ?
	`, parent, parent)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var entries []*CacheEntry
	for rows.Next() {
		var entry CacheEntry
		var modTimeNano, cachedAtUnix int64
		var size sql.NullInt64
		var hash sql.NullString

		if err := rows.Scan(
			&entry.Path, &entry.Parent, &modTimeNano, &entry.IsDir,
			&size, &hash, &entry.DirLoaded, &cachedAtUnix,
		); err != nil {
			return nil, err
		}

		entry.ModTime = time.Unix(0, modTimeNano)
		entry.CachedAt = time.Unix(cachedAtUnix, 0)
		if size.Valid {
			entry.Size = &size.Int64
		}
		if hash.Valid {
			entry.Hash = &hash.String
		}
		entries = append(entries, &entry)
	}

	return entries, rows.Err()
}

// MarkStale marks a single cache entry as stale.
func (s *CacheStore) MarkStale(path string) error {
	_, err := s.db.Exec(`
		UPDATE cache_entries
		SET cached_at = 0
		WHERE path = ?
	`, path)
	return err
}

// MarkStaleRecursive marks a cache entry and all its descendants as stale.
// This is called when a directory is moved or deleted.
func (s *CacheStore) MarkStaleRecursive(path string) error {
	// Set cached_at to 0 (Unix epoch) to force expiration
	// For root path (""), use "%" to match all entries
	pattern := path + "/%"
	if path == "" {
		pattern = "%"
	}
	_, err := s.db.Exec(`
		UPDATE cache_entries
		SET cached_at = 0
		WHERE path = ? OR path LIKE ?
	`, path, pattern)
	return err
}

// Delete removes a cache entry by path.
func (s *CacheStore) Delete(path string) error {
	_, err := s.db.Exec("DELETE FROM cache_entries WHERE path = ?", path)
	return s.notifyIfNoError(err)
}

// DeleteRecursive removes a cache entry and all its descendants.
func (s *CacheStore) DeleteRecursive(path string) error {
	// For root path (""), use "%" to match all entries
	pattern := path + "/%"
	if path == "" {
		pattern = "%"
	}
	_, err := s.db.Exec(`
		DELETE FROM cache_entries
		WHERE path = ? OR path LIKE ?
	`, path, pattern)
	return s.notifyIfNoError(err)
}

// DeleteChildren removes all children of the given directory.
func (s *CacheStore) DeleteChildren(parent string) error {
	_, err := s.db.Exec("DELETE FROM cache_entries WHERE parent = ?", parent)
	return s.notifyIfNoError(err)
}

// Clear removes all cache entries.
// Returns the number of entries that were deleted.
func (s *CacheStore) Clear() (int64, error) {
	result, err := s.db.Exec("DELETE FROM cache_entries")
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	if count > 0 {
		s.notifyChange()
	}
	return count, err
}

// GetEntriesCount returns the total number of cached entries.
func (s *CacheStore) GetEntriesCount() (int64, error) {
	var count int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM cache_entries").Scan(&count)
	return count, err
}

// GetDBSize returns the size of the cache database file in bytes.
func (s *CacheStore) GetDBSize() (int64, error) {
	info, err := os.Stat(s.dbPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetLastNotifyTime returns the last time a ChangeNotify notification was received.
func (s *CacheStore) GetLastNotifyTime() time.Time {
	t := s.lastNotifyTime.Load()
	if t == nil {
		return time.Time{}
	}
	return t.(time.Time)
}

// SetLastNotifyTime updates the last notification time.
func (s *CacheStore) SetLastNotifyTime(t time.Time) {
	s.lastNotifyTime.Store(t)
}

// Close closes the database connection.
// This is called internally when the reference count reaches zero.
func (s *CacheStore) Close() error {
	return s.close()
}

// close is the internal close method.
func (s *CacheStore) close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SetDirLoaded marks a directory as having all children loaded.
// Note: This does NOT update cached_at to avoid desynchronizing the directory's
// TTL from its children's TTL. The cached_at is only set during initial caching.
func (s *CacheStore) SetDirLoaded(path string, loaded bool) error {
	_, err := s.db.Exec(`
		UPDATE cache_entries
		SET dir_loaded = ?
		WHERE path = ?
	`, loaded, path)
	return s.notifyIfNoError(err)
}

// RefreshDirectory performs an incremental refresh of a directory's cache.
// Unlike DeleteChildren + SetBatch, this method:
// 1. Handles type changes (directory→file: deletes subtree first)
// 2. UPSERTs new entries while PRESERVING existing dir_loaded for subdirectories
// 3. Deletes entries that no longer exist in source (with recursive cleanup for directories)
// 4. Marks the directory as loaded
//
// This avoids the cascading refresh problem where refreshing a parent directory
// would cause all subdirectories to lose their dir_loaded state.
func (s *CacheStore) RefreshDirectory(dirPath string, entries []*CacheEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Step 1: Get current cached children for comparison
	currentChildren, err := s.listChildrenTx(tx, dirPath)
	if err != nil {
		return err
	}
	currentMap := make(map[string]*CacheEntry)
	for _, c := range currentChildren {
		currentMap[c.Path] = c
	}

	// Step 2: Build set of new entry paths
	newPaths := make(map[string]bool)
	for _, e := range entries {
		newPaths[e.Path] = true
	}

	// Step 3: Handle type changes (directory→file: delete subtree first)
	for _, e := range entries {
		if old, exists := currentMap[e.Path]; exists {
			if old.IsDir && !e.IsDir {
				// Directory became a file, delete all its children first
				if err := s.deleteChildrenRecursiveTx(tx, e.Path); err != nil {
					return err
				}
			}
		}
	}

	// Step 4: UPSERT all new entries (preserving dir_loaded for existing directories)
	upsertStmt, err := tx.Prepare(`
		INSERT INTO cache_entries 
			(path, parent, mod_time, is_dir, size, hash, dir_loaded, cached_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			parent = excluded.parent,
			mod_time = excluded.mod_time,
			is_dir = excluded.is_dir,
			size = excluded.size,
			hash = excluded.hash,
			dir_loaded = CASE 
				WHEN excluded.is_dir = 1 AND cache_entries.is_dir = 1 
				THEN cache_entries.dir_loaded
				ELSE excluded.dir_loaded
			END,
			cached_at = excluded.cached_at
	`)
	if err != nil {
		return err
	}
	defer func() {
		_ = upsertStmt.Close()
	}()

	for _, entry := range entries {
		var size, hash any
		if entry.Size != nil {
			size = *entry.Size
		}
		if entry.Hash != nil {
			hash = *entry.Hash
		}

		if _, err := upsertStmt.Exec(
			entry.Path,
			entry.Parent,
			entry.ModTime.UnixNano(),
			entry.IsDir,
			size,
			hash,
			entry.DirLoaded,
			entry.CachedAt.Unix(),
		); err != nil {
			return err
		}
	}

	// Step 5: Delete entries that no longer exist in source
	for path, old := range currentMap {
		if !newPaths[path] {
			if _, err := tx.Exec("DELETE FROM cache_entries WHERE path = ?", path); err != nil {
				return err
			}
			if old.IsDir {
				if err := s.deleteChildrenRecursiveTx(tx, path); err != nil {
					return err
				}
			}
		}
	}

	// Step 6: Mark the directory itself as loaded with updated timestamp
	now := time.Now()
	_, err = tx.Exec(`
		INSERT INTO cache_entries 
			(path, parent, mod_time, is_dir, size, hash, dir_loaded, cached_at)
		VALUES (?, ?, ?, 1, NULL, NULL, 1, ?)
		ON CONFLICT(path) DO UPDATE SET
			dir_loaded = 1,
			cached_at = excluded.cached_at
	`, dirPath, normalizeParentForStore(dirPath), now.UnixNano(), now.Unix())
	if err != nil {
		return err
	}

	return s.notifyIfNoError(tx.Commit())
}

// listChildrenTx returns all direct children of the given directory path within a transaction.
func (s *CacheStore) listChildrenTx(tx *sql.Tx, parent string) ([]*CacheEntry, error) {
	rows, err := tx.Query(`
		SELECT path, parent, mod_time, is_dir, size, hash, dir_loaded, cached_at
		FROM cache_entries WHERE parent = ? AND path != ?
	`, parent, parent)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var entries []*CacheEntry
	for rows.Next() {
		var entry CacheEntry
		var modTimeNano, cachedAtUnix int64
		var size sql.NullInt64
		var hash sql.NullString

		if err := rows.Scan(
			&entry.Path, &entry.Parent, &modTimeNano, &entry.IsDir,
			&size, &hash, &entry.DirLoaded, &cachedAtUnix,
		); err != nil {
			return nil, err
		}

		entry.ModTime = time.Unix(0, modTimeNano)
		entry.CachedAt = time.Unix(cachedAtUnix, 0)
		if size.Valid {
			entry.Size = &size.Int64
		}
		if hash.Valid {
			entry.Hash = &hash.String
		}
		entries = append(entries, &entry)
	}

	return entries, rows.Err()
}

// deleteChildrenRecursiveTx deletes all descendants of the given directory within a transaction.
func (s *CacheStore) deleteChildrenRecursiveTx(tx *sql.Tx, dirPath string) error {
	pattern := dirPath + "/%"
	_, err := tx.Exec("DELETE FROM cache_entries WHERE path LIKE ?", pattern)
	return err
}

// normalizeParentForStore returns the normalized parent path for cache entries.
// This is a copy of normalizeParent for use within the cache store package.
func normalizeParentForStore(p string) string {
	parent := filepath.Dir(p)
	if parent == "." || parent == "" || parent == "/" {
		return ""
	}
	return parent
}

// IsDirLoaded checks if a directory has all children loaded and none are expired.
// This ensures that when returning cached directory contents, all entries are valid.
// FR-009: If any child entry is expired, the entire directory needs to be refreshed.
// FR-004: If infoAge is 0 or negative, entries never expire.
func (s *CacheStore) IsDirLoaded(path string, infoAge time.Duration) (bool, error) {
	entry, err := s.Get(path)
	if err != nil {
		if errors.Is(err, ErrCacheNotFound) {
			return false, nil
		}
		return false, err
	}
	if !entry.IsDir || !entry.DirLoaded {
		return false, nil
	}

	// Check if the directory entry itself is expired
	if entry.IsExpired(infoAge) {
		return false, nil
	}

	// If infoAge <= 0, entries never expire, skip expiration check
	if infoAge <= 0 {
		return true, nil
	}

	// Check if any child entry is expired
	// This ensures we don't return stale children when the parent looks valid
	expiredBefore := time.Now().Add(-infoAge).Unix()
	var expiredCount int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM cache_entries 
		WHERE parent = ? AND cached_at < ?
	`, path, expiredBefore).Scan(&expiredCount)
	if err != nil {
		return false, err
	}

	// If any child is expired, the directory needs to be refreshed
	return expiredCount == 0, nil
}
