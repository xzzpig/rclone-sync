package subscription

import (
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
)

// JobProgressBus is a specialized event bus for JobProgressEvent.
type JobProgressBus = GenericEventBus[*model.JobProgressEvent]

// JobProgressSubscriber is a subscriber for JobProgressEvent.
type JobProgressSubscriber = GenericSubscriber[*model.JobProgressEvent]

// NewJobProgressBus creates a new JobProgressEvent event bus.
// Uses a default buffer size of 100.
func NewJobProgressBus() *JobProgressBus {
	return NewGenericEventBus[*model.JobProgressEvent](100)
}

// JobProgressFilter creates a filter function for JobProgressEvent.
// Filters by taskID and/or connectionID if provided.
// Pass nil for either parameter to skip that filter.
func JobProgressFilter(taskID, connectionID *uuid.UUID) func(*model.JobProgressEvent) bool {
	// If no filters specified, accept all events
	if taskID == nil && connectionID == nil {
		return nil
	}

	return func(event *model.JobProgressEvent) bool {
		if taskID != nil && event.TaskID != *taskID {
			return false
		}
		if connectionID != nil && event.ConnectionID != *connectionID {
			return false
		}
		return true
	}
}
