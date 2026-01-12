// Package metacache provides a metadata caching backend for rclone.
// It wraps remote backends to cache directory listings and file metadata,
// significantly speeding up sync operations when no changes have occurred.
//
// The cache is invalidated by:
// 1. TTL expiration (configurable via info_age option)
// 2. ChangeNotify callbacks from backends that support it (OneDrive, Google Drive, etc.)
package metacache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/cache"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/fspath"
	"github.com/rclone/rclone/fs/hash"
)

const (
	// DefInfoAge is the default TTL for cached metadata entries.
	DefInfoAge = 6 * time.Hour

	// DefChangeNotifyPoll is the default polling interval for ChangeNotify.
	DefChangeNotifyPoll = time.Minute
)

var (
	errRemoteRequired = errors.New("remote option is required")
	errDbPathRequired = errors.New("db_path option is required")
)

// Register the metacache backend
func init() {
	fs.Register(&fs.RegInfo{
		Name:        "metacache",
		Description: "Metadata cache wrapper for remote backends",
		NewFs:       NewFs,
		Options: []fs.Option{
			{
				Name:     "remote",
				Help:     "Remote to cache metadata for.",
				Required: true,
			},
			{
				Name:    "info_age",
				Help:    "How long to cache file structure information (e.g., 6h, 1d).",
				Default: fs.Duration(DefInfoAge),
			},
			{
				Name:    "change_notify_poll",
				Help:    "Polling interval for ChangeNotify (minimum 10s).",
				Default: fs.Duration(DefChangeNotifyPoll),
			},
			{
				Name:    "db_path",
				Help:    "Path to SQLite cache database file.",
				Default: "",
			},
		},
	})
}

// Options defines the configuration options for the metacache backend.
type Options struct {
	Remote           string      `config:"remote"`
	InfoAge          fs.Duration `config:"info_age"`
	ChangeNotifyPoll fs.Duration `config:"change_notify_poll"`
	DbPath           string      `config:"db_path"`
}

