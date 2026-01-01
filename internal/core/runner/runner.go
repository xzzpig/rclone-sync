// Package runner provides task execution management for the application.
package runner

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"go.uber.org/zap"
)

type runInfo struct {
	cancel context.CancelFunc
	runID  uuid.UUID
	done   chan struct{} // 任务完成时关闭，用于同步等待
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
// For Realtime triggers, it skips if the task is already running to avoid interrupting ongoing syncs.
// For Manual and Scheduled triggers, it cancels any existing execution before starting a new one.
func (r *Runner) StartTask(task *ent.Task, trigger model.JobTrigger) error {
	taskID := task.ID
	runID := uuid.New()

	r.mu.Lock()
	// Check if task is already running
	if info, ok := r.running[taskID]; ok {
		// For Realtime triggers, skip if task is already running
		// This prevents file system events (like downloads) from canceling the ongoing sync
		if trigger == model.JobTriggerRealtime {
			r.mu.Unlock()
			r.logger.Debug("Task already running, skipping realtime trigger",
				zap.Stringer("task_id", taskID),
				zap.Stringer("existing_run_id", info.runID))
			return nil
		}
		// For other triggers (Manual, Scheduled), cancel existing run and wait for it to finish
		r.logger.Info("Cancelling existing task execution", zap.Stringer("task_id", taskID), zap.Stringer("old_run_id", info.runID))
		info.cancel()
		// Wait for the old task to finish while holding the lock
		// This is safe because the goroutine only closes the channel without acquiring the lock
		<-info.done
		r.logger.Debug("Old task execution finished", zap.Stringer("task_id", taskID), zap.Stringer("old_run_id", info.runID))
		delete(r.running, taskID)
	}

	// Create new context for this run
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	r.running[taskID] = runInfo{
		cancel: cancel,
		runID:  runID,
		done:   done,
	}
	r.mu.Unlock()

	// Run asynchronously
	r.wg.Go(func() {
		defer func() {
			close(done)
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
		<-info.done
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
