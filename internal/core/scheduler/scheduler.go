// Package scheduler provides cron-based task scheduling for the application.
package scheduler

import (
	"context"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"go.uber.org/zap"
)

// Scheduler manages scheduled task executions using cron.
type Scheduler struct {
	cron    *cron.Cron
	taskSvc ports.TaskService
	runner  ports.Runner
	logger  *zap.Logger
	mu      sync.Mutex
	jobMap  map[string]cron.EntryID // Maps task ID to cron EntryID
	running bool
}

// NewScheduler creates a new Scheduler instance.
func NewScheduler(taskSvc ports.TaskService, runner ports.Runner, opts ...cron.Option) *Scheduler {
	return &Scheduler{
		cron:    cron.New(opts...), // Standard 5-field cron (minute, hour, day, month, weekday)
		taskSvc: taskSvc,
		runner:  runner,
		logger:  logger.Named("core.scheduler"),
		jobMap:  make(map[string]cron.EntryID),
	}
}

// Start starts the scheduler and loads scheduled tasks from the database.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		s.logger.Warn("Scheduler is already running")
		return
	}
	s.logger.Info("Starting scheduler")
	s.cron.Start()
	s.loadScheduledTasks()
	s.running = true
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		s.logger.Warn("Scheduler is not running")
		return
	}
	s.logger.Info("Stopping scheduler")
	s.cron.Stop()
	s.running = false
}

func (s *Scheduler) loadScheduledTasks() {
	s.logger.Info("Loading scheduled tasks from database")
	tasks, err := s.taskSvc.ListAllTasks(context.Background())
	if err != nil {
		s.logger.Error("Failed to load tasks for scheduler", zap.Error(err))
		return
	}

	for _, task := range tasks {
		if task.Schedule != "" {
			if err := s.addJob(task); err != nil {
				s.logger.Error("Failed to add task to scheduler on load",
					zap.String("task_name", task.Name),
					zap.String("task_id", task.ID.String()),
					zap.String("schedule", task.Schedule),
					zap.Error(err),
				)
			}
		}
	}
	s.logger.Info("Finished loading scheduled tasks", zap.Int("count", len(s.jobMap)))
}

// AddTask adds a task to the scheduler.
func (s *Scheduler) AddTask(task *ent.Task) error {
	if task.Schedule == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addJob(task)
}

// RemoveTask removes a task from the scheduler.
func (s *Scheduler) RemoveTask(task *ent.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeJob(task.ID.String())
	return nil
}

func (s *Scheduler) addJob(task *ent.Task) error {
	taskIDStr := task.ID.String()
	taskID := task.ID     // Capture uuid.UUID directly for use in closure
	taskName := task.Name // For logging

	s.removeJob(taskIDStr) // Remove existing job if any, to handle updates

	entryID, err := s.cron.AddFunc(task.Schedule, func() {
		s.logger.Info("Running scheduled task", zap.String("task_name", taskName), zap.String("task_id", taskIDStr))

		// Reload task from database to get the latest configuration
		ctx := context.Background()
		currentTask, err := s.taskSvc.GetTaskWithConnection(ctx, taskID)
		if err != nil {
			s.logger.Error("Failed to get task for scheduled run",
				zap.String("task_id", taskIDStr),
				zap.Error(err))
			return
		}

		_ = s.runner.StartTask(currentTask, model.JobTriggerSchedule)
	})

	if err != nil {
		return err
	}

	s.jobMap[taskIDStr] = entryID
	s.logger.Info("Scheduled task added", zap.String("task_name", task.Name), zap.String("schedule", task.Schedule))
	return nil
}

func (s *Scheduler) removeJob(taskID string) {
	if entryID, ok := s.jobMap[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.jobMap, taskID)
		s.logger.Info("Removed task from scheduler", zap.String("task_id", taskID))
	}
}

var _ ports.Scheduler = (*Scheduler)(nil)