// Fs implements a metadata caching wrapper around a remote Fs.
// It caches directory listings and file metadata to speed up subsequent
// sync operations, especially for backends that support ChangeNotify.
type Fs struct {
	name     string
	root     string
	wrapped  fs.Fs
	wrapper  fs.Fs // Fs that is wrapping this Fs
	cache    *CacheStore
	opt      Options
	features *fs.Features

	// ChangeNotify management
	pollIntervalChan chan time.Duration

	// Notify function forwarding
	notifyMu    sync.Mutex
	notifyFuncs []func(string, fs.EntryType)

	// Shutdown context for goroutine lifecycle management
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// Compile-time interface checks
var (
	_ fs.Fs              = (*Fs)(nil)
	_ fs.UnWrapper       = (*Fs)(nil)
	_ fs.Wrapper         = (*Fs)(nil)
	_ fs.Shutdowner      = (*Fs)(nil)
	_ fs.ChangeNotifier  = (*Fs)(nil)
	_ fs.Purger          = (*Fs)(nil)
	_ fs.Copier          = (*Fs)(nil)
	_ fs.Mover           = (*Fs)(nil)
	_ fs.DirMover        = (*Fs)(nil)
	_ fs.DirCacheFlusher = (*Fs)(nil)
)

// NewFs creates a new Fs wrapping the remote specified in the config.
// It initializes the cache store and subscribes to ChangeNotify if supported.
func NewFs(ctx context.Context, name, rootPath string, m configmap.Mapper) (fs.Fs, error) {
	// Parse options
	opt := new(Options)
	if err := configstruct.Set(m, opt); err != nil {
		return nil, fmt.Errorf("failed to parse metacache options: %w", err)
	}

	// Validate options
	if opt.Remote == "" {
		return nil, errRemoteRequired
	}
	if opt.DbPath == "" {
		return nil, errDbPathRequired
	}

	// Get shared cache store
	store, err := GetCacheStore(name, opt.DbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache store: %w", err)
	}

	// Get the wrapped Fs
	remotePath := fspath.JoinRootPath(opt.Remote, rootPath)
	wrappedFs, wrapErr := cache.Get(ctx, remotePath)
	if wrapErr != nil && !errors.Is(wrapErr, fs.ErrorIsFile) {
		ReleaseCacheStore(name)
		return nil, fmt.Errorf("failed to get remote %q: %w", remotePath, wrapErr)
	}

	// Handle fs.ErrorIsFile - correct the root to parent directory
	var fsErr error
	if errors.Is(wrapErr, fs.ErrorIsFile) {
		fsErr = fs.ErrorIsFile
		rootPath = path.Dir(rootPath)
		if rootPath == "." || rootPath == "/" {
			rootPath = ""
		}
	}

	// Create shutdown context for goroutine lifecycle management
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	// Normalize rootPath: strip leading/trailing slashes to ensure consistent cache paths
	// This follows rclone's cache backend convention where root is represented as ""
	rootPath = strings.Trim(rootPath, "/")

	f := &Fs{
		name:             name,
		root:             rootPath,
		wrapped:          wrappedFs,
		cache:            store,
		opt:              *opt,
		pollIntervalChan: make(chan time.Duration, 1),
		shutdownCtx:      shutdownCtx,
		shutdownCancel:   shutdownCancel,
	}

	// Build features
	// We wrap the underlying Fs features but advertise our own capabilities
	f.features = (&fs.Features{
		CanHaveEmptyDirectories: true,
		// Advertise ChangeNotify if the wrapped Fs supports it
		ChangeNotify: f.ChangeNotify,
	}).Fill(ctx, f).Mask(ctx, wrappedFs).WrapsFs(f, wrappedFs)

	// Set Shutdown and Purge after Mask() to ensure they're not overwritten
	// by the wrapped Fs's lack of support for these features
	f.features.Shutdown = f.Shutdown
	f.features.Purge = f.Purge

	// Subscribe to ChangeNotify from the wrapped Fs
	if doChangeNotify := wrappedFs.Features().ChangeNotify; doChangeNotify != nil {
		// Send initial poll interval
		f.pollIntervalChan <- time.Duration(opt.ChangeNotifyPoll)
		doChangeNotify(ctx, f.receiveChangeNotify, f.pollIntervalChan)
		fs.Debugf(f, "Subscribed to ChangeNotify with poll interval %v", time.Duration(opt.ChangeNotifyPoll))
	} else {
		fs.Debugf(f, "Wrapped Fs does not support ChangeNotify, relying on TTL only")
	}

	// Return fs.ErrorIsFile if the root path pointed to a file
	return f, fsErr
}

// Name returns the name of the remote.
func (f *Fs) Name() string {
	return f.name
}

// Root returns the root path of the Fs.
func (f *Fs) Root() string {
	return f.root
}

// String returns a description of the Fs.
func (f *Fs) String() string {
	return fmt.Sprintf("metacache:%s/%s", f.name, f.root)
}

// Features returns the optional features of this Fs.
func (f *Fs) Features() *fs.Features {
	return f.features
}

// UnWrap returns the underlying Fs.
func (f *Fs) UnWrap() fs.Fs {
	return f.wrapped
}

// Precision returns the precision of the wrapped Fs.
func (f *Fs) Precision() time.Duration {
	return f.wrapped.Precision()
}

// Hashes returns the supported hash types of the wrapped Fs.
func (f *Fs) Hashes() hash.Set {
	return f.wrapped.Hashes()
}

// cachePath converts a relative path to the full cache path.
// The cache stores paths relative to the connection root, not the Fs root.
func (f *Fs) cachePath(relativePath string) string {
	// Combine Fs root with relative path to get full path from connection root
	fullPath := path.Join(f.root, relativePath)
	// Clean the path to normalize it
	// NOTE: path.Clean("") returns "." which we normalize to "" for root directory
	cleaned := path.Clean(fullPath)
	if cleaned == "." {
		return ""
	}
	// Ensure no leading slash for consistent cache keys
	cleaned = strings.TrimPrefix(cleaned, "/")
	return cleaned
}

// normalizeParent returns the normalized parent path for cache entries.
// Empty string represents the root directory (used for entries at the root level).
// This ensures consistent parent representation across all cache operations.
func normalizeParent(p string) string {
	parent := path.Dir(p)
	if parent == "." || parent == "" || parent == "/" {
		return ""
	}
	return strings.TrimPrefix(parent, "/")
}

// List lists the objects and directories in dir into entries.
// It first checks the cache, and if the directory is loaded and not expired,
// returns the cached entries. Otherwise, it fetches from the remote and caches the result.
//
// FR-009: When a directory expires, all cached children are deleted and the directory
// is fully refreshed from the remote with dir_loaded=true and cached_at reset.
func (f *Fs) List(ctx context.Context, dir string) (fs.DirEntries, error) {
	cachePath := f.cachePath(dir)
	infoAge := time.Duration(f.opt.InfoAge)

	// Check if directory is cached and loaded (not expired)
	dirLoaded, err := f.cache.IsDirLoaded(cachePath, infoAge)
	if err != nil {
		fs.Debugf(f, "Error checking cache for %q: %v", cachePath, err)
	}

	if dirLoaded {
		// Cache hit - return cached entries
		cachedEntries, err := f.cache.ListChildren(cachePath)
		if err != nil {
			fs.Debugf(f, "Error listing cached children for %q: %v", cachePath, err)
		} else {
			fs.Debugf(f, "Cache hit for directory %q (%d entries)", cachePath, len(cachedEntries))
			return f.entriesToDirEntries(cachedEntries), nil
		}
	}

	// Cache miss or expired - fetch from remote and do full refresh
	fs.Debugf(f, "Cache miss/expired for directory %q, fetching from remote", cachePath)
	entries, err := f.wrapped.List(ctx, dir)
	if err != nil {
		return nil, err
	}

	// FR-009: Full refresh - delete old children, insert new entries, reset dir_loaded
	// Execute synchronously to ensure cache is consistent before returning
	f.refreshDirectoryCache(cachePath, entries)

	return entries, nil
}

// refreshDirectoryCache performs an incremental refresh of a directory's cache.
// This method preserves subdirectories' dir_loaded state to avoid cascading refreshes.
func (f *Fs) refreshDirectoryCache(dirPath string, entries fs.DirEntries) {
	cacheEntries := f.entriesToCacheEntries(dirPath, entries)
	if err := f.cache.RefreshDirectory(dirPath, cacheEntries); err != nil {
		fs.Logf(f, "Failed to refresh cache for %q: %v", dirPath, err)
	} else {
		fs.Debugf(f, "Refreshed cache for directory %q (%d entries)", dirPath, len(entries))
	}
}

// entriesToCacheEntries converts fs.DirEntries to CacheEntry slice for caching.
func (f *Fs) entriesToCacheEntries(dirPath string, entries fs.DirEntries) []*CacheEntry {
	now := time.Now()
	cacheEntries := make([]*CacheEntry, 0, len(entries))

	for _, entry := range entries {
		entryPath := f.cachePath(entry.Remote())
		parentPath := dirPath
		if parentPath == "." {
			parentPath = ""
		}

		cacheEntry := &CacheEntry{
			Path:     entryPath,
			Parent:   parentPath,
			ModTime:  entry.ModTime(context.Background()),
			CachedAt: now,
		}

		switch e := entry.(type) {
		case fs.Object:
			cacheEntry.IsDir = false
			size := e.Size()
			cacheEntry.Size = &size
			hashType := f.wrapped.Hashes().GetOne()
			if hashType != hash.None {
				if hashStr, err := e.Hash(context.Background(), hashType); err == nil && hashStr != "" {
					formattedHash := fmt.Sprintf("%s:%s", hashType.String(), hashStr)
					cacheEntry.Hash = &formattedHash
				}
			}
		case fs.Directory:
			cacheEntry.IsDir = true
			cacheEntry.DirLoaded = false
		}

		cacheEntries = append(cacheEntries, cacheEntry)
	}

	return cacheEntries
}

// entriesToDirEntries converts CacheEntries to fs.DirEntries.
func (f *Fs) entriesToDirEntries(entries []*CacheEntry) fs.DirEntries {
	result := make(fs.DirEntries, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			// CacheDir needs relative path for Remote()
			dirEntry := *entry
			if f.root != "" && f.root != "." {
				dirEntry.Path = strings.TrimPrefix(entry.Path, f.root+"/")
				if dirEntry.Path == entry.Path {
					dirEntry.Path = strings.TrimPrefix(entry.Path, f.root)
				}
			}
			result = append(result, NewCacheDir(&dirEntry, f))
		} else {
			result = append(result, NewCacheObject(entry, f))
		}
	}
	return result
}

