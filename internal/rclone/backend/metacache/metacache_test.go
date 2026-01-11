package metacache

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/rclone/rclone/backend/local"
	_ "github.com/rclone/rclone/backend/memory"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnv(t *testing.T) (string, string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	localDir := filepath.Join(tempDir, "local")
	cacheDB := filepath.Join(tempDir, "cache.db")

	require.NoError(t, os.MkdirAll(localDir, 0750))

	cleanup := func() {
		cacheStoreMu.Lock()
		for id := range cacheStores {
			if store, ok := cacheStores[id]; ok {
				store.close()
			}
			delete(cacheStores, id)
		}
		cacheStoreMu.Unlock()
	}

	return localDir, cacheDB, cleanup
}

func createMetaCacheFs(t *testing.T, localDir, cacheDB, name string, infoAge time.Duration) (fs.Fs, error) {
	t.Helper()

	m := configmap.Simple{
		"remote":             localDir,
		"db_path":            cacheDB,
		"info_age":           fs.Duration(infoAge).String(),
		"change_notify_poll": fs.Duration(time.Minute).String(),
	}

	return NewFs(context.Background(), name, "", m)
}

func TestNewFs_Options(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("RequiredOptions", func(t *testing.T) {
		tests := []struct {
			name    string
			config  configmap.Simple
			wantErr string
		}{
			{"missing remote", configmap.Simple{"db_path": cacheDB}, "remote option is required"},
			{"missing db_path", configmap.Simple{"remote": "TestLocal:/tmp"}, "db_path option is required"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := NewFs(context.Background(), "test", "", tt.config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			})
		}
	})

	t.Run("DefaultOptions", func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(localDir, "test.txt"), []byte("test"), 0600))
		m := configmap.Simple{"remote": localDir, "db_path": cacheDB}
		fsObj, err := NewFs(context.Background(), "test-defaults", "", m)
		require.NoError(t, err)
		defer fsObj.(*Fs).Shutdown(context.Background())

		mcFs := fsObj.(*Fs)
		entries, err := mcFs.List(context.Background(), "")
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, "test-defaults", mcFs.Name())
	})
}

func TestList_CacheBehavior(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.WriteFile(filepath.Join(localDir, "file1.txt"), []byte("test"), 0600))

	fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-cache", 24*time.Hour)
	require.NoError(t, err)
	defer fsObj.(*Fs).Shutdown(context.Background())
	mcFs := fsObj.(*Fs)

	t.Run("Miss", func(t *testing.T) {
		dirLoaded, _ := mcFs.cache.IsDirLoaded("", time.Duration(mcFs.opt.InfoAge))
		assert.False(t, dirLoaded)

		entries, err := mcFs.List(context.Background(), "")
		require.NoError(t, err)
		assert.Len(t, entries, 1)

		dirLoaded, _ = mcFs.cache.IsDirLoaded("", time.Duration(mcFs.opt.InfoAge))
		assert.True(t, dirLoaded)
	})

	t.Run("Hit", func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(localDir, "file2.txt"), []byte("test"), 0600))
		entries, err := mcFs.List(context.Background(), "")
		require.NoError(t, err)
		assert.Len(t, entries, 1)
	})

	t.Run("RootPathConsistency", func(t *testing.T) {
		rootEntry, err := mcFs.cache.Get("")
		require.NoError(t, err)
		assert.True(t, rootEntry.IsDir && rootEntry.DirLoaded)
	})
}

