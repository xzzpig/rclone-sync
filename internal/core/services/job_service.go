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
	"go.uber.org/zap"
)

type JobService struct {
	client *ent.Client
	logger *zap.Logger
}

func NewJobService(client *ent.Client) *JobService {
	return &JobService{
		client: client,
		logger: logger.L.Named("job-service"),
	}
}

// CreateJob creates a new job for a task.
func (s *JobService) CreateJob(ctx context.Context, taskID uuid.UUID, trigger string) (*ent.Job, error) {
	s.logger.Info("Creating new job", zap.String("task_id", taskID.String()), zap.String("trigger", trigger))
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
func (s *JobService) AddJobLog(ctx context.Context, jobID uuid.UUID, level, message, path string) (*ent.JobLog, error) {
	l, err := s.client.JobLog.Create().
		SetJobID(jobID).
		SetLevel(joblog.Level(level)).
		SetMessage(message).
		SetNillablePath(&path).
		SetTime(time.Now()).
		Save(ctx)
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
		builders[i] = s.client.JobLog.Create().
			SetJobID(jobID).
			SetLevel(l.Level).
			SetMessage(l.Message).
			SetNillablePath(&l.Path).
			SetTime(l.Time)
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
		SetStatus(job.StatusFailed).
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

// ListJobs retrieves jobs with optional filtering (taskID) and pagination.
func (s *JobService) ListJobs(ctx context.Context, taskID *uuid.UUID, limit, offset int) ([]*ent.Job, error) {
	query := s.client.Job.Query().
		Order(ent.Desc(job.FieldStartTime)).
		Limit(limit).
		Offset(offset).
		WithTask()

	if taskID != nil {
		query.Where(job.HasTaskWith(task.ID(*taskID)))
	}

	jobs, err := query.All(ctx)
	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return jobs, nil
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
