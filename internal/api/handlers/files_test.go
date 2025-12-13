package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListLocalFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create test directories
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "dir1"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "dir2"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "dir1", "subdir"), 0755))

	// Create test files (should be ignored)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dir1", "file.txt"), []byte("test"), 0644))

	tests := []struct {
		name           string
		path           string
		expectedDirs   []string
		expectedStatus int
	}{
		{
			name:           "list root directories",
			path:           tmpDir,
			expectedDirs:   []string{"dir1", "dir2"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "list subdirectories",
			path:           filepath.Join(tmpDir, "dir1"),
			expectedDirs:   []string{"subdir"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid path",
			path:           "/nonexistent/path",
			expectedDirs:   nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/files/local", ListLocalFiles)

			req := httptest.NewRequest(http.MethodGet, "/files/local?path="+tt.path, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedStatus == http.StatusOK {
				var result []FileEntry
				require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &result))

				// Check that we got the expected directories
				assert.Len(t, result, len(tt.expectedDirs))

				for i, dir := range tt.expectedDirs {
					assert.Equal(t, dir, result[i].Name)
					assert.True(t, result[i].IsDir)
					assert.NotEmpty(t, result[i].Path)
				}
			}
		})
	}
}

func TestListLocalFilesWithBlacklist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()

	// Create test directories
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "node_modules"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".git"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "src"), 0755))

	router := gin.New()
	router.GET("/files/local", ListLocalFiles)

	req := httptest.NewRequest(http.MethodGet, "/files/local?path="+tmpDir+"&blacklist=node_modules,.git", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var result []FileEntry
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &result))

	// Should only contain "src", not "node_modules" or ".git"
	assert.Len(t, result, 1)
	assert.Equal(t, "src", result[0].Name)
}

func TestListRemoteFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		remoteName     string
		path           string
		expectedStatus int
	}{
		{
			name:           "missing remote name",
			remoteName:     "",
			path:           "/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "missing path",
			remoteName:     "test-remote",
			path:           "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/files/remote/:name", ListRemoteFiles)

			url := "/files/remote/" + tt.remoteName
			if tt.path != "" {
				url += "?path=" + tt.path
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
		})
	}
}
