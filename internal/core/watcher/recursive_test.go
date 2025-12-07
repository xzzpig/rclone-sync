package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

func TestRecursiveWatcher_RecursiveAdd(t *testing.T) {
	setupTest(t)
	mockTaskSvc := new(MockTaskService)
	mockRunner := new(MockRunner)

	// Create directory structure:
	// tempDir/root
	// tempDir/root/subdir1
	// tempDir/root/subdir1/subsubdir1
	// tempDir/root/subdir2

	rootDir := t.TempDir()

	subdir1 := filepath.Join(rootDir, "subdir1")
	subsubdir1 := filepath.Join(subdir1, "subsubdir1")
	subdir2 := filepath.Join(rootDir, "subdir2")

	assert.NoError(t, os.MkdirAll(subsubdir1, 0755))
	assert.NoError(t, os.MkdirAll(subdir2, 0755))

	task := &ent.Task{
		ID:         uuid.New(),
		Name:       "Recursive Task",
		Realtime:   true,
		SourcePath: rootDir,
	}

	mockTaskSvc.On("GetTask", mock.Anything, task.ID).Return(task, nil)
	mockTaskSvc.On("ListAllTasks", mock.Anything).Return([]*ent.Task{}, nil) // Add this line
	// Expect StartTask to be called when we touch a file deep inside
	mockRunner.On("StartTask", task, "realtime").Return(nil)

	w, err := NewWatcher(mockTaskSvc, mockRunner)
	assert.NoError(t, err)

	w.Start()
	defer w.Stop()

	// Add the task
	err = w.AddTask(task)
	assert.NoError(t, err)

	// Test: Create a file in a deep subdirectory
	time.Sleep(100 * time.Millisecond) // Wait for watchers to be added
	testFile := filepath.Join(subsubdir1, "test.txt")
	err = os.WriteFile(testFile, []byte("data"), 0644)
	assert.NoError(t, err)

	// Wait for debounce (2.2s to be safe)
	time.Sleep(2500 * time.Millisecond)

	mockRunner.AssertCalled(t, "StartTask", task, "realtime")
}

func TestRecursiveWatcher_DynamicAdd(t *testing.T) {
	setupTest(t)
	mockTaskSvc := new(MockTaskService)
	mockRunner := new(MockRunner)

	rootDir := t.TempDir()

	task := &ent.Task{
		ID:         uuid.New(),
		Name:       "Dynamic Task",
		Realtime:   true,
		SourcePath: rootDir,
	}

	mockTaskSvc.On("GetTask", mock.Anything, task.ID).Return(task, nil)
	mockTaskSvc.On("ListAllTasks", mock.Anything).Return([]*ent.Task{}, nil) // Add this line
	mockRunner.On("StartTask", task, "realtime").Return(nil)

	w, err := NewWatcher(mockTaskSvc, mockRunner)
	assert.NoError(t, err)

	w.Start()
	defer w.Stop()

	err = w.AddTask(task)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Test: Create a NEW directory
	newSubDir := filepath.Join(rootDir, "newsubdir")
	assert.NoError(t, os.Mkdir(newSubDir, 0755))

	// Wait a bit for the watcher to pick up the new dir
	time.Sleep(500 * time.Millisecond)

	// Test: Create a file in the NEW directory
	testFile := filepath.Join(newSubDir, "test.txt")
	err = os.WriteFile(testFile, []byte("data"), 0644)
	assert.NoError(t, err)

	// Wait for debounce
	time.Sleep(2500 * time.Millisecond)

	// We expect 2 calls? Maybe one for mkdir and one for writefile if debounce didn't catch both?
	// But debounce resets timer, so if events happen within 2s, only one trigger.
	// The mkdir happened, then 500ms wait, then file write. Total time < 2s.
	// Actually Wait 500ms is < 2s debounce. So timer is just reset.
	// It should trigger ONCE after the file write + 2s.
	mockRunner.AssertCalled(t, "StartTask", task, "realtime")
}

func TestRecursiveWatcher_RefCount(t *testing.T) {
	setupTest(t)
	rw, err := NewRecursiveWatcher()
	assert.NoError(t, err)
	defer rw.Close()

	tmpDir := t.TempDir()

	// 1. Add first time
	err = rw.Add(tmpDir)
	assert.NoError(t, err)

	// Verify internal state (allowed since we are in package watcher)
	rw.mu.Lock()
	count, ok := rw.watchedDirs[tmpDir]
	rw.mu.Unlock()
	assert.True(t, ok)
	assert.Equal(t, 1, count, "Count should be 1 after first Add")

	// 2. Add second time
	err = rw.Add(tmpDir)
	assert.NoError(t, err)

	rw.mu.Lock()
	count, ok = rw.watchedDirs[tmpDir]
	rw.mu.Unlock()
	assert.True(t, ok)
	assert.Equal(t, 2, count, "Count should be 2 after second Add")

	// 3. Remove first time
	err = rw.Remove(tmpDir)
	assert.NoError(t, err)

	rw.mu.Lock()
	count, ok = rw.watchedDirs[tmpDir]
	rw.mu.Unlock()
	assert.True(t, ok)
	assert.Equal(t, 1, count, "Count should be 1 after first Remove")

	// Verify watcher is still active: Create a file and expect event
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("data"), 0644)
	assert.NoError(t, err)

	select {
	case event := <-rw.Events():
		assert.Equal(t, testFile, event.Name)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for event, watcher might have been removed prematurely")
	}

	// 4. Remove second time
	err = rw.Remove(tmpDir)
	assert.NoError(t, err)

	rw.mu.Lock()
	_, ok = rw.watchedDirs[tmpDir]
	rw.mu.Unlock()
	assert.False(t, ok, "Entry should be removed from map after count reaches 0")
}
