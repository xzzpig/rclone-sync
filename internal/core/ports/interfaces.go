// Package ports defines interfaces for core application services.
// These interfaces allow for dependency inversion, making the system
// more modular, testable, and maintainable.
package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

// Runner manages the lifecycle of background tasks.
type Runner interface {
	Start()
	Stop()
	StartTask(task *ent.Task, trigger model.JobTrigger) error
	StopTask(taskID uuid.UUID) error
	IsRunning(taskID uuid.UUID) bool
}

// SyncEngine executes the actual sync operation for a task.
type SyncEngine interface {
	RunTask(ctx context.Context, task *ent.Task, trigger model.JobTrigger) error
}

// Watcher defines the interface for file watching operations.
type Watcher interface {
	Start()
	Stop()
	AddTask(task *ent.Task) error
	RemoveTask(task *ent.Task) error
}

// Scheduler defines the interface for scheduled task operations.
type Scheduler interface {
	Start()
	Stop()
	AddTask(task *ent.Task) error
	RemoveTask(task *ent.Task) error
}

// TaskService provides CRUD operations for tasks.
type TaskService interface {
	GetTask(ctx context.Context, id uuid.UUID) (*ent.Task, error)
	GetTaskWithConnection(ctx context.Context, id uuid.UUID) (*ent.Task, error)
	ListAllTasks(ctx context.Context) ([]*ent.Task, error)
	// Add other methods as needed for testing
}

// JobService defines the interface for job management operations.
type JobService interface {
	CreateJob(ctx context.Context, taskID uuid.UUID, trigger model.JobTrigger) (*ent.Job, error)
	UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string, errStr string) (*ent.Job, error)
	UpdateJobStats(ctx context.Context, jobID uuid.UUID, files, bytes int64) (*ent.Job, error)
	AddJobLog(ctx context.Context, jobID uuid.UUID, level, what, path string, size int64) (*ent.JobLog, error)
	AddJobLogsBatch(ctx context.Context, jobID uuid.UUID, logs []*ent.JobLog) error
	GetJob(ctx context.Context, jobID uuid.UUID) (*ent.Job, error)
	GetLastJobByTaskID(ctx context.Context, taskID uuid.UUID) (*ent.Job, error)
	ListJobs(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID, limit, offset int) ([]*ent.Job, error)
	CountJobs(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID) (int, error)
	GetJobWithLogs(ctx context.Context, jobID uuid.UUID) (*ent.Job, error)
	ListJobLogs(ctx context.Context, connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string, limit, offset int) ([]*ent.JobLog, error)
	CountJobLogs(ctx context.Context, connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string) (int, error)
}

// ConnectionService defines the interface for connection management operations.
type ConnectionService interface {
	CreateConnection(ctx context.Context, name, providerType string, config map[string]string) (*ent.Connection, error)
	ListConnections(ctx context.Context) ([]*ent.Connection, error)
	ListConnectionNames(ctx context.Context) ([]string, error)
	GetConnectionByName(ctx context.Context, name string) (*ent.Connection, error)
	GetConnectionConfig(ctx context.Context, name string) (map[string]string, error)
	UpdateConnection(ctx context.Context, id uuid.UUID, name, connType *string, config map[string]string) error
	DeleteConnectionByName(ctx context.Context, name string) error
	HasAssociatedTasks(ctx context.Context, connectionID uuid.UUID) (bool, error)
}
