package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/i18n"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// FilesHandler handles file browsing requests
type FilesHandler struct {
	connService *services.ConnectionService
}

// NewFilesHandler creates a new FilesHandler
func NewFilesHandler(connService *services.ConnectionService) *FilesHandler {
	return &FilesHandler{
		connService: connService,
	}
}

// FileEntry represents a file or directory entry
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// ListLocalFiles lists directories in a local path
// Query params:
//   - path: The directory path to list (required)
//   - blacklist: Comma-separated list of directory names to exclude (optional)
func ListLocalFiles(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrMissingParameter, "path is required"))
		return
	}

	// Check if path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrPathNotExist, err.Error()))
			return
		}
		HandleError(c, err)
		return
	}

	if !info.IsDir() {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrPathNotDirectory, ""))
		return
	}

	// Parse blacklist
	blacklistStr := c.Query("blacklist")
	blacklist := make(map[string]bool)
	if blacklistStr != "" {
		for _, item := range strings.Split(blacklistStr, ",") {
			blacklist[strings.TrimSpace(item)] = true
		}
	}

	// Read directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		HandleError(c, err)
		return
	}

	// Filter and map to FileEntry (directories only)
	var result []FileEntry
	for _, entry := range entries {
		// Skip if not a directory
		if !entry.IsDir() {
			continue
		}

		// Skip if in blacklist
		if blacklist[entry.Name()] {
			continue
		}

		result = append(result, FileEntry{
			Name:  entry.Name(),
			Path:  filepath.Join(path, entry.Name()),
			IsDir: true,
		})
	}

	c.JSON(http.StatusOK, result)
}

// ListRemoteFiles lists directories in a remote path by connection ID
// URL params:
//   - id: The connection ID (required)
//
// Query params:
//   - path: The directory path to list (required)
func (h *FilesHandler) ListRemoteFiles(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrMissingParameter, "connection id is required"))
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidIDFormat, idStr))
		return
	}

	path := c.Query("path")
	if path == "" {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrMissingParameter, "path is required"))
		return
	}

	// Get connection by ID to retrieve the remote name
	conn, err := h.connService.GetConnectionByID(c.Request.Context(), id)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusNotFound, i18n.ErrConnectionNotFound, idStr))
		return
	}

	// List remote directory using rclone
	entries, err := rclone.ListRemoteDir(c.Request.Context(), conn.Name, path)
	if err != nil {
		HandleError(c, err)
		return
	}

	// Map rclone.DirEntry to FileEntry
	result := make([]FileEntry, len(entries))
	for i, entry := range entries {
		result[i] = FileEntry{
			Name:  entry.Name,
			Path:  entry.Path,
			IsDir: entry.IsDir,
		}
	}

	c.JSON(http.StatusOK, result)
}