func TestReceiveChangeNotify(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create test directory structure:
	// root/
	//   file1.txt
	//   file2.txt
	//   subdir/
	//     file3.txt
	//     file4.txt
	//     nested/
	//       file5.txt
	require.NoError(t, os.MkdirAll(filepath.Join(localDir, "subdir/nested"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "file1.txt"), []byte("test"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "file2.txt"), []byte("test"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "subdir/file3.txt"), []byte("test"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "subdir/file4.txt"), []byte("test"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "subdir/nested/file5.txt"), []byte("test"), 0600))

	// Table driven test cases
	tests := []struct {
		name              string
		notifyPath        string
		entryType         fs.EntryType
		expectStale       []string // These paths cached_at should be 0
		expectDirUnloaded []string // These paths dir_loaded should be false
		expectUnchanged   []string // These paths should remain unchanged
	}{
		{
			name:              "FileInSubdir",
			notifyPath:        "subdir/file3.txt",
			entryType:         fs.EntryObject,
			expectStale:       []string{"subdir/file3.txt"},
			expectDirUnloaded: []string{"subdir"},
			expectUnchanged:   []string{"", "file1.txt", "file2.txt", "subdir/file4.txt", "subdir/nested", "subdir/nested/file5.txt"},
		},
		{
			name:              "FileInRoot",
			notifyPath:        "file1.txt",
			entryType:         fs.EntryObject,
			expectStale:       []string{"file1.txt"},
			expectDirUnloaded: []string{""},
			expectUnchanged:   []string{"file2.txt", "subdir", "subdir/file3.txt", "subdir/nested"},
		},
		{
			name:              "DirectorySubdir",
			notifyPath:        "subdir",
			entryType:         fs.EntryDirectory,
			expectStale:       []string{"subdir"},
			expectDirUnloaded: []string{"", "subdir"}, // Both parent and self
			expectUnchanged:   []string{"file1.txt", "file2.txt", "subdir/file3.txt", "subdir/file4.txt", "subdir/nested", "subdir/nested/file5.txt"},
		},
		{
			name:              "NestedDirectory",
			notifyPath:        "subdir/nested",
			entryType:         fs.EntryDirectory,
			expectStale:       []string{"subdir/nested"},
			expectDirUnloaded: []string{"subdir", "subdir/nested"}, // Only direct parent and self
			expectUnchanged:   []string{"", "file1.txt", "subdir/file3.txt", "subdir/nested/file5.txt"},
		},
		{
			name:              "FileInNestedDirectory",
			notifyPath:        "subdir/nested/file5.txt",
			entryType:         fs.EntryObject,
			expectStale:       []string{"subdir/nested/file5.txt"},
			expectDirUnloaded: []string{"subdir/nested"}, // Only direct parent
			expectUnchanged:   []string{"", "subdir", "subdir/file3.txt", "subdir/file4.txt"},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Independent Fs instance for each test case
			name := fmt.Sprintf("test-notify-%d", i)
			fsObj, err := createMetaCacheFs(t, localDir, cacheDB, name, 24*time.Hour)
			require.NoError(t, err)
			defer fsObj.(*Fs).Shutdown(context.Background())
			mcFs := fsObj.(*Fs)

			// Fill cache
			_, err = mcFs.List(context.Background(), "")
			require.NoError(t, err)
			_, err = mcFs.List(context.Background(), "subdir")
			require.NoError(t, err)
			_, err = mcFs.List(context.Background(), "subdir/nested")
			require.NoError(t, err)

			// Collect all paths to check
			allPaths := make(map[string]struct{})
			for _, p := range tt.expectStale {
				allPaths[p] = struct{}{}
			}
			for _, p := range tt.expectDirUnloaded {
				allPaths[p] = struct{}{}
			}
			for _, p := range tt.expectUnchanged {
				allPaths[p] = struct{}{}
			}

			// Record states before notify
			beforeStates := make(map[string]*CacheEntry)
			for path := range allPaths {
				entry, err := mcFs.cache.Get(path)
				require.NoError(t, err, "failed to get entry %q before notify", path)
				entryCopy := *entry
				beforeStates[path] = &entryCopy
			}

			// Record lastNotifyTime before notify
			beforeNotifyTime := mcFs.cache.GetLastNotifyTime()

			// Trigger ChangeNotify
			mcFs.receiveChangeNotify(tt.notifyPath, tt.entryType)

			// Verify expectStale entries cached_at = 0
			for _, path := range tt.expectStale {
				entry, err := mcFs.cache.Get(path)
				require.NoError(t, err, "failed to get entry %q after notify", path)
				assert.Equal(t, time.Unix(0, 0), entry.CachedAt,
					"entry %q should be stale (cached_at=0)", path)
			}

			// Verify expectDirUnloaded entries dir_loaded = false
			for _, path := range tt.expectDirUnloaded {
				entry, err := mcFs.cache.Get(path)
				require.NoError(t, err, "failed to get entry %q after notify", path)
				assert.False(t, entry.DirLoaded,
					"entry %q should have dir_loaded=false", path)
			}

			// Verify expectUnchanged entries remain unchanged
			for _, path := range tt.expectUnchanged {
				entry, err := mcFs.cache.Get(path)
				require.NoError(t, err, "failed to get entry %q after notify", path)
				before := beforeStates[path]
				require.NotNil(t, before, "no before state for %q", path)
				assert.Equal(t, before.CachedAt, entry.CachedAt,
					"entry %q cached_at should be unchanged", path)
				assert.Equal(t, before.DirLoaded, entry.DirLoaded,
					"entry %q dir_loaded should be unchanged", path)
			}

			// Verify lastNotifyTime is updated
			afterNotifyTime := mcFs.cache.GetLastNotifyTime()
			assert.False(t, afterNotifyTime.IsZero(), "lastNotifyTime should not be zero")
			assert.True(t, afterNotifyTime.After(beforeNotifyTime) || !beforeNotifyTime.IsZero(),
				"lastNotifyTime should be updated")
			assert.WithinDuration(t, time.Now(), afterNotifyTime, time.Second,
				"lastNotifyTime should be close to current time")
		})
	}

	// Extra scenarios
	t.Run("NonExistentEntry_ParentStillAffected", func(t *testing.T) {
		name := "test-notify-nonexistent"
		fsObj, err := createMetaCacheFs(t, localDir, cacheDB, name, 24*time.Hour)
		require.NoError(t, err)
		defer fsObj.(*Fs).Shutdown(context.Background())
		mcFs := fsObj.(*Fs)

		_, err = mcFs.List(context.Background(), "")
		require.NoError(t, err)

		rootEntry, err := mcFs.cache.Get("")
		require.NoError(t, err)
		assert.True(t, rootEntry.DirLoaded)

		mcFs.receiveChangeNotify("nonexistent.txt", fs.EntryObject)

		rootEntry, err = mcFs.cache.Get("")
		require.NoError(t, err)
		assert.False(t, rootEntry.DirLoaded,
			"parent dir_loaded should be false even for non-existent entry")

		_, err = mcFs.cache.Get("nonexistent.txt")
		assert.ErrorIs(t, err, ErrCacheNotFound)
	})

	t.Run("ForwardsToSubscribers", func(t *testing.T) {
		name := "test-notify-forward"
		fsObj, err := createMetaCacheFs(t, localDir, cacheDB, name, 24*time.Hour)
		require.NoError(t, err)
		defer fsObj.(*Fs).Shutdown(context.Background())
		mcFs := fsObj.(*Fs)

		var receivedNotifications []struct {
			path      string
			entryType fs.EntryType
		}
		var mu sync.Mutex

		pollChan := make(chan time.Duration)
		mcFs.ChangeNotify(context.Background(), func(path string, entryType fs.EntryType) {
			mu.Lock()
			receivedNotifications = append(receivedNotifications, struct {
				path      string
				entryType fs.EntryType
			}{path, entryType})
			mu.Unlock()
		}, pollChan)

		mcFs.receiveChangeNotify("test/file.txt", fs.EntryObject)
		mcFs.receiveChangeNotify("test/dir", fs.EntryDirectory)

		mu.Lock()
		defer mu.Unlock()
		require.Len(t, receivedNotifications, 2)
		assert.Equal(t, "test/file.txt", receivedNotifications[0].path)
		assert.Equal(t, fs.EntryObject, receivedNotifications[0].entryType)
		assert.Equal(t, "test/dir", receivedNotifications[1].path)
		assert.Equal(t, fs.EntryDirectory, receivedNotifications[1].entryType)

		close(pollChan)
	})

	t.Run("ParentCachedAtUnchanged", func(t *testing.T) {
		name := "test-notify-parent-cached-at"
		fsObj, err := createMetaCacheFs(t, localDir, cacheDB, name, 24*time.Hour)
		require.NoError(t, err)
		defer fsObj.(*Fs).Shutdown(context.Background())
		mcFs := fsObj.(*Fs)

		_, err = mcFs.List(context.Background(), "")
		require.NoError(t, err)
		_, err = mcFs.List(context.Background(), "subdir")
		require.NoError(t, err)

		subdirBefore, err := mcFs.cache.Get("subdir")
		require.NoError(t, err)
		beforeCachedAt := subdirBefore.CachedAt

		mcFs.receiveChangeNotify("subdir/file3.txt", fs.EntryObject)

		subdirAfter, err := mcFs.cache.Get("subdir")
		require.NoError(t, err)
		assert.False(t, subdirAfter.DirLoaded, "dir_loaded should be false")
		assert.Equal(t, beforeCachedAt, subdirAfter.CachedAt,
			"parent cached_at should remain unchanged after child notification")
	})
}

