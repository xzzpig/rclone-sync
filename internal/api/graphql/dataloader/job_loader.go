package dataloader

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vikstrous/dataloadgen"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
)

// JobLoader batches and caches Job loads.
type JobLoader = dataloadgen.Loader[uuid.UUID, *ent.Job]

type jobReader struct {
	client *ent.Client
}

func (r *jobReader) getJobs(ctx context.Context, ids []uuid.UUID) ([]*ent.Job, []error) {
	jobs, err := r.client.Job.Query().
		Where(job.IDIn(ids...)).
		All(ctx)
	if err != nil {
		errs := make([]error, len(ids))
		for i := range errs {
			errs[i] = err
		}
		return nil, errs
	}

	// Build a map for O(1) lookup
	jobMap := make(map[uuid.UUID]*ent.Job, len(jobs))
	for _, j := range jobs {
		jobMap[j.ID] = j
	}

	// Return results in the same order as requested IDs
	result := make([]*ent.Job, len(ids))
	errs := make([]error, len(ids))
	for i, id := range ids {
		if j, ok := jobMap[id]; ok {
			result[i] = j
		} else {
			errs[i] = fmt.Errorf("job not found: %s", id)
		}
	}

	return result, errs
}

// NewJobLoader creates a new JobLoader.
func NewJobLoader(client *ent.Client) *JobLoader {
	reader := &jobReader{client: client}
	return dataloadgen.NewLoader(reader.getJobs)
}
