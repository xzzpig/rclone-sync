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

// CalculateListPath calculates the Fs root path and the relative list path for directory listing.
// When basePath is set and currentPath is under basePath, it returns:
//   - fsRootPath: basePath (for Fs caching)
//   - listPath: the relative path from basePath to currentPath
//
// When currentPath is not under basePath, it returns:
//   - fsRootPath: currentPath
//   - listPath: ""
//
// Examples:
//   - basePath="a/b", currentPath="a/b/c/d" → fsRootPath="a/b", listPath="c/d"
//   - basePath="a/b", currentPath="a/b" → fsRootPath="a/b", listPath=""
//   - basePath="", currentPath="x/y" → fsRootPath="x/y", listPath=""
//   - basePath="a/b", currentPath="x/y" → fsRootPath="x/y", listPath=""
func CalculateListPath(basePath, currentPath string) (fsRootPath, listPath string) {
	// Default: use currentPath as Fs root
	fsRootPath = currentPath
	listPath = ""

	// If basePath is empty, just use currentPath
	if basePath == "" {
		return fsRootPath, listPath
	}

	// Normalize paths by removing trailing slashes
	basePath = strings.TrimSuffix(basePath, "/")
	currentPath = strings.TrimSuffix(currentPath, "/")

	// If paths are equal, use basePath as root with empty listPath
	if currentPath == basePath {
		return basePath, ""
	}

	// Check if currentPath is under basePath
	if strings.HasPrefix(currentPath, basePath+"/") {
		fsRootPath = basePath
		listPath = strings.TrimPrefix(currentPath, basePath+"/")
		return fsRootPath, listPath
	}

	// currentPath is not under basePath, use currentPath as root
	return currentPath, ""
}

// ExtractEntryName extracts the last path segment from a path.
// This is used to get the display name for directory entries.
//
// Examples:
//   - "subdir/file.txt" → "file.txt"
//   - "file.txt" → "file.txt"
//   - "a/b/c" → "c"
//   - "" → ""
func ExtractEntryName(path string) string {
	if lastSlash := strings.LastIndex(path, "/"); lastSlash >= 0 {
		return path[lastSlash+1:]
	}
	return path
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
// Caching strategy: When BasePath is set, the Fs is cached using remote:BasePath as the key.
// This allows browsing different subdirectories within the same task to reuse the cached Fs.
// When BasePath is empty, opts.Path is used as the Fs root (no caching benefit for subdirs).
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

	// Calculate the Fs root path and relative list path
	// When BasePath is set, use it as the Fs root to enable cache reuse across subdirectories
	fsRootPath, listPath := CalculateListPath(opts.BasePath, opts.Path)

	// Create the filesystem using GetFs for cache support
	// When RemoteName is empty, path is treated as local filesystem (no caching)
	// When RemoteName is non-empty, cache.Get is used to reuse existing Fs instances
	f, err := GetFs(ctx, opts.RemoteName, fsRootPath)
	if err != nil {
		return nil, i18n.NewI18nError(i18n.ErrPathNotExist).WithCause(err)
	}

	// List entries at the relative path
	entries, err := f.List(ctx, listPath)
	if err != nil {
		return nil, i18n.NewI18nError(i18n.ErrFailedToListRemotes).WithCause(err)
	}

	// Build result based on options, manually applying filter
	var result []DirEntry
	for _, entry := range entries {
		// entry.Remote() returns the path relative to Fs root (BasePath)
		// e.g., when listing "subdir" in Fs rooted at BasePath, entry.Remote() returns "subdir/file.txt"
		entryRemote := entry.Remote()

		// Get the entry name (last path segment) for display
		entryName := ExtractEntryName(entryRemote)

		// Manually apply filter if present
		// rclone's fs.List() doesn't auto-apply context filters
		// entry.Remote() is already relative to BasePath (Fs root), so use it directly for filter matching
		if fi != nil {
			if !fi.IncludeRemote(entryRemote) {
				continue
			}
		}

		switch entry.(type) {
		case fs.Directory:
			result = append(result, DirEntry{
				Name:  entryName,
				Path:  fmt.Sprintf("%s/%s", opts.Path, entryName),
				IsDir: true,
			})
		case fs.Object:
			// Only include files if requested
			if opts.IncludeFiles {
				result = append(result, DirEntry{
					Name:  entryName,
					Path:  fmt.Sprintf("%s/%s", opts.Path, entryName),
					IsDir: false,
				})
			}
		}
	}

	return result, nil
}
