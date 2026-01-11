// Test metacache filesystem interface using rclone's fstests framework
//
//go:build !plan9 && !js && !race

package metacache_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/rclone/rclone/backend/local"
	"github.com/rclone/rclone/fstest"
	"github.com/rclone/rclone/fstest/fstests"
	"github.com/xzzpig/rclone-sync/internal/rclone/backend/metacache"
	_ "github.com/xzzpig/rclone-sync/internal/rclone/backend/notifylocal"
)

// TestIntegration runs integration tests against the metacache remote
// wrapping a local filesystem using rclone's fstests framework.
func TestIntegration(t *testing.T) {
	// Create temporary directories for test
	tempDir := t.TempDir()
	localDir := filepath.Join(tempDir, "local")
	cacheDB := filepath.Join(tempDir, "cache.db")
	configFile := filepath.Join(tempDir, "rclone.conf")

	if err := os.MkdirAll(localDir, 0750); err != nil {
		t.Fatalf("Failed to create local dir: %v", err)
	}

	// Create rclone config file with TestLocal and TestMetaCache remotes
	// Use a very short info_age (1s) to ensure cache is always refreshed
	// This avoids cache consistency issues during testing
	configContent := fmt.Sprintf(`[TestLocal]
type = notifylocal

[TestMetaCache]
type = metacache
remote = TestLocal:%s
db_path = %s
connection_id = test-fstests
info_age = 1h
`, localDir, cacheDB)

	if err := os.WriteFile(configFile, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Set RCLONE_CONFIG environment variable before Initialise
	oldConfig := os.Getenv("RCLONE_CONFIG")
	os.Setenv("RCLONE_CONFIG", configFile)
	defer func() {
		if oldConfig != "" {
			os.Setenv("RCLONE_CONFIG", oldConfig)
		} else {
			os.Unsetenv("RCLONE_CONFIG")
		}
	}()

	// Initialize rclone test environment
	fstest.Initialise()

	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestMetaCache:",
		NilObject:  (*metacache.CacheObject)(nil),
		// Methods not implemented by metacache (delegated to wrapped Fs)
		// These are not wrapped because metacache is a pure metadata cache
		// that doesn't need to intercept these operations
		UnimplementableFsMethods: []string{
			"PublicLink",
			"OpenWriterAt",
			"OpenChunkWriter",
			"DirSetModTime",
			"MkdirMetadata",
			"ListP",
			// Fs wrapper methods - metacache delegates these without caching
			"PutUnchecked",
			"PutStream",
			"MergeDirs",
			"CleanUp",
			"ListR",
			"About",
			"UserInfo",
			"Disconnect",
		},
		UnimplementableObjectMethods: []string{
			"MimeType",
			"ID",
			"GetTier",
			"SetTier",
			"Metadata",
			"SetMetadata",
		},
		UnimplementableDirectoryMethods: []string{
			"Metadata",
			"SetMetadata",
			"SetModTime",
		},
		SkipInvalidUTF8: true, // invalid UTF-8 may confuse the cache
	})
}