// NewObject finds the Object at remote.
// It first checks the cache, and if found and not expired, returns the cached object.
// Otherwise, it fetches from the remote and caches the result.
func (f *Fs) NewObject(ctx context.Context, remote string) (fs.Object, error) {
	cachePath := f.cachePath(remote)
	infoAge := time.Duration(f.opt.InfoAge)

	// Check cache first
	entry, err := f.cache.Get(cachePath)
	if err == nil && !entry.IsExpired(infoAge) && !entry.IsDir {
		fs.Debugf(f, "Cache hit for object %q", cachePath)
		return NewCacheObject(entry, f), nil
	}

	// Cache miss or expired - fetch from remote
	fs.Debugf(f, "Cache miss for object %q, fetching from remote", cachePath)
	obj, err := f.wrapped.NewObject(ctx, remote)
	if err != nil {
		return nil, err
	}

	// Cache the object
	go f.cacheObject(cachePath, obj)

	// Return a PutObject wrapper so that Remove() updates the cache
	return &PutObject{Object: obj, f: f}, nil
}

// cacheObject caches a single object's metadata.
func (f *Fs) cacheObject(objPath string, obj fs.Object) {
	now := time.Now()
	size := obj.Size()
	entry := &CacheEntry{
		Path:     objPath,
		Parent:   normalizeParent(objPath),
		ModTime:  obj.ModTime(context.Background()),
		IsDir:    false,
		Size:     &size,
		CachedAt: now,
	}

	// Try to get hash - check if Fs supports any hash type first
	hashType := f.wrapped.Hashes().GetOne()
	if hashType != hash.None {
		if hashStr, err := obj.Hash(context.Background(), hashType); err == nil && hashStr != "" {
			// Format as "algorithm:value" per spec
			formattedHash := fmt.Sprintf("%s:%s", hashType.String(), hashStr)
			entry.Hash = &formattedHash
		}
		// fs.ErrorCantHash and other errors are silently ignored, hash remains nil
	}

	if err := f.cache.Set(entry); err != nil {
		fs.Errorf(f, "Failed to cache object %q: %v", objPath, err)
	}
}

