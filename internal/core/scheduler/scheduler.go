package scheduler

import (
	"context"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"go.uber.org/zap"
)

type Scheduler struct {
	cron    *cron.Cron
	taskSvc ports.TaskService
	runner  ports.Runner
	logger  *zap.Logger
	mu      sync.Mutex
	jobMap  map[string]cron.EntryID // Maps task ID to cron EntryID
	running bool
}

func NewScheduler(taskSvc ports.TaskService, runner ports.Runner) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()), // Support seconds in cron spec
		taskSvc: taskSvc,
		runner:  runner,
		logger:  logger.L.Named("scheduler"),
		jobMap:  make(map[string]cron.EntryID),
	}
}

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

func (s *Scheduler) AddTask(task *ent.Task) error {
	if task.Schedule == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addJob(task)
}

func (s *Scheduler) RemoveTask(task *ent.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeJob(task.ID.String())
	return nil
}

func (s *Scheduler) addJob(task *ent.Task) error {
	taskIDStr := task.ID.String()
	s.removeJob(taskIDStr) // Remove existing job if any, to handle updates

	entryID, err := s.cron.AddFunc(task.Schedule, func() {
		s.logger.Info("Running scheduled task", zap.String("task_name", task.Name), zap.String("task_id", taskIDStr))
		// Run in a new context to avoid cancellation from other jobs
		s.runner.StartTask(task, "schedule")
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
