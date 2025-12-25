package dataloader

import (
	"context"

	"github.com/google/uuid"
	"github.com/vikstrous/dataloadgen"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
)

// JobLoader batches and caches Job loads.
type JobLoader = dataloadgen.Loader[uuid.UUID, *ent.Job]

// NewJobLoader creates a new JobLoader.
func NewJobLoader(client *ent.Client) *JobLoader {
	return NewGenericLoader(
		func(ctx context.Context, ids []uuid.UUID) ([]*ent.Job, error) {
			return client.Job.Query().Where(job.IDIn(ids...)).All(ctx)
		},
		func(j *ent.Job) uuid.UUID { return j.ID },
		"job",
	)
}