// Put uploads an object to the remote and caches it.
func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	obj, err := f.wrapped.Put(ctx, in, src, options...)
	if err != nil {
		return nil, err
	}

	// Invalidate cache for all ancestor directories up to (but not including) the Fs root
	// This ensures that any parent Fs with a higher root will also have its cache invalidated
	remotePath := src.Remote()
	for {
		parentPath := path.Dir(remotePath)
		if parentPath == "." || parentPath == "" || parentPath == remotePath {
			break
		}
		cachePath := f.cachePath(parentPath)
		if err := f.cache.SetDirLoaded(cachePath, false); err != nil {
			fs.Debugf(f, "Failed to invalidate cache for %q: %v", cachePath, err)
		}
		remotePath = parentPath
	}
	// Also invalidate the root directory of this Fs
	if err := f.cache.SetDirLoaded(f.cachePath(""), false); err != nil {
		fs.Debugf(f, "Failed to invalidate cache for root: %v", err)
	}

	// Notify upstream if wrapped Fs doesn't support ChangeNotify
	f.notifyChangeUpstreamIfNeeded(normalizeParent(src.Remote()), fs.EntryDirectory)

	// Return a PutObject that wraps the returned object so Remove() updates the cache
	return &PutObject{Object: obj, f: f}, nil
}

// Mkdir creates a directory.
func (f *Fs) Mkdir(ctx context.Context, dir string) error {
	if err := f.wrapped.Mkdir(ctx, dir); err != nil {
		return err
	}

	// Invalidate parent directory cache
	parentDir := normalizeParent(dir)
	parentPath := f.cachePath(parentDir)
	if err := f.cache.SetDirLoaded(parentPath, false); err != nil {
		fs.Debugf(f, "Failed to invalidate cache for %q: %v", parentPath, err)
	}

	// Notify upstream if wrapped Fs doesn't support ChangeNotify
	f.notifyChangeUpstreamIfNeeded(parentDir, fs.EntryDirectory)

	return nil
}

