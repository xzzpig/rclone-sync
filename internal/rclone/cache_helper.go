// Package rclone provides rclone integration functionality.
// This file provides helper functions for working with rclone's internal cache.
package rclone

import (
	"context"

	"github.com/rclone/rclone/fs/cache"
)

// IsConnectionLoaded checks if a connection (remote) has been loaded into rclone's cache.
// This is useful for determining if a remote is currently active and usable.
//
// When a remote is first accessed (via fs.NewFs), it gets loaded into the cache.
// This function checks if that has happened for the given connection name.
func IsConnectionLoaded(name string) bool {
	if name == "" {
		return false
	}

	// Check if the remote is in rclone's cache
	// cache.Get returns a cached Fs if it exists
	// We use the remote path format "name:" to check
	_, err := cache.Get(context.Background(), name+":")
	return err == nil
}
