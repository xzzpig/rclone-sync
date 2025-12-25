package dataloader

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vikstrous/dataloadgen"
)

// QueryFunc is the batch query function type.
type QueryFunc[T any] func(ctx context.Context, ids []uuid.UUID) ([]*T, error)

// IDFunc is the function type for getting entity ID.
type IDFunc[T any] func(*T) uuid.UUID

// NewGenericLoader creates a generic dataloader for any entity type.
func NewGenericLoader[T any](
	queryFunc QueryFunc[T],
	idFunc IDFunc[T],
	entityName string,
) *dataloadgen.Loader[uuid.UUID, *T] {
	fetch := func(ctx context.Context, ids []uuid.UUID) ([]*T, []error) {
		entities, err := queryFunc(ctx, ids)
		if err != nil {
			errs := make([]error, len(ids))
			for i := range errs {
				errs[i] = err
			}
			return nil, errs
		}

		// Build a map for O(1) lookup
		entityMap := make(map[uuid.UUID]*T, len(entities))
		for _, e := range entities {
			entityMap[idFunc(e)] = e
		}

		// Return results in the same order as requested IDs
		result := make([]*T, len(ids))
		errs := make([]error, len(ids))
		for i, id := range ids {
			if e, ok := entityMap[id]; ok {
				result[i] = e
			} else {
				errs[i] = fmt.Errorf("%s not found: %s", entityName, id) //nolint:err113
			}
		}

		return result, errs
	}

	return dataloadgen.NewLoader(fetch)
}
