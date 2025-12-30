// Package resolver contains GraphQL resolver helper functions.
// These helper functions are separated to avoid being overwritten by gqlgen code generation.
package resolver

import (
	"time"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

// entConnectionToModel converts an ent Connection to a GraphQL model Connection.
func entConnectionToModel(c *ent.Connection) *model.Connection {
	return &model.Connection{
		ID:        c.ID,
		Name:      c.Name,
		Type:      c.Type,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// entTaskToModel converts an ent Task to a GraphQL model Task.
func entTaskToModel(t *ent.Task) *model.Task {
	var schedule *string
	if t.Schedule != "" {
		schedule = &t.Schedule
	}

	return &model.Task{
		ID:           t.ID,
		Name:         t.Name,
		SourcePath:   t.SourcePath,
		RemotePath:   t.RemotePath,
		Direction:    t.Direction,
		Schedule:     schedule,
		Realtime:     t.Realtime,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
		ConnectionID: t.ConnectionID, // FK for dataloader optimization
	}
}

// entJobToModel converts an ent Job to a GraphQL model Job.
func entJobToModel(j *ent.Job) *model.Job {
	var errStr *string
	if j.Errors != "" {
		errStr = &j.Errors
	}

	// EndTime is time.Time in ent, but *time.Time in model
	// Check if it's zero value to determine if it should be nil
	var endTime *time.Time
	if !j.EndTime.IsZero() {
		endTime = &j.EndTime
	}

	return &model.Job{
		ID:               j.ID,
		Status:           j.Status,
		Trigger:          j.Trigger,
		StartTime:        j.StartTime,
		EndTime:          endTime,
		FilesTransferred: j.FilesTransferred,
		BytesTransferred: j.BytesTransferred,
		FilesDeleted:     j.FilesDeleted,
		ErrorCount:       j.ErrorCount,
		Errors:           errStr,
		TaskID:           j.TaskID, // FK for dataloader optimization
	}
}

// entJobLogToModel converts an ent JobLog to a GraphQL model JobLog.
func entJobLogToModel(l *ent.JobLog) *model.JobLog {
	return &model.JobLog{
		ID:    l.ID,
		Level: l.Level,
		Time:  l.Time,
		Path:  l.Path,
		What:  l.What,
		Size:  l.Size,
		JobID: l.JobID, // FK for dataloader optimization
	}
}

// buildOptions converts TaskSyncOptionsInput to TaskSyncOptions for database storage.
// It only includes fields that are explicitly set (non-nil).
func buildOptions(input *model.TaskSyncOptionsInput) *model.TaskSyncOptions {
	if input == nil {
		return nil
	}

	options := &model.TaskSyncOptions{
		ConflictResolution: input.ConflictResolution,
		Filters:            input.Filters,
		NoDelete:           input.NoDelete,
		Transfers:          input.Transfers,
	}

	// Return nil if all fields are empty
	if options.ConflictResolution == nil && len(options.Filters) == 0 && options.NoDelete == nil && options.Transfers == nil {
		return nil
	}

	return options
}