// Rmdir removes a directory.
func (f *Fs) Rmdir(ctx context.Context, dir string) error {
	if err := f.wrapped.Rmdir(ctx, dir); err != nil {
		return err
	}

	// Remove from cache
	cachePath := f.cachePath(dir)
	if err := f.cache.Delete(cachePath); err != nil {
		fs.Debugf(f, "Failed to delete cache entry for %q: %v", cachePath, err)
	}

	// Invalidate parent directory cache
	parentDir := normalizeParent(dir)
	parentPath := f.cachePath(parentDir)
	if err := f.cache.SetDirLoaded(parentPath, false); err != nil {
		fs.Debugf(f, "Failed to invalidate cache for %q: %v", parentPath, err)
	}

	// Notify upstream if wrapped Fs doesn't support ChangeNotify
	f.notifyChangeUpstreamIfNeeded(parentDir, fs.EntryDirectory)

	return nil
}

// Purge removes a directory and all of its contents.
// It delegates to the wrapped Fs's Purge if available, otherwise uses a fallback.
func (f *Fs) Purge(ctx context.Context, dir string) error {
	// First check if the directory exists
	_, err := f.wrapped.List(ctx, dir)
	if err != nil {
		// If we can't list, the directory doesn't exist
		return fs.ErrorDirNotFound
	}

	// Check if wrapped Fs has Purge
	if doPurge := f.wrapped.Features().Purge; doPurge != nil {
		err := doPurge(ctx, dir)
		if err != nil {
			return err
		}
	} else {
		// Fallback: recursively delete contents
		err := f.purgeRecursive(ctx, dir)
		if err != nil {
			return err
		}
	}

	// Clean up cache for this directory and all descendants
	cachePath := f.cachePath(dir)
	if err := f.cache.DeleteRecursive(cachePath); err != nil {
		fs.Debugf(f, "Failed to delete recursive cache entries for %q: %v", cachePath, err)
	}

	// Invalidate parent directory cache
	parentDir := normalizeParent(dir)
	parentPath := f.cachePath(parentDir)
	if err := f.cache.SetDirLoaded(parentPath, false); err != nil {
		fs.Debugf(f, "Failed to invalidate cache for %q: %v", parentPath, err)
	}

	// Notify upstream if wrapped Fs doesn't support ChangeNotify
	f.notifyChangeUpstreamIfNeeded(parentDir, fs.EntryDirectory)

	return nil
}

// purgeRecursive recursively deletes all contents of a directory.
func (f *Fs) purgeRecursive(ctx context.Context, dir string) error {
	entries, err := f.wrapped.List(ctx, dir)
	if err != nil {
		return err
	}

	// Delete all files and subdirectories
	for _, entry := range entries {
		switch x := entry.(type) {
		case fs.Object:
			if err := x.Remove(ctx); err != nil {
				return err
			}
		case fs.Directory:
			subDir := x.Remote()
			if err := f.purgeRecursive(ctx, subDir); err != nil {
				return err
			}
		}
	}

	// Remove the directory itself
	return f.wrapped.Rmdir(ctx, dir)
}

// notifyChangeUpstream notifies all registered upstream subscribers of a change.
// This is used to notify VFS and other wrappers of changes.
func (f *Fs) notifyChangeUpstream(remote string, entryType fs.EntryType) {
	f.notifyMu.Lock()
	notifyFuncs := f.notifyFuncs
	f.notifyMu.Unlock()

	for _, fn := range notifyFuncs {
		fn(remote, entryType)
	}
}

// notifyChangeUpstreamIfNeeded checks if the wrapped remote doesn't support ChangeNotify,
// and if so, notifies upstream subscribers of changes.
// Following the pattern from rclone's cache backend, if the wrapped Fs supports ChangeNotify,
// it will handle notifications automatically, so we don't need to duplicate them.
func (f *Fs) notifyChangeUpstreamIfNeeded(remote string, entryType fs.EntryType) {
	if f.wrapped.Features().ChangeNotify == nil {
		f.notifyChangeUpstream(remote, entryType)
	}
}

