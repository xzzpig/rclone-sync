package runner_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/runner"
)

// MockSyncEngine is a mock implementation of the SyncEngine interface.
type MockSyncEngine struct {
	mock.Mock
}

// RunTask mocks the RunTask method.
func (m *MockSyncEngine) RunTask(ctx context.Context, task *ent.Task, trigger model.JobTrigger) error {
	args := m.Called(ctx, task, trigger)
	// Simulate work or a blocking call that respects context cancellation.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(200 * time.Millisecond): // Simulate some work
		return args.Error(0)
	}
}

func setupTest() {
	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)
}

func TestRunner_StartAndStopTask(t *testing.T) {
	setupTest()
	mockEngine := new(MockSyncEngine)
	r := runner.NewRunner(mockEngine)

	task := &ent.Task{ID: uuid.New()}
	trigger := model.JobTriggerManual

	// We use .On() to set an expectation that RunTask will be called.
	// We use .Run() to capture the context passed to the mock, so we can check if it gets canceled.
	var taskCtx context.Context
	startedChan := make(chan struct{})
	mockEngine.On("RunTask", mock.Anything, task, trigger).Return(nil).Run(func(args mock.Arguments) {
		if ctx, ok := args.Get(0).(context.Context); ok {
			taskCtx = ctx
		}
		// Notify the test that the goroutine has started
		close(startedChan)
	})

	// Start the task
	err := r.StartTask(task, trigger)
	assert.NoError(t, err)
	assert.True(t, r.IsRunning(task.ID))

	// Wait for the goroutine to start
	select {
	case <-startedChan:
		// Continue with the test
	case <-time.After(1 * time.Second):
		t.Fatal("goroutine failed to start within timeout")
	}

	// Stop the task
	err = r.StopTask(task.ID)
	assert.NoError(t, err)
	assert.False(t, r.IsRunning(task.ID))

	// Verify that the context was actually canceled by StopTask
	if taskCtx == nil {
		t.Fatal("taskCtx should not be nil")
	}
	select {
	case <-taskCtx.Done():
		// This is the expected outcome.
	case <-time.After(100 * time.Millisecond):
		t.Fatal("context should have been canceled after StopTask was called")
	}

	mockEngine.AssertExpectations(t)
}

func TestRunner_Concurrency_StartTaskTwiceCancelsFirst(t *testing.T) {
	setupTest()
	mockEngine := new(MockSyncEngine)
	r := runner.NewRunner(mockEngine)

	task := &ent.Task{ID: uuid.New()}
	trigger1 := model.JobTriggerManual
	trigger2 := model.JobTriggerRealtime

	var firstTaskCtx, secondTaskCtx context.Context
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})

	// Expectation for the first call
	mockEngine.On("RunTask", mock.Anything, task, trigger1).Return(nil).Run(func(args mock.Arguments) {
		firstTaskCtx = args.Get(0).(context.Context)
		close(firstStarted)
	}).Once()

	// Expectation for the second call
	mockEngine.On("RunTask", mock.Anything, task, trigger2).Return(nil).Run(func(args mock.Arguments) {
		secondTaskCtx = args.Get(0).(context.Context)
		close(secondStarted)
	}).Once()

	// Start the task for the first time
	err := r.StartTask(task, trigger1)
	assert.NoError(t, err)
	assert.True(t, r.IsRunning(task.ID))
	// Wait for the first goroutine to start
	select {
	case <-firstStarted:
		// Continue with the test
	case <-time.After(1 * time.Second):
		t.Fatal("first goroutine failed to start within timeout")
	}

	// Start the task for the second time, which should cancel the first run
	err = r.StartTask(task, trigger2)
	assert.NoError(t, err)
	assert.True(t, r.IsRunning(task.ID))
	// Wait for the second goroutine to start
	select {
	case <-secondStarted:
		// Continue with the test
	case <-time.After(1 * time.Second):
		t.Fatal("second goroutine failed to start within timeout")
	}

	// Check that the first task's context is now canceled
	select {
	case <-firstTaskCtx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("the first task's context should have been canceled by the second StartTask call")
	}

	// Check that the second task's context is still active
	select {
	case <-secondTaskCtx.Done():
		t.Fatal("the second task's context should not be canceled yet")
	default:
		// Expected
	}

	// Stop the second (and currently running) task
	err = r.StopTask(task.ID)
	assert.NoError(t, err)
	assert.False(t, r.IsRunning(task.ID))

	// Check that the second task's context is now also canceled
	select {
	case <-secondTaskCtx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("the second task's context should have been canceled after StopTask was called")
	}

	mockEngine.AssertExpectations(t)
}

func TestRunner_Stop_CancelsAllTasks(t *testing.T) {
	setupTest()
	mockEngine := new(MockSyncEngine)
	r := runner.NewRunner(mockEngine)

	// Create multiple tasks
	task1 := &ent.Task{ID: uuid.New()}
	task2 := &ent.Task{ID: uuid.New()}
	trigger := model.JobTriggerManual

	var ctx1, ctx2 context.Context
	task1Started := make(chan struct{})
	task2Started := make(chan struct{})

	// Setup expectations
	mockEngine.On("RunTask", mock.Anything, task1, trigger).Return(nil).Run(func(args mock.Arguments) {
		ctx1 = args.Get(0).(context.Context)
		close(task1Started)
	})
	mockEngine.On("RunTask", mock.Anything, task2, trigger).Return(nil).Run(func(args mock.Arguments) {
		ctx2 = args.Get(0).(context.Context)
		close(task2Started)
	})

	// Start tasks
	err := r.StartTask(task1, trigger)
	assert.NoError(t, err)
	err = r.StartTask(task2, trigger)
	assert.NoError(t, err)

	// Wait for both goroutines to start and contexts to be captured
	select {
	case <-task1Started:
		// task1 has started
	case <-time.After(1 * time.Second):
		t.Fatal("task1 goroutine failed to start within timeout")
	}
	select {
	case <-task2Started:
		// task2 has started
	case <-time.After(1 * time.Second):
		t.Fatal("task2 goroutine failed to start within timeout")
	}

	assert.True(t, r.IsRunning(task1.ID))
	assert.True(t, r.IsRunning(task2.ID))

	// Stop the runner
	// This should cancel all tasks and wait for them
	done := make(chan struct{})
	go func() {
		r.Stop()
		close(done)
	}()

	// Wait for Stop to return
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Runner.Stop() timed out")
	}

	// Verify contexts are canceled
	select {
	case <-ctx1.Done():
	default:
		t.Error("Task 1 context should be canceled")
	}
	select {
	case <-ctx2.Done():
	default:
		t.Error("Task 2 context should be canceled")
	}

	// Verify tasks are removed from running map
	// Since cleanup defer runs before wg.Done(), and Stop waits for wg.Wait(),
	// the map should be empty now.
	assert.False(t, r.IsRunning(task1.ID), "Task 1 should be removed from running map")
	assert.False(t, r.IsRunning(task2.ID), "Task 2 should be removed from running map")

	mockEngine.AssertExpectations(t)
}
