// Package rclone provides rclone integration functionality.
// This file provides helper functions for working with rclone's internal cache.
package rclone

import (
	"context"
	"errors"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/cache"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"go.uber.org/zap"
)

// GetFs returns a cached Fs for remote paths, or creates a new Fs for direct local paths.
//
// Parameters:
//   - ctx: context for the operation
//   - remote: the remote name (e.g., "myremote") or empty string for local paths
//   - path: the path within the remote or local filesystem
//
// When remote is empty, path is treated as a direct local filesystem path and
// fs.NewFs is used without caching (per FR-009 specification).
// When remote is non-empty, cache.GetFn is used to reuse existing Fs instances,
// with logging when a new Fs is created.
//
// Examples:
//
//	GetFs(ctx, "", "/home/user/data")           // local path, no caching
//	GetFs(ctx, "myremote", "path/to/folder")    // remote path, uses cache
//	GetFs(ctx, "myremote", "")                  // remote root, uses cache
func GetFs(ctx context.Context, remote string, path string) (fs.Fs, error) {
	if remote == "" {
		// Direct local path - no caching
		return fs.NewFs(ctx, path)
	}
	// Remote path - use cache with logging
	// Build fsPath in format "remote:path" or "remote:" for root
	fsPath := remote + ":" + path

	newCtx := context.WithoutCancel(ctx)
	return cache.GetFn(newCtx, fsPath, func(ctx context.Context, fsPath string) (fs.Fs, error) {
		logger.Named("rclone.cache").Info("Creating new Fs", zap.String("fsPath", fsPath))
		return fs.NewFs(ctx, fsPath)
	})
}

// ClearFsCache clears the Fs cache for the given remote name.
// This should be called when a remote configuration is updated or deleted.
//
// The remoteName should be just the name, without the colon (e.g., "myremote" not "myremote:").
// If remoteName is empty, this function does nothing.
//
// Returns the number of cache entries that were deleted.
func ClearFsCache(remoteName string) int {
	if remoteName == "" {
		return 0
	}
	return cache.ClearConfig(remoteName)
}

// errNotLoaded is a sentinel error used by IsConnectionLoaded to prevent
// cache.GetFn from creating a new Fs when checking if a remote is already loaded.
var errNotLoaded = errors.New("not loaded")

// IsConnectionLoaded checks if a connection (remote) with a specific path has been loaded into rclone's cache.
// This is useful for determining if a remote path is currently cached and usable.
//
// When a remote is first accessed (via cache.Get), it gets loaded into the cache with the full "remote:path" key.
// This function checks if that has happened for the given remote name and path combination.
//
// Parameters:
//   - remoteName: the remote name (e.g., "myremote") or empty string for local paths
//   - path: the path within the remote or local filesystem
//
// Unlike cache.Get, this function does NOT create a new Fs if it doesn't exist in the cache.
func IsConnectionLoaded(remoteName string, path string) bool {
	if remoteName == "" {
		// Local paths are not cached, so they are never "loaded" in the cache sense
		return false
	}

	// Build fsPath in format "remote:path" or "remote:" for root, same as GetFs
	fsPath := remoteName + ":" + path

	// Use cache.GetFn with a create function that returns an error
	// If the Fs is already in cache, GetFn returns it without calling the create function
	// If the Fs is not in cache, GetFn calls the create function which returns errNotLoaded
	_, err := cache.GetFn(context.Background(), fsPath, func(_ context.Context, _ string) (fs.Fs, error) {
		// Return an error to prevent creating a new Fs
		return nil, errNotLoaded
	})

	// If err is nil, the Fs was already in the cache
	// If err == errNotLoaded (or any error), the Fs was not in the cache
	return err == nil
}