// receiveChangeNotify is the callback function for ChangeNotify from the wrapped Fs.
// It marks the affected paths as stale in the cache.
//
// IMPORTANT: As per FR-002, this callback updates/inserts objects regardless of
// dir_loaded state to maintain cache consistency.
func (f *Fs) receiveChangeNotify(relativePath string, entryType fs.EntryType) {
	// Convert to full cache path
	fullPath := f.cachePath(relativePath)

	fs.Debugf(f, "ChangeNotify: %s (%v)", fullPath, entryType)

	// Update last notify time for monitoring
	f.cache.SetLastNotifyTime(time.Now())

	// Mark the path as stale (non-recursive to avoid full cache invalidation on backends like OneDrive)
	if err := f.cache.MarkStale(fullPath); err != nil {
		fs.Errorf(f, "Failed to mark %q as stale: %v", fullPath, err)
	}

	// Also invalidate the parent directory's dir_loaded flag
	// NOTE: We always invalidate, including root (parentPath="").
	// This ensures that changes to files/directories at the root level
	// will trigger a refresh on the next List("") call.
	parentPath := normalizeParent(fullPath)
	if err := f.cache.SetDirLoaded(parentPath, false); err != nil {
		fs.Debugf(f, "Failed to invalidate parent cache for %q: %v", parentPath, err)
	}

	// If it's a directory, also invalidate the directory's own dir_loaded flag
	if entryType == fs.EntryDirectory {
		if err := f.cache.SetDirLoaded(fullPath, false); err != nil {
			fs.Debugf(f, "Failed to invalidate directory cache for %q: %v", fullPath, err)
		}
	}

	// Forward notification to any subscribers
	f.notifyMu.Lock()
	notifyFuncs := f.notifyFuncs
	f.notifyMu.Unlock()

	for _, fn := range notifyFuncs {
		fn(relativePath, entryType)
	}
}

// ChangeNotify implements fs.ChangeNotifier interface.
// It allows consumers of this Fs to subscribe to change notifications.
// Following the pattern from rclone's cache backend, we register the notify function
// and consume the poll interval channel without forwarding it.
// The poll interval for the wrapped Fs is already configured in NewFs.
func (f *Fs) ChangeNotify(_ context.Context, notifyFunc func(string, fs.EntryType), pollIntervalChan <-chan time.Duration) {
	f.notifyMu.Lock()
	defer f.notifyMu.Unlock()
	fs.Debugf(f, "subscribing to ChangeNotify")
	f.notifyFuncs = append(f.notifyFuncs, notifyFunc)
	// Consume the poll interval channel to avoid blocking callers
	// Poll interval to wrapped Fs is already set up in NewFs
	go func() {
		for {
			select {
			case <-f.shutdownCtx.Done():
				return
			case _, ok := <-pollIntervalChan:
				if !ok {
					return
				}
			}
		}
	}()
}

// Shutdown implements fs.Shutdowner interface.
// It stops all goroutines and releases the cache store.
func (f *Fs) Shutdown(ctx context.Context) error {
	fs.Debugf(f, "Shutting down metacache")

	// Cancel shutdown context to signal any goroutines to exit
	if f.shutdownCancel != nil {
		f.shutdownCancel()
	}

	// Close the poll interval channel to stop ChangeNotify subscription to wrapped Fs
	// Note: We use notifyMu to protect the channel close since it's the only mutex we have
	f.notifyMu.Lock()
	if f.pollIntervalChan != nil {
		close(f.pollIntervalChan)
		f.pollIntervalChan = nil
	}
	f.notifyMu.Unlock()

	// Release cache store reference
	ReleaseCacheStore(f.name)

	// Shutdown wrapped Fs if it supports it
	if shutdowner, ok := f.wrapped.(fs.Shutdowner); ok {
		return shutdowner.Shutdown(ctx)
	}

	return nil
}

// SupportsChangeNotify returns true if the wrapped Fs supports ChangeNotify.
func (f *Fs) SupportsChangeNotify() bool {
	return f.wrapped.Features().ChangeNotify != nil
}

// Copy src to this remote using server-side copy operations.
// It returns the destination Object and a possible error.
// Will only be called if src.Fs().Name() == f.Name()
// If it isn't possible then return fs.ErrorCantCopy
func (f *Fs) Copy(ctx context.Context, src fs.Object, remote string) (fs.Object, error) {
	do := f.wrapped.Features().Copy
	if do == nil {
		return nil, fs.ErrorCantCopy
	}

	// Unwrap the source object if it's wrapped by us
	srcObj := src
	if po, ok := src.(*PutObject); ok {
		srcObj = po.Object
	} else if _, ok := src.(*CacheObject); ok {
		// For CacheObject, we need to get the real object from the wrapped Fs
		var err error
		srcObj, err = f.wrapped.NewObject(ctx, src.Remote())
		if err != nil {
			return nil, fmt.Errorf("copy: failed to get source object: %w", err)
		}
	}

	// Perform the copy
	obj, err := do(ctx, srcObj, remote)
	if err != nil {
		return nil, err
	}
	fs.Debugf(f, "copy: %s -> %s", src.Remote(), remote)

	// Invalidate destination parent directory cache
	dstParentDir := normalizeParent(remote)
	dstParentPath := f.cachePath(dstParentDir)
	if err := f.cache.SetDirLoaded(dstParentPath, false); err != nil {
		fs.Debugf(f, "Failed to invalidate cache for %q: %v", dstParentPath, err)
	}

	// Notify upstream if wrapped Fs doesn't support ChangeNotify
	f.notifyChangeUpstreamIfNeeded(dstParentDir, fs.EntryDirectory)

	return &PutObject{Object: obj, f: f}, nil
}

