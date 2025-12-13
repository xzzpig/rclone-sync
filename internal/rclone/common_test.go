package rclone_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// setupTestConfig initializes rclone configuration for testing
func setupTestConfig(t *testing.T) {
	t.Helper()

	// Create a temporary directory for the test config
	tempDir := t.TempDir()
	rcloneConfPath := filepath.Join(tempDir, "rclone.conf")

	// Create an empty config file
	err := os.WriteFile(rcloneConfPath, []byte(""), 0644)
	require.NoError(t, err)

	// Initialize rclone config
	rclone.InitConfig(rcloneConfPath)
}
