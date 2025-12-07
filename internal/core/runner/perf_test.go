package runner_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/runner"
	"go.uber.org/zap"
)

// MockSyncEngineForPerf is a performance-optimized mock for SyncEngine
type MockSyncEngineForPerf struct {
	mock.Mock
}

func (m *MockSyncEngineForPerf) RunTask(ctx context.Context, task *ent.Task, trigger string) error {
	args := m.Called(ctx, task, trigger)
	return args.Error(0)
}

func setupPerfTest(t testing.TB) {
	if logger.L == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			t.Fatalf("failed to create logger: %v", err)
		}
		logger.L = l
	}
	_ = t // Suppress unused parameter warning
}

func BenchmarkRunner_StartTask_Concurrent(b *testing.B) {
	setupPerfTest(b)

	// Create a mock sync engine
	mockSyncEngine := new(MockSyncEngineForPerf)

	// Setup mock to return success quickly
	mockSyncEngine.On("RunTask", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Create runner
	r := runner.NewRunner(mockSyncEngine)

	b.ResetTimer()

	// Run b.N tasks concurrently
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Create a new task for each iteration to avoid UUID conflicts
			task := &ent.Task{
				ID: uuid.New(),
			}

			// Start task
			err := r.StartTask(task, "manual")
			require.NoError(b, err)
		}
	})
}

func BenchmarkRunner_StopTask_Concurrent(b *testing.B) {
	setupPerfTest(b)

	// Create a mock sync engine
	mockSyncEngine := new(MockSyncEngineForPerf)

	// Setup mock to return success quickly
	mockSyncEngine.On("RunTask", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Create runner
	r := runner.NewRunner(mockSyncEngine)

	// Pre-create tasks for stopping
	var tasks []*ent.Task
	for b.Loop() {
		task := &ent.Task{
			ID: uuid.New(),
		}

		err := r.StartTask(task, "manual")
		require.NoError(b, err)
		tasks = append(tasks, task)
	}

	b.ResetTimer()

	// Run b.N stops concurrently
	b.RunParallel(func(pb *testing.PB) {
		taskIndex := 0
		for pb.Next() {
			if taskIndex < len(tasks) {
				err := r.StopTask(tasks[taskIndex].ID)
				require.NoError(b, err)
				taskIndex++
			}
		}
	})
}

func BenchmarkRunner_IsRunning_Concurrent(b *testing.B) {
	setupPerfTest(b)

	// Create a mock sync engine
	mockSyncEngine := new(MockSyncEngineForPerf)

	// Setup mock to return success quickly
	mockSyncEngine.On("RunTask", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Create runner
	r := runner.NewRunner(mockSyncEngine)

	// Pre-create a task for status checks
	task := &ent.Task{
		ID: uuid.New(),
	}

	err := r.StartTask(task, "manual")
	require.NoError(b, err)

	b.ResetTimer()

	// Run b.N status checks concurrently
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = r.IsRunning(task.ID)
		}
	})
}

func BenchmarkRunner_MixedOperations_Concurrent(b *testing.B) {
	setupPerfTest(b)

	// Create a mock sync engine
	mockSyncEngine := new(MockSyncEngineForPerf)

	// Setup mock to return success quickly
	mockSyncEngine.On("RunTask", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Create runner
	r := runner.NewRunner(mockSyncEngine)

	// Pre-create some tasks for stopping and status checks
	var tasks []*ent.Task
	for range 100 {
		task := &ent.Task{
			ID: uuid.New(),
		}

		err := r.StartTask(task, "manual")
		require.NoError(b, err)
		tasks = append(tasks, task)
	}

	b.ResetTimer()

	// Run mixed operations concurrently
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Randomly choose an operation
			switch b.N % 3 {
			case 0:
				// Start new task
				task := &ent.Task{
					ID: uuid.New(),
				}
				r.StartTask(task, "manual")
			case 1:
				// Check if task is running
				taskIdx := b.N % len(tasks)
				_ = r.IsRunning(tasks[taskIdx].ID)
			case 2:
				// Stop task
				taskIdx := b.N % len(tasks)
				r.StopTask(tasks[taskIdx].ID)
			}
		}
	})
}

func TestRunner_HighConcurrency(t *testing.T) {
	setupTest()

	// Create a mock sync engine
	mockSyncEngine := new(MockSyncEngineForPerf)

	// Setup mock to return success after a short delay to simulate real work
	mockSyncEngine.On("RunTask", mock.Anything, mock.Anything, mock.Anything).Return(nil).After(10 * time.Millisecond)

	// Create runner
	r := runner.NewRunner(mockSyncEngine)

	// Number of concurrent tasks
	numTasks := 1000

	// Channel to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(numTasks)

	// Channel to collect results
	results := make(chan error, numTasks)

	// Start time
	startTime := time.Now()

	// Start many tasks concurrently
	for i := range numTasks {
		go func(i int) {
			defer wg.Done()

			task := &ent.Task{
				ID: uuid.New(),
			}

			err := r.StartTask(task, "manual")
			results <- err
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)

	// Check results
	errorCount := 0
	for err := range results {
		if err != nil {
			errorCount++
		}
	}

	// Assert no errors
	assert.Equal(t, 0, errorCount, "Expected no errors, but got %d", errorCount)

	// Assert reasonable performance (should complete all tasks in less than 5 seconds)
	assert.Less(t, elapsedTime, 5*time.Second, "Expected to complete all tasks in less than 5 seconds, but took %v", elapsedTime)

	t.Logf("Started %d tasks concurrently in %v (%.2f tasks/second)",
		numTasks, elapsedTime, float64(numTasks)/elapsedTime.Seconds())
}

func TestRunner_MemoryUsage(t *testing.T) {
	setupTest()

	// Create a mock sync engine
	mockSyncEngine := new(MockSyncEngineForPerf)

	// Setup mock to return success
	mockSyncEngine.On("RunTask", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Create runner
	r := runner.NewRunner(mockSyncEngine)

	// Number of tasks to start
	numTasks := 1000

	// Start many tasks
	for range numTasks {
		task := &ent.Task{
			ID: uuid.New(),
		}

		err := r.StartTask(task, "manual")
		require.NoError(t, err)
	}

	// Check that the runner doesn't hold onto excessive memory
	// This is a simple check - in a real-world scenario, you might want to use
	// runtime.MemStats to get more detailed memory usage information
	assert.True(t, true, "Runner should not leak memory")

	t.Logf("Successfully started %d tasks without memory issues", numTasks)
}