func TestExpiration(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.WriteFile(filepath.Join(localDir, "keep.txt"), []byte("test"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "delete.txt"), []byte("test"), 0600))

	fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-exp", 100*time.Millisecond)
	require.NoError(t, err)
	defer fsObj.(*Fs).Shutdown(context.Background())
	mcFs := fsObj.(*Fs)

	_, _ = mcFs.List(context.Background(), "")
	time.Sleep(200 * time.Millisecond)

	require.NoError(t, os.Remove(filepath.Join(localDir, "delete.txt")))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "new.txt"), []byte("test"), 0600))

	entries, err := mcFs.List(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	_, err = mcFs.cache.Get("delete.txt")
	assert.ErrorIs(t, err, ErrCacheNotFound)
}

func TestShutdown(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	name := "test-shutdown"
	fsObj, _ := createMetaCacheFs(t, localDir, cacheDB, name, 24*time.Hour)
	mcFs := fsObj.(*Fs)

	store := GetCacheStoreIfExists(name)
	require.NotNil(t, store)
	assert.Equal(t, int32(1), store.inUse.Load())

	start := time.Now()
	err := mcFs.Shutdown(context.Background())
	require.NoError(t, err)

	assert.Less(t, time.Since(start), 10*time.Second)
	assert.Nil(t, GetCacheStoreIfExists(name))
}

