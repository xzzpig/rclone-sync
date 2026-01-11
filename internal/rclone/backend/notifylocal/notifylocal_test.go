//go:build !plan9 && !js && !race

package notifylocal_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fstest"
	"github.com/rclone/rclone/fstest/fstests"
	_ "github.com/xzzpig/rclone-sync/internal/rclone/backend/notifylocal"
)

func TestIntegration(t *testing.T) {
	tempDir := t.TempDir()
	localDir := filepath.Join(tempDir, "local")

	if err := os.MkdirAll(localDir, 0750); err != nil {
		t.Fatalf("Failed to create local dir: %v", err)
	}

	configFile := filepath.Join(tempDir, "rclone.conf")
	configContent := `[TestNotifyLocal]
type = notifylocal
`

	if err := os.WriteFile(configFile, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	oldConfig := os.Getenv("RCLONE_CONFIG")
	os.Setenv("RCLONE_CONFIG", configFile)
	defer func() {
		if oldConfig != "" {
			os.Setenv("RCLONE_CONFIG", oldConfig)
		} else {
			os.Unsetenv("RCLONE_CONFIG")
		}
	}()

	fstest.Initialise()

	fstests.Run(t, &fstests.Opt{
		RemoteName:          "TestNotifyLocal:" + localDir,
		NilObject:           nil,
		SkipFsCheckWrap:     true,
		SkipObjectCheckWrap: true,
		SkipInvalidUTF8:     true,
	})
}

func TestChangeNotify(t *testing.T) {
	tempDir := t.TempDir()

	configFile := filepath.Join(tempDir, "rclone.conf")
	configContent := `[TestNotifyLocal]
type = notifylocal
`
	if err := os.WriteFile(configFile, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	oldConfig := os.Getenv("RCLONE_CONFIG")
	os.Setenv("RCLONE_CONFIG", configFile)
	defer func() {
		if oldConfig != "" {
			os.Setenv("RCLONE_CONFIG", oldConfig)
		} else {
			os.Unsetenv("RCLONE_CONFIG")
		}
	}()

	fstest.Initialise()

	testDir := filepath.Join(tempDir, "testdata")
	if err := os.MkdirAll(testDir, 0750); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	testFs, err := fs.NewFs(context.Background(), "TestNotifyLocal:"+testDir)
	if err != nil {
		t.Fatalf("Failed to create notifylocal Fs: %v", err)
	}

	doChangeNotify := testFs.Features().ChangeNotify
	if doChangeNotify == nil {
		t.Fatal("notifylocal should support ChangeNotify")
	}

	var mu sync.Mutex
	notifications := make([]struct {
		path      string
		entryType fs.EntryType
	}, 0)

	pollChan := make(chan time.Duration)
	doChangeNotify(context.Background(), func(path string, entryType fs.EntryType) {
		mu.Lock()
		notifications = append(notifications, struct {
			path      string
			entryType fs.EntryType
		}{path, entryType})
		mu.Unlock()
	}, pollChan)

	time.Sleep(100 * time.Millisecond)

	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	notificationCount := len(notifications)
	mu.Unlock()

	if notificationCount == 0 {
		t.Error("Expected at least one ChangeNotify callback after file creation")
	}

	var foundTestFile bool
	mu.Lock()
	for _, n := range notifications {
		if n.path == "test.txt" {
			foundTestFile = true
			break
		}
	}
	mu.Unlock()

	if !foundTestFile {
		mu.Lock()
		t.Errorf("Expected notification for 'test.txt', got: %v", notifications)
		mu.Unlock()
	}

	close(pollChan)

	if shutdowner, ok := testFs.(fs.Shutdowner); ok {
		if err := shutdowner.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}
}

func TestChangeNotifyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	configFile := filepath.Join(tempDir, "rclone.conf")
	configContent := `[TestNotifyLocal]
type = notifylocal
`
	if err := os.WriteFile(configFile, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	oldConfig := os.Getenv("RCLONE_CONFIG")
	os.Setenv("RCLONE_CONFIG", configFile)
	defer func() {
		if oldConfig != "" {
			os.Setenv("RCLONE_CONFIG", oldConfig)
		} else {
			os.Unsetenv("RCLONE_CONFIG")
		}
	}()

	fstest.Initialise()

	testDir := filepath.Join(tempDir, "testdata")
	if err := os.MkdirAll(testDir, 0750); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	testFs, err := fs.NewFs(context.Background(), "TestNotifyLocal:"+testDir)
	if err != nil {
		t.Fatalf("Failed to create notifylocal Fs: %v", err)
	}

	doChangeNotify := testFs.Features().ChangeNotify
	if doChangeNotify == nil {
		t.Fatal("notifylocal should support ChangeNotify")
	}

	var mu sync.Mutex
	notifications := make([]struct {
		path      string
		entryType fs.EntryType
	}, 0)

	pollChan := make(chan time.Duration)
	doChangeNotify(context.Background(), func(path string, entryType fs.EntryType) {
		mu.Lock()
		notifications = append(notifications, struct {
			path      string
			entryType fs.EntryType
		}{path, entryType})
		mu.Unlock()
	}, pollChan)

	time.Sleep(100 * time.Millisecond)

	subDir := filepath.Join(testDir, "subdir")
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	notificationCount := len(notifications)
	mu.Unlock()

	if notificationCount == 0 {
		t.Error("Expected at least one ChangeNotify callback after directory creation")
	}

	var foundSubDir bool
	var entryType fs.EntryType
	mu.Lock()
	for _, n := range notifications {
		if n.path == "subdir" {
			foundSubDir = true
			entryType = n.entryType
			break
		}
	}
	mu.Unlock()

	if !foundSubDir {
		mu.Lock()
		t.Errorf("Expected notification for 'subdir', got: %v", notifications)
		mu.Unlock()
	} else if entryType != fs.EntryDirectory {
		t.Errorf("Expected EntryDirectory for 'subdir', got: %v", entryType)
	}

	close(pollChan)

	if shutdowner, ok := testFs.(fs.Shutdowner); ok {
		if err := shutdowner.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}
}
