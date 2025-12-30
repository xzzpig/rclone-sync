package rclone

import (
	"context"
	"fmt"
	"strings"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/filter"
	"github.com/xzzpig/rclone-sync/internal/i18n"
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
		return i18n.NewI18nError(i18n.ErrProviderNotFound).WithCause(err)
	}

	// Create a ConfigMap from the params
	m := fs.ConfigMap("", regItem.Options, "", configmap.Simple(params))

	// regItem.NewFs doesn't persist config. It creates an Fs instance from arguments.
	// This is exactly what we want for testing without saving.
	// The `name` here is a temporary name for the instance, can be empty.
	// The `root` is empty for the root of the bucket/drive.
	f, err := regItem.NewFs(ctx, "", "", m)
	if err != nil {
		return i18n.NewI18nError(i18n.ErrConnectionTestFailed).WithCause(err)
	}

	// Double check connectivity by listing the root.
	// Some backends initialize without error but fail on the first API call.
	_, err = f.List(ctx, "")
	if err != nil {
		return i18n.NewI18nError(i18n.ErrConnectionTestFailed).WithCause(err)
	}

	return nil
}

// CalculateFilterPrefix calculates the path prefix for filter matching.
// When BasePath is set and differs from currentPath, we need to calculate the relative prefix.
// This ensures filter rules written relative to BasePath match correctly.
//
// Example:
//   - BasePath="a/b/c", currentPath="a/b/c/xxx" → prefix="xxx/"
//   - BasePath="a/b/c", currentPath="a/b/c" → prefix=""
//   - BasePath="", currentPath="a/b/c" → prefix=""
//   - BasePath="a/b/c/", currentPath="a/b/c/xxx/" → prefix="xxx/"
func CalculateFilterPrefix(basePath, currentPath string) string {
	if basePath == "" {
		return ""
	}

	basePath = strings.TrimSuffix(basePath, "/")
	currentPath = strings.TrimSuffix(currentPath, "/")

	if currentPath == basePath {
		return ""
	}

	// Ensure we match complete path segments by checking for basePath + "/"
	// This prevents "a/b/c" from matching "a/b/cd/xxx"
	basePathWithSlash := basePath + "/"
	if !strings.HasPrefix(currentPath, basePathWithSlash) {
		return ""
	}

	prefix := strings.TrimPrefix(currentPath, basePathWithSlash)
	if prefix != "" {
		prefix += "/"
	}
	return prefix
}

// ListRemoteDirOptions contains options for listing remote directory entries.
type ListRemoteDirOptions struct {
	// RemoteName is the name of the configured remote (required)
	RemoteName string
	// Path is the path within the remote to list (required)
	Path string
	// BasePath is the sync task's root path (optional, used for filter matching)
	// When filters are applied, file paths are calculated relative to BasePath
	// to match the behavior during actual sync operations.
	// If empty, defaults to Path.
	BasePath string
	// Filters contains rclone filter rules to apply (optional)
	// Each rule should be in the format "- pattern" (exclude) or "+ pattern" (include)
	Filters []string
	// IncludeFiles when true includes files in the result, not just directories
	IncludeFiles bool
}

// ListRemoteDir lists directory entries in a remote or local path with filter support.
// This is used for filter preview functionality where users can see which files
// would be included/excluded by their filter rules.
//
// When RemoteName is empty, it treats Path as a local filesystem path.
// Otherwise, it uses RemoteName as the rclone remote name.
//
// Note: rclone's fs.List() does not automatically apply filters from context.
// Filters are applied at a higher level (e.g., in fs/operations or fs/walk packages).
// For directory listing, we manually apply the filter to each entry.
func ListRemoteDir(ctx context.Context, opts ListRemoteDirOptions) ([]DirEntry, error) {
	// Create filter if specified
	var fi *filter.Filter
	if len(opts.Filters) > 0 {
		var err error
		fi, err = createFilterFromRules(opts.Filters)
		if err != nil {
			return nil, err
		}
	}

	// Create the filesystem
	// When RemoteName is empty, use Path directly (local filesystem)
	// Otherwise, format as "RemoteName:Path" for remote access
	var fsPath string
	if opts.RemoteName == "" {
		fsPath = opts.Path
	} else {
		fsPath = fmt.Sprintf("%s:%s", opts.RemoteName, opts.Path)
	}
	f, err := fs.NewFs(ctx, fsPath)
	if err != nil {
		return nil, i18n.NewI18nError(i18n.ErrPathNotExist).WithCause(err)
	}

	// List entries
	entries, err := f.List(ctx, "")
	if err != nil {
		return nil, i18n.NewI18nError(i18n.ErrFailedToListRemotes).WithCause(err)
	}

	// Calculate path prefix for filter matching
	filterPrefix := ""
	if fi != nil {
		filterPrefix = CalculateFilterPrefix(opts.BasePath, opts.Path)
	}

	// Build result based on options, manually applying filter
	var result []DirEntry
	for _, entry := range entries {
		// Manually apply filter if present
		// rclone's fs.List() doesn't auto-apply context filters
		// Calculate the filter path relative to BasePath for accurate matching
		if fi != nil {
			filterPath := filterPrefix + entry.Remote()
			if !fi.IncludeRemote(filterPath) {
				continue
			}
		}

		switch e := entry.(type) {
		case fs.Directory:
			result = append(result, DirEntry{
				Name:  e.Remote(),
				Path:  fmt.Sprintf("%s/%s", opts.Path, e.Remote()),
				IsDir: true,
			})
		case fs.Object:
			// Only include files if requested
			if opts.IncludeFiles {
				result = append(result, DirEntry{
					Name:  e.Remote(),
					Path:  fmt.Sprintf("%s/%s", opts.Path, e.Remote()),
					IsDir: false,
				})
			}
		}
	}

	return result, nil
}