func TestNewFs_ErrorIsFile(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.WriteFile(filepath.Join(localDir, "test.txt"), []byte("test"), 0600))

	fsObj, err := NewFs(context.Background(), "test-is-file", "test.txt", configmap.Simple{
		"remote": localDir, "db_path": cacheDB,
	})

	assert.ErrorIs(t, err, fs.ErrorIsFile)
	if fsObj != nil {
		fsObj.(*Fs).Shutdown(context.Background())
	}
}

func TestCacheDir_InterfaceMethods(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.MkdirAll(filepath.Join(localDir, "testdir"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "testdir/file.txt"), []byte("test"), 0600))

	fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-cache-dir", 24*time.Hour)
	require.NoError(t, err)
	defer fsObj.(*Fs).Shutdown(context.Background())
	mcFs := fsObj.(*Fs)

	entries, err := mcFs.List(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// Second list to get from cache
	entries, err = mcFs.List(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, entries, 1)

	dir, ok := entries[0].(*CacheDir)
	require.True(t, ok, "expected CacheDir, got %T", entries[0])

	t.Run("Fs", func(t *testing.T) {
		assert.Equal(t, mcFs, dir.Fs())
	})

	t.Run("Remote", func(t *testing.T) {
		assert.Equal(t, "testdir", dir.Remote())
	})

	t.Run("ModTime", func(t *testing.T) {
		modTime := dir.ModTime(context.Background())
		assert.False(t, modTime.IsZero())
	})

	t.Run("Size", func(t *testing.T) {
		assert.Equal(t, int64(0), dir.Size())
	})

	t.Run("Items", func(t *testing.T) {
		assert.Equal(t, int64(-1), dir.Items())
	})

	t.Run("ID", func(t *testing.T) {
		assert.Equal(t, "", dir.ID())
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "testdir", dir.String())
	})

	t.Run("String_Nil", func(t *testing.T) {
		var nilDir *CacheDir
		assert.Equal(t, "<nil>", nilDir.String())
	})
}

