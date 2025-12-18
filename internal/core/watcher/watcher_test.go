package watcher

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"go.uber.org/zap"
)

// MockRunner is a mock for the Runner interface
type MockRunner struct {
	mock.Mock
}

func (m *MockRunner) Start() { m.Called() }
func (m *MockRunner) Stop()  { m.Called() }
func (m *MockRunner) StartTask(task *ent.Task, trigger job.Trigger) error {
	args := m.Called(task, string(trigger))
	return args.Error(0)
}
func (m *MockRunner) StopTask(taskID uuid.UUID) error {
	args := m.Called(taskID)
	return args.Error(0)
}
func (m *MockRunner) IsRunning(taskID uuid.UUID) bool {
	args := m.Called(taskID)
	return args.Bool(0)
}

// MockTaskService is a mock for the TaskService interface
type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) GetTask(ctx context.Context, id uuid.UUID) (*ent.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Task), args.Error(1)
}

func (m *MockTaskService) GetTaskWithConnection(ctx context.Context, id uuid.UUID) (*ent.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Task), args.Error(1)
}

func (m *MockTaskService) ListAllTasks(ctx context.Context) ([]*ent.Task, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ent.Task), args.Error(1)
}

func setupTest(t *testing.T) {
	if logger.L == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			t.Fatalf("failed to create logger: %v", err)
		}
		logger.L = l
	}
	_ = t // Suppress unused parameter warning
}

func TestWatcher_Debounce(t *testing.T) {
	setupTest(t)
	mockTaskSvc := new(MockTaskService)
	mockRunner := new(MockRunner)

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	task := &ent.Task{
		ID:         uuid.New(),
		Name:       "Realtime Task",
		Realtime:   true,
		SourcePath: tempDir,
	}

	// Watcher will call GetTaskWithConnection when the debounce timer fires.
	mockTaskSvc.On("GetTaskWithConnection", mock.Anything, task.ID).Return(task, nil)
	// We expect StartTask to be called only once.
	mockRunner.On("StartTask", task, "realtime").Return(nil).Once()

	w, err := NewWatcher(mockTaskSvc, mockRunner)
	assert.NoError(t, err)

	// Add the task to the watcher
	err = w.AddTask(task)
	assert.NoError(t, err)

	// Simulate rapid-fire events
	for i := 0; i < 5; i++ {
		w.handleEvent(fsnotify.Event{Name: filepath.Join(tempDir, "file.txt"), Op: fsnotify.Write})
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for the debounce timer (2s in code) + a buffer
	time.Sleep(2500 * time.Millisecond)

	mockTaskSvc.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestWatcher_PathFiltering(t *testing.T) {
	setupTest(t)
	mockTaskSvc := new(MockTaskService)
	mockRunner := new(MockRunner)

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	task := &ent.Task{
		ID:         uuid.New(),
		Name:       "Realtime Task",
		Realtime:   true,
		SourcePath: tempDir,
	}

	w, err := NewWatcher(mockTaskSvc, mockRunner)
	assert.NoError(t, err)

	err = w.AddTask(task)
	assert.NoError(t, err)

	// This event should NOT trigger a sync because the path doesn't match.
	w.handleEvent(fsnotify.Event{Name: filepath.Join(tempDir, "..", "other", "file.txt"), Op: fsnotify.Write})

	time.Sleep(2500 * time.Millisecond)

	// Assert that StartTask was never called.
	mockRunner.AssertNotCalled(t, "StartTask", mock.Anything, mock.Anything)
}

func TestWatcher_StartStopBehavior(t *testing.T) {
	setupTest(t)
	mockTaskSvc := new(MockTaskService)
	mockRunner := new(MockRunner)

	// Expect ListAllTasks to be called only once across all Start() calls.
	mockTaskSvc.On("ListAllTasks", mock.Anything).Return([]*ent.Task{}, nil).Once()

	w, err := NewWatcher(mockTaskSvc, mockRunner)
	assert.NoError(t, err)

	// 1. Test Start idempotency
	w.Start()
	w.Start() // Second call should be a no-op

	// 2. Test Stop idempotency
	w.Stop()
	w.Stop() // Second call should be a no-op

	// 3. Test that it cannot be restarted
	w.Start() // This call should be a no-op

	// Verify that ListAllTasks was only ever called once.
	mockTaskSvc.AssertExpectations(t)
}

// MockFileWatcher is a mock for the FileWatcher interface
type MockFileWatcher struct {
	mock.Mock
	EventsCh chan fsnotify.Event
	ErrorsCh chan error
}

func NewMockFileWatcher() *MockFileWatcher {
	return &MockFileWatcher{
		EventsCh: make(chan fsnotify.Event),
		ErrorsCh: make(chan error),
	}
}

func (m *MockFileWatcher) Add(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockFileWatcher) Remove(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockFileWatcher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockFileWatcher) Events() chan fsnotify.Event {
	return m.EventsCh
}

func (m *MockFileWatcher) Errors() chan error {
	return m.ErrorsCh
}

func TestWatcher_RemoveTask(t *testing.T) {
	setupTest(t)
	mockTaskSvc := new(MockTaskService)
	mockRunner := new(MockRunner)
	mockFW := NewMockFileWatcher()

	task := &ent.Task{
		ID:         uuid.New(),
		Name:       "Task1",
		Realtime:   true,
		SourcePath: "/tmp/test",
	}

	w := newWatcher(mockTaskSvc, mockRunner, mockFW)

	// Pre-fill watchMap to simulate task being watched
	w.watchMap[task.ID.String()] = task.SourcePath

	// Expect Remove to be called
	mockFW.On("Remove", task.SourcePath).Return(nil).Once()

	err := w.RemoveTask(task)
	assert.NoError(t, err)

	// Verify path was removed from watchMap
	assert.NotContains(t, w.watchMap, task.ID.String())

	mockFW.AssertExpectations(t)
}

func TestWatcher_Start_LoadWatchTasks(t *testing.T) {
	setupTest(t)
	mockTaskSvc := new(MockTaskService)
	mockRunner := new(MockRunner)
	mockFW := NewMockFileWatcher()

	task1 := &ent.Task{
		ID:         uuid.New(),
		Name:       "Realtime Task",
		Realtime:   true,
		SourcePath: "/tmp/task1",
	}
	task2 := &ent.Task{
		ID:         uuid.New(),
		Name:       "Manual Task",
		Realtime:   false,
		SourcePath: "/tmp/task2",
	}

	// Mock ListAllTasks to return one realtime and one manual task
	mockTaskSvc.On("ListAllTasks", mock.Anything).Return([]*ent.Task{task1, task2}, nil)

	// Expect Add to be called ONLY for the realtime task
	mockFW.On("Add", task1.SourcePath).Return(nil).Once()

	// Expect Close to be called when stopping
	mockFW.On("Close").Return(nil).Once()

	w := newWatcher(mockTaskSvc, mockRunner, mockFW)

	w.Start()
	// Stop immediately to prevent blocking loop (though loop runs in goroutine)
	w.Stop()

	// Verify only realtime task was added to watchMap
	assert.Contains(t, w.watchMap, task1.ID.String())
	assert.NotContains(t, w.watchMap, task2.ID.String())

	mockTaskSvc.AssertExpectations(t)
	mockFW.AssertExpectations(t)
}
