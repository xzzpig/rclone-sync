package dataloader

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vikstrous/dataloadgen"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/task"
)

// TaskLoader batches and caches Task loads.
type TaskLoader = dataloadgen.Loader[uuid.UUID, *ent.Task]

type taskReader struct {
	client *ent.Client
}

func (r *taskReader) getTasks(ctx context.Context, ids []uuid.UUID) ([]*ent.Task, []error) {
	tasks, err := r.client.Task.Query().
		Where(task.IDIn(ids...)).
		All(ctx)
	if err != nil {
		errs := make([]error, len(ids))
		for i := range errs {
			errs[i] = err
		}
		return nil, errs
	}

	// Build a map for O(1) lookup
	taskMap := make(map[uuid.UUID]*ent.Task, len(tasks))
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	// Return results in the same order as requested IDs
	result := make([]*ent.Task, len(ids))
	errs := make([]error, len(ids))
	for i, id := range ids {
		if t, ok := taskMap[id]; ok {
			result[i] = t
		} else {
			errs[i] = fmt.Errorf("task not found: %s", id)
		}
	}

	return result, errs
}

// NewTaskLoader creates a new TaskLoader.
func NewTaskLoader(client *ent.Client) *TaskLoader {
	reader := &taskReader{client: client}
	return dataloadgen.NewLoader(reader.getTasks)
}