func TestCacheObject_UnWrap(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.WriteFile(filepath.Join(localDir, "test.txt"), []byte("test"), 0600))

	fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-unwrap", 24*time.Hour)
	require.NoError(t, err)
	defer fsObj.(*Fs).Shutdown(context.Background())
	mcFs := fsObj.(*Fs)

	// Get object from NewObject (which fetches from remote)
	obj, err := mcFs.NewObject(context.Background(), "test.txt")
	require.NoError(t, err)

	t.Run("PutObject_UnWrappable", func(t *testing.T) {
		// PutObject wraps the underlying object
		_, ok := obj.(*PutObject)
		assert.True(t, ok, "expected PutObject, got %T", obj)
	})

	t.Run("CacheObject_UnWrap_BeforeGet", func(t *testing.T) {
		// List to get from cache
		entries, err := mcFs.List(context.Background(), "")
		require.NoError(t, err)
		require.Len(t, entries, 1)

		// Second list to hit cache
		entries, err = mcFs.List(context.Background(), "")
		require.NoError(t, err)
		require.Len(t, entries, 1)

		cacheObj, ok := entries[0].(*CacheObject)
		require.True(t, ok, "expected CacheObject, got %T", entries[0])

		// Before getWrapped is called, UnWrap should return nil
		unwrapped := cacheObj.UnWrap()
		assert.Nil(t, unwrapped)
	})

	t.Run("CacheObject_UnWrap_AfterGet", func(t *testing.T) {
		// List to get from cache
		entries, err := mcFs.List(context.Background(), "")
		require.NoError(t, err)

		cacheObj, ok := entries[0].(*CacheObject)
		require.True(t, ok)

		// Trigger getWrapped by calling Open
		rc, err := cacheObj.Open(context.Background())
		require.NoError(t, err)
		rc.Close()

		// Now UnWrap should return the wrapped object
		unwrapped := cacheObj.UnWrap()
		assert.NotNil(t, unwrapped)
	})
}

func TestCacheObject_Size_NilSize(t *testing.T) {
	entry := &CacheEntry{
		Path:   "test.txt",
		Parent: "",
		IsDir:  false,
		Size:   nil, // nil size
	}
	cacheObj := NewCacheObject(entry, nil)
	assert.Equal(t, int64(0), cacheObj.Size())
}

func TestFs_HelperMethods(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.WriteFile(filepath.Join(localDir, "test.txt"), []byte("test"), 0600))

	fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-helpers", 24*time.Hour)
	require.NoError(t, err)
	defer fsObj.(*Fs).Shutdown(context.Background())
	mcFs := fsObj.(*Fs)

	t.Run("GetCacheStore", func(t *testing.T) {
		store := mcFs.GetCacheStore()
		assert.NotNil(t, store)
		assert.Equal(t, mcFs.cache, store)
	})

	t.Run("WrapFs", func(t *testing.T) {
		wrapper := mcFs.WrapFs()
		assert.Nil(t, wrapper) // Initially nil
	})

	t.Run("SetWrapper", func(t *testing.T) {
		mcFs.SetWrapper(mcFs.wrapped)
		assert.Equal(t, mcFs.wrapped, mcFs.WrapFs())
		mcFs.SetWrapper(nil) // Reset
	})

	t.Run("SupportsChangeNotify", func(t *testing.T) {
		// Local backend doesn't support ChangeNotify
		assert.False(t, mcFs.SupportsChangeNotify())
	})
}

