package rclone

import (
	"context"
	"fmt"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
)

// DirEntry represents a directory entry from rclone
type DirEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// TestRemote verifies if the remote configuration is valid by attempting to create the Fs
// and doing a lightweight check.
func TestRemote(ctx context.Context, providerName string, params map[string]string) error {
	regItem, err := fs.Find(providerName)
	if err != nil {
		return fmt.Errorf("provider %q not found: %w", providerName, err)
	}

	// Create a ConfigMap from the params
	m := fs.ConfigMap("", regItem.Options, "", configmap.Simple(params))

	// regItem.NewFs doesn't persist config. It creates an Fs instance from arguments.
	// This is exactly what we want for testing without saving.
	// The `name` here is a temporary name for the instance, can be empty.
	// The `root` is empty for the root of the bucket/drive.
	f, err := regItem.NewFs(ctx, "", "", m)
	if err != nil {
		return fmt.Errorf("failed to initialize backend: %w", err)
	}

	// Double check connectivity by listing the root.
	// Some backends initialize without error but fail on the first API call.
	_, err = f.List(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to list root of remote: %w", err)
	}

	return nil
}

// ListRemoteDir lists directories in a remote path
func ListRemoteDir(ctx context.Context, remoteName string, path string) ([]DirEntry, error) {
	// Create the remote filesystem
	remotePath := fmt.Sprintf("%s:%s", remoteName, path)
	f, err := fs.NewFs(ctx, remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem for %s: %w", remotePath, err)
	}

	// List entries
	entries, err := f.List(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	// Filter directories only
	var result []DirEntry
	for _, entry := range entries {
		if dir, ok := entry.(fs.Directory); ok {
			result = append(result, DirEntry{
				Name:  dir.Remote(),
				Path:  fmt.Sprintf("%s/%s", path, dir.Remote()),
				IsDir: true,
			})
		}
	}

	return result, nil
}
