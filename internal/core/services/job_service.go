package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
	"github.com/xzzpig/rclone-sync/internal/core/ent/joblog"
	"github.com/xzzpig/rclone-sync/internal/core/ent/task"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"go.uber.org/zap"
)

// JobService provides operations for managing jobs and job logs.
type JobService struct {
	client *ent.Client
	logger *zap.Logger
}

// NewJobService creates a new JobService instance.
func NewJobService(client *ent.Client) *JobService {
	return &JobService{
		client: client,
		logger: logger.Named("service.job"),
	}
}

// CreateJob creates a new job for a task.
func (s *JobService) CreateJob(ctx context.Context, taskID uuid.UUID, trigger job.Trigger) (*ent.Job, error) {
	s.logger.Info("Creating new job", zap.String("task_id", taskID.String()), zap.Stringer("trigger", trigger))
	j, err := s.client.Job.Create().
		SetTaskID(taskID).
		SetTrigger(job.Trigger(trigger)).
		SetStatus(job.StatusPending).
		SetStartTime(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return j, nil
}

// UpdateJobStatus updates the status of a job.
func (s *JobService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string, errStr string) (*ent.Job, error) {
	update := s.client.Job.UpdateOneID(jobID).
		SetStatus(job.Status(status))

	if status == string(job.StatusSuccess) || status == string(job.StatusFailed) || status == string(job.StatusCancelled) {
		update.SetEndTime(time.Now())
	}

	if errStr != "" {
		update.SetErrors(errStr)
	}

	j, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return j, nil
}

// UpdateJobStats updates the statistics of a job.
func (s *JobService) UpdateJobStats(ctx context.Context, jobID uuid.UUID, files, bytes int64) (*ent.Job, error) {
	j, err := s.client.Job.UpdateOneID(jobID).
		SetFilesTransferred(int(files)).
		SetBytesTransferred(bytes).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return j, nil
}

// AddJobLog adds a log entry for a job.
func (s *JobService) AddJobLog(ctx context.Context, jobID uuid.UUID, level, what, path string, size int64) (*ent.JobLog, error) {
	create := s.client.JobLog.Create().
		SetJobID(jobID).
		SetLevel(joblog.Level(level)).
		SetWhat(joblog.What(what)).
		SetNillablePath(&path).
		SetTime(time.Now())

	if size > 0 {
		create.SetSize(size)
	}

	l, err := create.Save(ctx)
	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return l, nil
}

// AddJobLogsBatch adds multiple log entries for a job in a batch.
func (s *JobService) AddJobLogsBatch(ctx context.Context, jobID uuid.UUID, logs []*ent.JobLog) error {
	if len(logs) == 0 {
		return nil
	}
	builders := make([]*ent.JobLogCreate, len(logs))
	for i, l := range logs {
		builder := s.client.JobLog.Create().
			SetJobID(jobID).
			SetLevel(l.Level).
			SetWhat(l.What).
			SetNillablePath(&l.Path).
			SetTime(l.Time)

		if l.Size > 0 {
			builder.SetSize(l.Size)
		}

		builders[i] = builder
	}
	_, err := s.client.JobLog.CreateBulk(builders...).Save(ctx)
	if err != nil {
		return errors.Join(errs.ErrSystem, err)
	}
	return nil
}

// ResetStuckJobs marks all jobs that are still in 'running' state as 'failed'.
// This is typically called on application startup to handle crash recovery.
func (s *JobService) ResetStuckJobs(ctx context.Context) error {
	s.logger.Info("Checking for stuck running jobs...")
	count, err := s.client.Job.Update().
		Where(job.StatusEQ(job.StatusRunning)).
		SetStatus(job.StatusCancelled).
		SetErrors("System crash or unexpected shutdown").
		SetEndTime(time.Now()).
		Save(ctx)

	if err != nil {
		return errors.Join(errs.ErrSystem, err)
	}

	if count > 0 {
		s.logger.Info("Reset stuck jobs", zap.Int("count", count))
	}
	return nil
}

// GetJob retrieves a job by ID.
func (s *JobService) GetJob(ctx context.Context, jobID uuid.UUID) (*ent.Job, error) {
	j, err := s.client.Job.Get(ctx, jobID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return j, nil
}

// GetLastJobByTaskID retrieves the most recent job for a task.
func (s *JobService) GetLastJobByTaskID(ctx context.Context, taskID uuid.UUID) (*ent.Job, error) {
	j, err := s.client.Job.Query().
		Where(job.HasTaskWith(task.ID(taskID))).
		Order(ent.Desc(job.FieldStartTime)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return j, nil
}

func (s *JobService) buildJobQuery(taskID *uuid.UUID, connectionID *uuid.UUID) *ent.JobQuery {
	query := s.client.Job.Query()

	if taskID != nil {
		query.Where(job.HasTaskWith(task.ID(*taskID)))
	}

	if connectionID != nil {
		query.Where(job.HasTaskWith(task.ConnectionIDEQ(*connectionID)))
	}

	return query
}

// ListJobs retrieves jobs with optional filtering (taskID, connectionID) and pagination.
func (s *JobService) ListJobs(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID, limit, offset int) ([]*ent.Job, error) {
	query := s.buildJobQuery(taskID, connectionID)
	jobs, err := query.
		Order(ent.Desc(job.FieldStartTime)).
		Limit(limit).
		Offset(offset).
		WithTask().
		All(ctx)

	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return jobs, nil
}

// CountJobs returns the total count of jobs with optional filtering.
func (s *JobService) CountJobs(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID) (int, error) {
	query := s.buildJobQuery(taskID, connectionID)
	count, err := query.Count(ctx)
	if err != nil {
		return 0, errors.Join(errs.ErrSystem, err)
	}
	return count, nil
}

// GetJobWithLogs retrieves a job by ID, including its logs.
func (s *JobService) GetJobWithLogs(ctx context.Context, jobID uuid.UUID) (*ent.Job, error) {
	j, err := s.client.Job.Query().
		Where(job.ID(jobID)).
		WithTask().
		WithLogs(func(q *ent.JobLogQuery) {
			q.Order(ent.Asc(joblog.FieldTime))
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return j, nil
}

func (s *JobService) buildJobLogQuery(connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string) *ent.JobLogQuery {
	query := s.client.JobLog.Query()

	// Filter by connection_id through job -> task relationship
	if connectionID != nil {
		query.Where(joblog.HasJobWith(job.HasTaskWith(task.ConnectionIDEQ(*connectionID))))
	}

	// Optional: filter by task_id
	if taskID != nil {
		query.Where(joblog.HasJobWith(job.HasTaskWith(task.ID(*taskID))))
	}

	// Optional: filter by job_id
	if jobID != nil {
		query.Where(joblog.HasJobWith(job.ID(*jobID)))
	}

	// Optional: filter by level
	if level != "" {
		query.Where(joblog.LevelEQ(joblog.Level(level)))
	}

	return query
}

// ListJobLogs retrieves job logs with flexible filtering.
// Optional: connectionID, taskID, jobID, level
func (s *JobService) ListJobLogs(ctx context.Context, connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string, limit, offset int) ([]*ent.JobLog, error) {
	query := s.buildJobLogQuery(connectionID, taskID, jobID, level)
	logs, err := query.
		Order(ent.Desc(joblog.FieldTime)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return logs, nil
}

// CountJobLogs returns the total count of job logs with flexible filtering.
func (s *JobService) CountJobLogs(ctx context.Context, connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string) (int, error) {
	query := s.buildJobLogQuery(connectionID, taskID, jobID, level)
	count, err := query.Count(ctx)
	if err != nil {
		return 0, errors.Join(errs.ErrSystem, err)
	}
	return count, nil
}

var _ ports.JobService = (*JobService)(nil)
