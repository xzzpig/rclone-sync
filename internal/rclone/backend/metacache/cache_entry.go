// Package metacache provides metadata caching for rclone backends.
// It caches file and directory metadata in SQLite to speed up subsequent sync operations,
// especially for backends that support ChangeNotify (like OneDrive, Google Drive).
package metacache

import (
	"context"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/hash"
)

// CacheEntry represents a cached file or directory entry.
// This struct stores the essential metadata used by rclone sync/bisync
// for file comparison (ModTime, Size, Hash).
type CacheEntry struct {
	// Path is the full path relative to the connection root (primary key)
	Path string

	// Parent is the parent directory path, used for efficient List() queries
	Parent string

	// ModTime is the modification time of the file/directory
	ModTime time.Time

	// IsDir indicates whether this entry is a directory
	IsDir bool

	// Size is the file size in bytes (nil for directories)
	Size *int64

	// Hash is the content hash in format "algorithm:value" (e.g., "md5:abc123")
	// Only applicable to files, nil for directories
	Hash *string

	// DirLoaded indicates whether all children of this directory have been loaded.
	// Only applicable to directories.
	DirLoaded bool

	// CachedAt is the timestamp when this entry was cached.
	// Used for TTL expiration checking.
	CachedAt time.Time
}

// IsExpired checks if the cache entry has expired based on the given infoAge TTL.
// If infoAge is 0 or negative, the entry never expires (per FR-004).
func (e *CacheEntry) IsExpired(infoAge time.Duration) bool {
	if infoAge <= 0 {
		return false // 0 or negative means never expire
	}
	return time.Now().After(e.CachedAt.Add(infoAge))
}

// Compile-time interface checks
var (
	_ fs.Object          = (*CacheObject)(nil)
	_ fs.ObjectUnWrapper = (*CacheObject)(nil)
	_ fs.Directory       = (*CacheDir)(nil)
)

// CacheObject wraps a CacheEntry to implement the fs.Object interface.
// It provides cached metadata for fast access while delegating actual
// file operations (Open, Update, Remove) to the underlying wrapped object.
type CacheObject struct {
	entry       *CacheEntry
	f           *Fs
	wrappedOnce sync.Once
	wrapped     fs.Object
	wrappedErr  error
}

// NewCacheObject creates a new CacheObject from a CacheEntry.
func NewCacheObject(entry *CacheEntry, f *Fs) *CacheObject {
	return &CacheObject{entry: entry, f: f}
}

// getWrapped lazily fetches the underlying wrapped object.
// This is called when we need to perform actual file operations.
func (o *CacheObject) getWrapped(ctx context.Context) (fs.Object, error) {
	o.wrappedOnce.Do(func() {
		// Get the relative path from the Fs root
		remote := o.entry.Path
		if o.f.root != "" && o.f.root != "." {
			remote = strings.TrimPrefix(o.entry.Path, o.f.root+"/")
			if remote == o.entry.Path {
				remote = strings.TrimPrefix(o.entry.Path, o.f.root)
			}
		}
		o.wrapped, o.wrappedErr = o.f.wrapped.NewObject(ctx, remote)
	})
	return o.wrapped, o.wrappedErr
}

// Fs returns the parent Fs.
func (o *CacheObject) Fs() fs.Info {
	return o.f
}

// Remote returns the remote path.
func (o *CacheObject) Remote() string {
	// Return path relative to Fs root
	remote := o.entry.Path
	if o.f.root != "" && o.f.root != "." {
		remote = strings.TrimPrefix(o.entry.Path, o.f.root+"/")
		if remote == o.entry.Path {
			remote = strings.TrimPrefix(o.entry.Path, o.f.root)
		}
	}
	return remote
}

// ModTime returns the modification time of the Object.
func (o *CacheObject) ModTime(_ context.Context) time.Time {
	return o.entry.ModTime
}

// Size returns the size of the Object in bytes.
func (o *CacheObject) Size() int64 {
	if o.entry.Size == nil {
		return 0
	}
	return *o.entry.Size
}

// String returns a description of the Object.
func (o *CacheObject) String() string {
	if o == nil {
		return "<nil>"
	}
	return o.Remote()
}

