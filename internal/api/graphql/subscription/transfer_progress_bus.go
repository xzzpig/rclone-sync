package subscription

import (
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
)

// TransferProgressBus is a specialized event bus for TransferProgressEvent.
type TransferProgressBus = GenericEventBus[*model.TransferProgressEvent]

// TransferProgressSubscriber is a subscriber for TransferProgressEvent.
type TransferProgressSubscriber = GenericSubscriber[*model.TransferProgressEvent]

// NewTransferProgressBus creates a new TransferProgressEvent event bus.
// Uses a default buffer size of 100.
func NewTransferProgressBus() *TransferProgressBus {
	return NewGenericEventBus[*model.TransferProgressEvent](100)
}

// TransferProgressFilter creates a filter function for TransferProgressEvent.
// Filters by connectionID, taskID, and/or jobID if provided.
// Pass nil for any parameter to skip that filter.
func TransferProgressFilter(connectionID, taskID, jobID *uuid.UUID) func(*model.TransferProgressEvent) bool {
	// If no filters specified, accept all events
	if connectionID == nil && taskID == nil && jobID == nil {
		return nil
	}

	return func(event *model.TransferProgressEvent) bool {
		if connectionID != nil && event.ConnectionID != *connectionID {
			return false
		}
		if taskID != nil && event.TaskID != *taskID {
			return false
		}
		if jobID != nil && event.JobID != *jobID {
			return false
		}
		return true
	}
}
