// Package runner provides task execution management for the application.
package runner

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"go.uber.org/zap"
)

type runInfo struct {
	cancel context.CancelFunc
	runID  uuid.UUID
}

// Runner manages the execution of sync tasks.
type Runner struct {
	syncEngine ports.SyncEngine
	logger     *zap.Logger
	mu         sync.Mutex
	running    map[uuid.UUID]runInfo
	wg         sync.WaitGroup
}

// NewRunner creates a new Runner instance.
func NewRunner(syncEngine ports.SyncEngine) *Runner {
	return &Runner{
		syncEngine: syncEngine,
		logger:     logger.Named("core.runner"),
		running:    make(map[uuid.UUID]runInfo),
	}
}

// Start initializes the runner (no-op currently, reserved for future use).
func (r *Runner) Start() {}

// Stop cancels all running tasks and waits for them to finish.
func (r *Runner) Stop() {
	r.logger.Info("Stopping runner, cancelling all tasks...")
	r.mu.Lock()
	for id, info := range r.running {
		r.logger.Info("Cancelling task", zap.Stringer("task_id", id))
		info.cancel()
	}
	// We don't clear r.running here immediately, let the goroutines cleanup
	r.mu.Unlock()

	r.logger.Info("Waiting for running tasks to finish...")
	r.wg.Wait()
	r.logger.Info("Runner stopped")
}

// StartTask starts a task execution asynchronously.
// It cancels any existing execution of the same task before starting a new one.
func (r *Runner) StartTask(task *ent.Task, trigger job.Trigger) error {
	taskID := task.ID
	runID := uuid.New()

	r.mu.Lock()
	// Cancel existing run if any
	if info, ok := r.running[taskID]; ok {
		r.logger.Info("Cancelling existing task execution", zap.Stringer("task_id", taskID), zap.Stringer("old_run_id", info.runID))
		info.cancel()
	}

	// Create new context for this run
	ctx, cancel := context.WithCancel(context.Background())
	r.running[taskID] = runInfo{
		cancel: cancel,
		runID:  runID,
	}
	r.mu.Unlock()

	// Run asynchronously
	r.wg.Go(func() {
		defer func() {
			r.mu.Lock()
			// Only delete if it's still the SAME execution
			if info, ok := r.running[taskID]; ok && info.runID == runID {
				delete(r.running, taskID)
			}
			r.mu.Unlock()
		}()

		r.logger.Info("Starting task execution", zap.Stringer("task_id", taskID), zap.Stringer("run_id", runID), zap.Stringer("trigger", trigger))
		// The error is already handled and logged within RunTask (e.g., job status updated).
		// We don't need to log it again here.
		err := r.syncEngine.RunTask(ctx, task, trigger)
		if err != nil {
			r.logger.Error("Task execution failed", zap.Stringer("task_id", taskID), zap.Stringer("run_id", runID), zap.Error(err))
		}
	})
	return nil
}

// StopTask cancels a running task.
func (r *Runner) StopTask(taskID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if info, ok := r.running[taskID]; ok {
		r.logger.Info("Stopping task", zap.Stringer("task_id", taskID))
		info.cancel()
		delete(r.running, taskID)
	}
	return nil
}

// IsRunning checks if a task is currently running.
func (r *Runner) IsRunning(taskID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.running[taskID]
	return ok
}

var _ ports.Runner = (*Runner)(nil)
