package rclone

import (
	"context"
	"fmt"

	"github.com/rclone/rclone/fs"
)

// AboutInfo represents quota information for a remote storage.
type AboutInfo = fs.Usage

// GetRemoteQuota gets the quota information for a remote
// It corresponds to the `rclone about` command
func GetRemoteQuota(ctx context.Context, remoteName string) (*AboutInfo, error) {
	// Create the Fs for the remote using cached Fs via GetFs
	// GetFs with remote name and empty path returns cached "remote:" Fs
	f, err := GetFs(ctx, remoteName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create fs for remote %s: %w", remoteName, err)
	}

	// Check if the Fs implements the Abouter interface
	abouter, ok := f.(fs.Abouter)
	if !ok {
		return nil, fmt.Errorf("remote %s does not support quota information (About)", remoteName) //nolint:err113
	}

	// Call the About method
	usage, err := abouter.About(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get quota information for remote %s: %w", remoteName, err)
	}

	// Convert fs.Usage to AboutInfo
	return &AboutInfo{
		Total:   usage.Total,
		Used:    usage.Used,
		Trashed: usage.Trashed,
		Other:   usage.Other,
		Free:    usage.Free,
		Objects: usage.Objects,
	}, nil
}
