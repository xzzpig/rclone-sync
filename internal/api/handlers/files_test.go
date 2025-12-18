package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"

	_ "github.com/mattn/go-sqlite3"
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
			path:           filepath.Join(tmpDir, "nonexistent"),
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

	// Setup test database and services
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	encryptor, err := crypto.NewEncryptor("test-key-12345678901234567890123")
	require.NoError(t, err)

	connService := services.NewConnectionService(client, encryptor)
	filesHandler := NewFilesHandler(connService)

	// Create a test connection
	conn, err := connService.CreateConnection(t.Context(), "test_remote", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	storage := rclone.NewDBStorage(connService)
	storage.Install()

	tmpDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "test_folder"), 0755))

	tests := []struct {
		name           string
		connectionID   string
		path           string
		expectedStatus int
	}{
		{
			name:           "successful listing",
			connectionID:   conn.ID.String(),
			path:           tmpDir,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing connection id",
			connectionID:   "",
			path:           "/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid connection id format",
			connectionID:   "invalid-uuid",
			path:           "/",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "connection not found",
			connectionID:   uuid.New().String(),
			path:           "/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "missing path",
			connectionID:   conn.ID.String(),
			path:           "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/files/remote/:id", filesHandler.ListRemoteFiles)

			url := "/files/remote/" + tt.connectionID
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