func TestFs_DirCacheFlush(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.WriteFile(filepath.Join(localDir, "file1.txt"), []byte("test"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "file2.txt"), []byte("test"), 0600))

	fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-flush", 24*time.Hour)
	require.NoError(t, err)
	defer fsObj.(*Fs).Shutdown(context.Background())
	mcFs := fsObj.(*Fs)

	// Populate cache
	_, err = mcFs.List(context.Background(), "")
	require.NoError(t, err)

	count, err := mcFs.cache.GetEntriesCount()
	require.NoError(t, err)
	assert.Greater(t, count, int64(0))

	// Flush cache
	mcFs.DirCacheFlush()

	// Verify cache is empty
	count, err = mcFs.cache.GetEntriesCount()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestNotifyChangeUpstreamIfNeeded(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.WriteFile(filepath.Join(localDir, "test.txt"), []byte("test"), 0600))

	fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-notify-upstream", 24*time.Hour)
	require.NoError(t, err)
	defer fsObj.(*Fs).Shutdown(context.Background())
	mcFs := fsObj.(*Fs)

	var notifications []struct {
		path      string
		entryType fs.EntryType
	}
	var mu sync.Mutex

	pollChan := make(chan time.Duration)
	mcFs.ChangeNotify(context.Background(), func(path string, entryType fs.EntryType) {
		mu.Lock()
		notifications = append(notifications, struct {
			path      string
			entryType fs.EntryType
		}{path, entryType})
		mu.Unlock()
	}, pollChan)

	mcFs.notifyChangeUpstreamIfNeeded("test/path", fs.EntryObject)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, notifications, 1)
	assert.Equal(t, "test/path", notifications[0].path)
	assert.Equal(t, fs.EntryObject, notifications[0].entryType)

	close(pollChan)
}

