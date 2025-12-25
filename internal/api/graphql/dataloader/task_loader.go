package dataloader

import (
	"context"

	"github.com/google/uuid"
	"github.com/vikstrous/dataloadgen"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/task"
)

// TaskLoader batches and caches Task loads.
type TaskLoader = dataloadgen.Loader[uuid.UUID, *ent.Task]

// NewTaskLoader creates a new TaskLoader.
func NewTaskLoader(client *ent.Client) *TaskLoader {
	return NewGenericLoader(
		func(ctx context.Context, ids []uuid.UUID) ([]*ent.Task, error) {
			return client.Task.Query().Where(task.IDIn(ids...)).All(ctx)
		},
		func(t *ent.Task) uuid.UUID { return t.ID },
		"task",
	)
}