// Move src to this remote using server-side move operations.
// This is stored with the remote path given.
// It returns the destination Object and a possible error.
// Will only be called if src.Fs().Name() == f.Name()
// If it isn't possible then return fs.ErrorCantMove
func (f *Fs) Move(ctx context.Context, src fs.Object, remote string) (fs.Object, error) {
	do := f.wrapped.Features().Move
	if do == nil {
		return nil, fs.ErrorCantMove
	}

	// Unwrap the source object if it's wrapped by us
	srcObj := src
	srcRemote := src.Remote()
	if po, ok := src.(*PutObject); ok {
		srcObj = po.Object
	} else if _, ok := src.(*CacheObject); ok {
		// For CacheObject, we need to get the real object from the wrapped Fs
		var err error
		srcObj, err = f.wrapped.NewObject(ctx, src.Remote())
		if err != nil {
			return nil, fmt.Errorf("move: failed to get source object: %w", err)
		}
	}

	// Perform the move
	obj, err := do(ctx, srcObj, remote)
	if err != nil {
		return nil, err
	}
	fs.Debugf(f, "move: %s -> %s", srcRemote, remote)

	// Remove source from cache
	srcCachePath := f.cachePath(srcRemote)
	if err := f.cache.Delete(srcCachePath); err != nil {
		fs.Debugf(f, "Failed to delete cache entry for %q: %v", srcCachePath, err)
	}

	// Invalidate source parent directory cache
	srcParentDir := normalizeParent(srcRemote)
	srcParentPath := f.cachePath(srcParentDir)
	if err := f.cache.SetDirLoaded(srcParentPath, false); err != nil {
		fs.Debugf(f, "Failed to invalidate cache for %q: %v", srcParentPath, err)
	}

	// Invalidate destination parent directory cache
	dstParentDir := normalizeParent(remote)
	dstParentPath := f.cachePath(dstParentDir)
	if err := f.cache.SetDirLoaded(dstParentPath, false); err != nil {
		fs.Debugf(f, "Failed to invalidate cache for %q: %v", dstParentPath, err)
	}

	// Notify upstream if wrapped Fs doesn't support ChangeNotify
	f.notifyChangeUpstreamIfNeeded(srcParentDir, fs.EntryDirectory)
	if srcParentDir != dstParentDir {
		f.notifyChangeUpstreamIfNeeded(dstParentDir, fs.EntryDirectory)
	}

	return &PutObject{Object: obj, f: f}, nil
}