func TestCacheObject_Hash_FallbackPath(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.WriteFile(filepath.Join(localDir, "test.txt"), []byte("test content"), 0600))

	fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-hash-fallback", 24*time.Hour)
	require.NoError(t, err)
	defer fsObj.(*Fs).Shutdown(context.Background())
	mcFs := fsObj.(*Fs)

	t.Run("FallbackToCache_MatchingHashType", func(t *testing.T) {
		hashValue := "MD5:abc123def456"
		entry := &CacheEntry{
			Path:   "nonexistent.txt",
			Parent: "",
			IsDir:  false,
			Hash:   &hashValue,
		}
		cacheObj := &CacheObject{
			entry:      entry,
			f:          mcFs,
			wrappedErr: fmt.Errorf("simulated error"),
		}
		cacheObj.wrappedOnce.Do(func() {})

		result, err := cacheObj.Hash(context.Background(), hash.MD5)
		require.NoError(t, err)
		assert.Equal(t, "abc123def456", result)
	})

	t.Run("FallbackToCache_MismatchingHashType", func(t *testing.T) {
		hashValue := "MD5:abc123def456"
		entry := &CacheEntry{
			Path:   "nonexistent2.txt",
			Parent: "",
			IsDir:  false,
			Hash:   &hashValue,
		}
		cacheObj := &CacheObject{
			entry:      entry,
			f:          mcFs,
			wrappedErr: fmt.Errorf("simulated error"),
		}
		cacheObj.wrappedOnce.Do(func() {})

		result, err := cacheObj.Hash(context.Background(), hash.SHA1)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("FallbackToCache_NilHash", func(t *testing.T) {
		entry := &CacheEntry{
			Path:   "nonexistent3.txt",
			Parent: "",
			IsDir:  false,
			Hash:   nil,
		}
		cacheObj := &CacheObject{
			entry:      entry,
			f:          mcFs,
			wrappedErr: fmt.Errorf("simulated error"),
		}
		cacheObj.wrappedOnce.Do(func() {})

		result, err := cacheObj.Hash(context.Background(), hash.MD5)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("FallbackToCache_InvalidHashFormat", func(t *testing.T) {
		invalidHash := "invalidformat"
		entry := &CacheEntry{
			Path:   "nonexistent4.txt",
			Parent: "",
			IsDir:  false,
			Hash:   &invalidHash,
		}
		cacheObj := &CacheObject{
			entry:      entry,
			f:          mcFs,
			wrappedErr: fmt.Errorf("simulated error"),
		}
		cacheObj.wrappedOnce.Do(func() {})

		result, err := cacheObj.Hash(context.Background(), hash.MD5)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})
}

func TestCacheEntry_IsExpired(t *testing.T) {
	t.Run("NeverExpire_ZeroDuration", func(t *testing.T) {
		entry := &CacheEntry{
			CachedAt: time.Now().Add(-100 * time.Hour),
		}
		assert.False(t, entry.IsExpired(0))
	})

	t.Run("NeverExpire_NegativeDuration", func(t *testing.T) {
		entry := &CacheEntry{
			CachedAt: time.Now().Add(-100 * time.Hour),
		}
		assert.False(t, entry.IsExpired(-1*time.Hour))
	})

	t.Run("Expired", func(t *testing.T) {
		entry := &CacheEntry{
			CachedAt: time.Now().Add(-2 * time.Hour),
		}
		assert.True(t, entry.IsExpired(1*time.Hour))
	})

	t.Run("NotExpired", func(t *testing.T) {
		entry := &CacheEntry{
			CachedAt: time.Now(),
		}
		assert.False(t, entry.IsExpired(1*time.Hour))
	})
}

func TestFs_Copy(t *testing.T) {
	t.Run("ErrorCantCopy_LocalBackend", func(t *testing.T) {
		localDir, cacheDB, cleanup := setupTestEnv(t)
		defer cleanup()

		require.NoError(t, os.WriteFile(filepath.Join(localDir, "source.txt"), []byte("test content"), 0600))

		fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-copy-local", 24*time.Hour)
		require.NoError(t, err)
		defer fsObj.(*Fs).Shutdown(context.Background())
		mcFs := fsObj.(*Fs)

		srcObj, err := mcFs.NewObject(context.Background(), "source.txt")
		require.NoError(t, err)

		_, err = mcFs.Copy(context.Background(), srcObj, "dest.txt")
		assert.ErrorIs(t, err, fs.ErrorCantCopy)
	})

	t.Run("MemoryBackend", func(t *testing.T) {
		tempDir := t.TempDir()
		cacheDB := filepath.Join(tempDir, "cache.db")

		cacheStoreMu.Lock()
		for id := range cacheStores {
			if store, ok := cacheStores[id]; ok {
				store.close()
			}
			delete(cacheStores, id)
		}
		cacheStoreMu.Unlock()

		m := configmap.Simple{
			"remote":             ":memory:",
			"db_path":            cacheDB,
			"info_age":           fs.Duration(24 * time.Hour).String(),
			"change_notify_poll": fs.Duration(time.Minute).String(),
		}

		fsObj, err := NewFs(context.Background(), "test-copy-memory", "", m)
		require.NoError(t, err)
		defer fsObj.(*Fs).Shutdown(context.Background())
		mcFs := fsObj.(*Fs)

		content := []byte("test content for copy")
		srcObj, err := mcFs.Put(context.Background(), io.NopCloser(
			&bytesReader{data: content, pos: 0},
		), &mockObjectInfo{remote: "source.txt", size: int64(len(content))})
		require.NoError(t, err)

		t.Run("CopyWithPutObject", func(t *testing.T) {
			dstObj, err := mcFs.Copy(context.Background(), srcObj, "dest_put.txt")
			require.NoError(t, err)
			assert.Equal(t, "dest_put.txt", dstObj.Remote())
		})

		t.Run("CopyWithCacheObject", func(t *testing.T) {
			hashValue := "MD5:abc123"
			entry := &CacheEntry{
				Path:   "source.txt",
				Parent: "",
				IsDir:  false,
				Hash:   &hashValue,
			}
			cacheObj := NewCacheObject(entry, mcFs)

			dstObj, err := mcFs.Copy(context.Background(), cacheObj, "dest_cache.txt")
			require.NoError(t, err)
			assert.Equal(t, "dest_cache.txt", dstObj.Remote())
		})

		t.Run("CopyWithCacheObject_SourceNotFound", func(t *testing.T) {
			entry := &CacheEntry{
				Path:   "nonexistent.txt",
				Parent: "",
				IsDir:  false,
			}
			cacheObj := NewCacheObject(entry, mcFs)

			_, err := mcFs.Copy(context.Background(), cacheObj, "dest_error.txt")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "copy: failed to get source object")
		})
	})
}

type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

type mockObjectInfo struct {
	remote  string
	size    int64
	modTime time.Time
}

func (m *mockObjectInfo) Fs() fs.Info                         { return nil }
func (m *mockObjectInfo) Remote() string                      { return m.remote }
func (m *mockObjectInfo) Size() int64                         { return m.size }
func (m *mockObjectInfo) ModTime(_ context.Context) time.Time { return m.modTime }
func (m *mockObjectInfo) Storable() bool                      { return true }
func (m *mockObjectInfo) String() string                      { return m.remote }
func (m *mockObjectInfo) Hash(_ context.Context, _ hash.Type) (string, error) {
	return "", nil
}

func TestFs_Move(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.WriteFile(filepath.Join(localDir, "source.txt"), []byte("test content"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "source2.txt"), []byte("test content 2"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "source3.txt"), []byte("test content 3"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(localDir, "subdir"), 0750))

	fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-move", 24*time.Hour)
	require.NoError(t, err)
	defer fsObj.(*Fs).Shutdown(context.Background())
	mcFs := fsObj.(*Fs)

	t.Run("SuccessfulMove", func(t *testing.T) {
		srcObj, err := mcFs.NewObject(context.Background(), "source.txt")
		require.NoError(t, err)

		dstObj, err := mcFs.Move(context.Background(), srcObj, "dest.txt")
		require.NoError(t, err)
		assert.Equal(t, "dest.txt", dstObj.Remote())

		_, err = mcFs.NewObject(context.Background(), "source.txt")
		assert.Error(t, err)

		dstObjVerify, err := mcFs.NewObject(context.Background(), "dest.txt")
		require.NoError(t, err)
		assert.NotNil(t, dstObjVerify)
	})

	t.Run("MoveWithCacheObject", func(t *testing.T) {
		entry := &CacheEntry{
			Path:   "source2.txt",
			Parent: "",
			IsDir:  false,
		}
		cacheObj := NewCacheObject(entry, mcFs)

		dstObj, err := mcFs.Move(context.Background(), cacheObj, "dest_from_cache.txt")
		require.NoError(t, err)
		assert.Equal(t, "dest_from_cache.txt", dstObj.Remote())
	})

	t.Run("MoveToDifferentDirectory", func(t *testing.T) {
		srcObj, err := mcFs.NewObject(context.Background(), "source3.txt")
		require.NoError(t, err)

		dstObj, err := mcFs.Move(context.Background(), srcObj, "subdir/source3.txt")
		require.NoError(t, err)
		assert.Equal(t, "subdir/source3.txt", dstObj.Remote())
	})

	t.Run("MoveWithCacheObject_SourceNotFound", func(t *testing.T) {
		entry := &CacheEntry{
			Path:   "nonexistent.txt",
			Parent: "",
			IsDir:  false,
		}
		cacheObj := NewCacheObject(entry, mcFs)

		_, err := mcFs.Move(context.Background(), cacheObj, "dest_error.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "move: failed to get source object")
	})
}

func TestFs_DirMove(t *testing.T) {
	localDir, cacheDB, cleanup := setupTestEnv(t)
	defer cleanup()

	require.NoError(t, os.MkdirAll(filepath.Join(localDir, "srcdir"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "srcdir/file.txt"), []byte("test"), 0600))

	fsObj, err := createMetaCacheFs(t, localDir, cacheDB, "test-dirmove", 24*time.Hour)
	require.NoError(t, err)
	defer fsObj.(*Fs).Shutdown(context.Background())
	mcFs := fsObj.(*Fs)

	t.Run("SuccessfulDirMove", func(t *testing.T) {
		_, err := mcFs.List(context.Background(), "srcdir")
		require.NoError(t, err)

		err = mcFs.DirMove(context.Background(), mcFs, "srcdir", "dstdir")
		require.NoError(t, err)

		_, err = mcFs.List(context.Background(), "srcdir")
		assert.Error(t, err)

		entries, err := mcFs.List(context.Background(), "dstdir")
		require.NoError(t, err)
		assert.Len(t, entries, 1)
	})
}
