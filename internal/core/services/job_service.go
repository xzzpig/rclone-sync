package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
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
func (s *JobService) CreateJob(ctx context.Context, taskID uuid.UUID, trigger model.JobTrigger) (*ent.Job, error) {
	s.logger.Info("Creating new job", zap.String("task_id", taskID.String()), zap.Stringer("trigger", trigger))
	j, err := s.client.Job.Create().
		SetTaskID(taskID).
		SetTrigger(trigger).
		SetStatus(model.JobStatusPending).
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
		SetStatus(model.JobStatus(status))

	if status == string(model.JobStatusSuccess) || status == string(model.JobStatusFailed) || status == string(model.JobStatusCancelled) {
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
func (s *JobService) UpdateJobStats(ctx context.Context, jobID uuid.UUID, files, bytes, filesDeleted, errorCount int64) (*ent.Job, error) {
	j, err := s.client.Job.UpdateOneID(jobID).
		SetFilesTransferred(int(files)).
		SetBytesTransferred(bytes).
		SetFilesDeleted(int(filesDeleted)).
		SetErrorCount(int(errorCount)).
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
		SetLevel(model.LogLevel(level)).
		SetWhat(model.LogAction(what)).
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

// ResetStuckJobs marks all jobs that are still in 'running' state as 'cancelled'.
// This is typically called on application startup to handle crash recovery.
// It also calculates and updates the files_transferred and bytes_transferred
// statistics from the job logs before marking the job as cancelled.
func (s *JobService) ResetStuckJobs(ctx context.Context) error {
	s.logger.Info("Checking for stuck running jobs...")

	// Find all jobs that are still in 'running' state
	stuckJobs, err := s.client.Job.Query().
		Where(job.StatusEQ(model.JobStatusRunning)).
		All(ctx)

	if err != nil {
		return errors.Join(errs.ErrSystem, err)
	}

	if len(stuckJobs) == 0 {
		return nil
	}

	s.logger.Info("Found stuck jobs", zap.Int("count", len(stuckJobs)))

	// Process each stuck job
	for _, j := range stuckJobs {
		// Calculate statistics from job logs
		// Count files transferred (UPLOAD, DOWNLOAD, MOVE with INFO level)
		logs, err := s.client.JobLog.Query().
			Where(
				joblog.JobIDEQ(j.ID),
				joblog.LevelEQ(model.LogLevelInfo),
				joblog.WhatIn(model.LogActionUpload, model.LogActionDownload, model.LogActionMove),
			).
			All(ctx)

		if err != nil {
			s.logger.Error("Failed to query job logs for stuck job",
				zap.String("job_id", j.ID.String()),
				zap.Error(err))
			continue
		}

		// Calculate totals
		filesTransferred := len(logs)
		var bytesTransferred int64
		for _, log := range logs {
			bytesTransferred += log.Size
		}

		// Update the job with statistics and mark as cancelled
		_, err = s.client.Job.UpdateOneID(j.ID).
			SetFilesTransferred(filesTransferred).
			SetBytesTransferred(bytesTransferred).
			SetStatus(model.JobStatusCancelled).
			SetErrors("System crash or unexpected shutdown").
			SetEndTime(time.Now()).
			Save(ctx)

		if err != nil {
			s.logger.Error("Failed to update stuck job",
				zap.String("job_id", j.ID.String()),
				zap.Error(err))
			continue
		}

		s.logger.Info("Reset stuck job with statistics",
			zap.String("job_id", j.ID.String()),
			zap.Int("files_transferred", filesTransferred),
			zap.Int64("bytes_transferred", bytesTransferred))
	}

	s.logger.Info("Reset stuck jobs completed", zap.Int("count", len(stuckJobs)))
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
		query.Where(joblog.LevelEQ(model.LogLevel(level)))
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

// ListJobLogsByJobPaginated lists job logs for a specific job with pagination.
func (s *JobService) ListJobLogsByJobPaginated(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]*ent.JobLog, int, error) {
	query := s.client.JobLog.Query().
		Where(joblog.HasJobWith(job.ID(jobID))).
		Order(ent.Asc(joblog.FieldTime))

	// Get total count
	totalCount, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, errors.Join(errs.ErrSystem, err)
	}

	// Apply pagination and fetch items
	logs, err := query.
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, errors.Join(errs.ErrSystem, err)
	}

	return logs, totalCount, nil
}

// DeleteOldLogsForConnection deletes old logs for a connection, keeping only the newest keepCount logs.
// Returns the number of logs deleted.
func (s *JobService) DeleteOldLogsForConnection(ctx context.Context, connectionID uuid.UUID, keepCount int) (int, error) {
	s.logger.Info("Deleting old logs for connection",
		zap.String("connection_id", connectionID.String()),
		zap.Int("keep_count", keepCount))

	// Query all log IDs for this connection, ordered by time descending (newest first)
	// Skip the first keepCount (which we want to keep), and collect the rest for deletion
	idsToDelete, err := s.client.JobLog.Query().
		Where(joblog.HasJobWith(
			job.HasTaskWith(
				task.ConnectionIDEQ(connectionID),
			),
		)).
		Order(ent.Desc(joblog.FieldTime)).
		Offset(keepCount).
		IDs(ctx)

	if err != nil {
		return 0, errors.Join(errs.ErrSystem, err)
	}

	if len(idsToDelete) == 0 {
		return 0, nil
	}

	// Batch delete the old logs
	deleted, err := s.client.JobLog.Delete().
		Where(joblog.IDIn(idsToDelete...)).
		Exec(ctx)

	if err != nil {
		return 0, errors.Join(errs.ErrSystem, err)
	}

	s.logger.Info("Deleted old logs for connection",
		zap.String("connection_id", connectionID.String()),
		zap.Int("deleted_count", deleted))

	return deleted, nil
}

// DeleteJob deletes a job by ID.
// This will cascade delete all associated job logs.
func (s *JobService) DeleteJob(ctx context.Context, jobID uuid.UUID) error {
	s.logger.Info("Deleting job", zap.String("job_id", jobID.String()))

	err := s.client.Job.DeleteOneID(jobID).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.Join(errs.ErrNotFound, err)
		}
		return errors.Join(errs.ErrSystem, err)
	}

	s.logger.Info("Deleted job successfully", zap.String("job_id", jobID.String()))
	return nil
}

var _ ports.JobService = (*JobService)(nil)