// DirMove moves src, srcRemote to this remote at dstRemote
// using server-side move operations.
// Will only be called if src.Fs().Name() == f.Name()
// If it isn't possible then return fs.ErrorCantDirMove
// If destination exists then return fs.ErrorDirExists
func (f *Fs) DirMove(ctx context.Context, src fs.Fs, srcRemote, dstRemote string) error {
	do := f.wrapped.Features().DirMove
	if do == nil {
		return fs.ErrorCantDirMove
	}

	// Unwrap the source Fs if it's wrapped by us
	srcFs := src
	var srcMc *Fs
	if mc, ok := src.(*Fs); ok {
		srcMc = mc
		srcFs = mc.wrapped
	}

	// Perform the move
	err := do(ctx, srcFs, srcRemote, dstRemote)
	if err != nil {
		return err
	}
	fs.Debugf(f, "dirmove: %s -> %s", srcRemote, dstRemote)

	// Remove source directory and its descendants from cache
	// Use source Fs's cachePath if available, otherwise use our own
	if srcMc != nil {
		srcCachePath := srcMc.cachePath(srcRemote)
		if err := srcMc.cache.DeleteRecursive(srcCachePath); err != nil {
			fs.Debugf(f, "Failed to delete recursive cache entries for %q: %v", srcCachePath, err)
		}

		// Invalidate source parent directory cache
		srcParentDir := normalizeParent(srcRemote)
		srcParentPath := srcMc.cachePath(srcParentDir)
		if err := srcMc.cache.SetDirLoaded(srcParentPath, false); err != nil {
			fs.Debugf(f, "Failed to invalidate cache for %q: %v", srcParentPath, err)
		}

		// Notify upstream if wrapped Fs doesn't support ChangeNotify
		srcMc.notifyChangeUpstreamIfNeeded(srcParentDir, fs.EntryDirectory)
	}

	// Invalidate destination parent directory cache
	dstParentDir := normalizeParent(dstRemote)
	dstParentPath := f.cachePath(dstParentDir)
	if err := f.cache.SetDirLoaded(dstParentPath, false); err != nil {
		fs.Debugf(f, "Failed to invalidate cache for %q: %v", dstParentPath, err)
	}

	// Notify upstream if wrapped Fs doesn't support ChangeNotify
	f.notifyChangeUpstreamIfNeeded(dstParentDir, fs.EntryDirectory)

	return nil
}

// DirCacheFlush resets the directory cache - used in testing as an optional interface.
func (f *Fs) DirCacheFlush() {
	fs.Debugf(f, "Flushing directory cache")
	// Clear all cached data for this connection
	if _, err := f.cache.Clear(); err != nil {
		fs.Errorf(f, "Failed to clear cache: %v", err)
	}
}

// GetCacheStore returns the underlying cache store for testing/monitoring.
func (f *Fs) GetCacheStore() *CacheStore {
	return f.cache
}

// WrapFs returns the Fs that is wrapping this Fs.
func (f *Fs) WrapFs() fs.Fs {
	return f.wrapper
}

// SetWrapper sets the Fs that is wrapping this Fs.
func (f *Fs) SetWrapper(wrapper fs.Fs) {
	f.wrapper = wrapper
}

// Compile-time interface checks for wrapper objects
var (
	_ fs.Object = (*PutObject)(nil)
)

// PutObject wraps an fs.Object returned from Put() so that Remove() updates the cache.
// This is necessary because Put() returns the wrapped Fs's object, and we need to
// intercept Remove() calls to update the cache.
type PutObject struct {
	fs.Object
	f *Fs
}

// Fs returns the parent Fs.
func (o *PutObject) Fs() fs.Info {
	return o.f
}

// Remote returns the remote path relative to the Fs root.
func (o *PutObject) Remote() string {
	// The wrapped object's Remote() is relative to the wrapped Fs
	// We need to return the path relative to our Fs root
	return o.Object.Remote()
}

// Remove removes the object and updates the cache.
func (o *PutObject) Remove(ctx context.Context) error {
	err := o.Object.Remove(ctx)
	if err != nil {
		return err
	}

	// Delete from cache
	cachePath := o.f.cachePath(o.Remote())
	if delErr := o.f.cache.Delete(cachePath); delErr != nil {
		fs.Debugf(o.f, "Failed to delete cache entry for %q: %v", cachePath, delErr)
	}

	// Invalidate all ancestor directories
	remotePath := o.Remote()
	for {
		parentPath := path.Dir(remotePath)
		if parentPath == "." || parentPath == "" || parentPath == remotePath {
			break
		}
		parentCachePath := o.f.cachePath(parentPath)
		if setErr := o.f.cache.SetDirLoaded(parentCachePath, false); setErr != nil {
			fs.Debugf(o.f, "Failed to invalidate cache for %q: %v", parentCachePath, setErr)
		}
		remotePath = parentPath
	}
	// Also invalidate the root directory
	if setErr := o.f.cache.SetDirLoaded(o.f.cachePath(""), false); setErr != nil {
		fs.Debugf(o.f, "Failed to invalidate cache for root: %v", setErr)
	}

	// Notify upstream if wrapped Fs doesn't support ChangeNotify
	// Notify both the object and its parent directory
	o.f.notifyChangeUpstreamIfNeeded(o.Remote(), fs.EntryObject)
	parentDir := normalizeParent(o.Remote())
	o.f.notifyChangeUpstreamIfNeeded(parentDir, fs.EntryDirectory)

	return nil
}