// Hash returns the hash of an object returning an empty string if it's not available.
// We always fetch the hash from the wrapped object to ensure data integrity,
// as cached hashes may become stale if the file is modified.
func (o *CacheObject) Hash(ctx context.Context, t hash.Type) (string, error) {
	// Always get hash from wrapped object to ensure correctness
	wrapped, err := o.getWrapped(ctx)
	if err != nil {
		// If we can't get wrapped object, fall back to cached hash
		if o.entry.Hash == nil {
			return "", nil
		}
		// Parse hash format "algorithm:value"
		parts := strings.SplitN(*o.entry.Hash, ":", 2)
		if len(parts) != 2 {
			return "", nil
		}
		// Check if the requested hash type matches
		if strings.EqualFold(parts[0], t.String()) {
			return parts[1], nil
		}
		return "", nil
	}
	return wrapped.Hash(ctx, t)
}

// Storable returns whether this object can be stored.
func (o *CacheObject) Storable() bool {
	return true
}

// SetModTime sets the metadata on the object to set the modification date.
func (o *CacheObject) SetModTime(ctx context.Context, t time.Time) error {
	wrapped, err := o.getWrapped(ctx)
	if err != nil {
		return err
	}
	err = wrapped.SetModTime(ctx, t)
	if err == nil {
		// Update cached entry and persist
		o.entry.ModTime = t
		o.entry.CachedAt = time.Now()
		_ = o.f.cache.Set(o.entry)
	}
	return err
}

// Open opens the Object for read.
func (o *CacheObject) Open(ctx context.Context, options ...fs.OpenOption) (io.ReadCloser, error) {
	wrapped, err := o.getWrapped(ctx)
	if err != nil {
		return nil, err
	}
	return wrapped.Open(ctx, options...)
}

// Update the object with the contents of the io.Reader.
func (o *CacheObject) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	wrapped, err := o.getWrapped(ctx)
	if err != nil {
		return err
	}
	err = wrapped.Update(ctx, in, src, options...)
	if err == nil {
		// Update cached entry with new metadata from src
		o.entry.ModTime = src.ModTime(ctx)
		size := src.Size()
		o.entry.Size = &size
		o.entry.Hash = nil // Clear hash as content has changed
		o.entry.CachedAt = time.Now()
		_ = o.f.cache.Set(o.entry)

		// Invalidate parent directory cache
		parentPath := normalizeParent(o.entry.Path)
		if parentPath != "" {
			_ = o.f.cache.SetDirLoaded(parentPath, false)
		}
	}
	return err
}

// Remove the object.
func (o *CacheObject) Remove(ctx context.Context) error {
	wrapped, err := o.getWrapped(ctx)
	if err != nil {
		return err
	}
	err = wrapped.Remove(ctx)
	if err == nil {
		// Remove from cache
		_ = o.f.cache.Delete(o.entry.Path)

		// Invalidate all ancestor directories
		remotePath := o.Remote()
		for {
			parentPath := path.Dir(remotePath)
			if parentPath == "." || parentPath == "" || parentPath == remotePath {
				break
			}
			parentCachePath := o.f.cachePath(parentPath)
			_ = o.f.cache.SetDirLoaded(parentCachePath, false)
			remotePath = parentPath
		}
		// Also invalidate the root directory
		_ = o.f.cache.SetDirLoaded(o.f.cachePath(""), false)
	}
	return err
}

// UnWrap returns the underlying Object or nil if not available.
// This implements fs.ObjectUnWrapper interface.
func (o *CacheObject) UnWrap() fs.Object {
	return o.wrapped
}

// CacheDir represents a cached directory entry for Fs.List().
type CacheDir struct {
	entry *CacheEntry
	fs    fs.Fs
}

// NewCacheDir creates a new CacheDir from a CacheEntry.
func NewCacheDir(entry *CacheEntry, f fs.Fs) *CacheDir {
	return &CacheDir{entry: entry, fs: f}
}

// Fs returns the parent Fs.
func (d *CacheDir) Fs() fs.Info {
	return d.fs
}

// Remote returns the remote path.
func (d *CacheDir) Remote() string {
	return d.entry.Path
}

// ModTime returns the modification date of the directory.
func (d *CacheDir) ModTime(_ context.Context) time.Time {
	return d.entry.ModTime
}

// Size returns the size of the directory.
func (d *CacheDir) Size() int64 {
	return 0
}

// Items returns the count of items in this directory or this
// temporary value if unknown.
func (d *CacheDir) Items() int64 {
	return -1
}

// ID returns the internal ID of this directory if known, or "" otherwise.
func (d *CacheDir) ID() string {
	return ""
}

// String returns the name of the directory.
func (d *CacheDir) String() string {
	if d == nil {
		return "<nil>"
	}
	return d.entry.Path
}
