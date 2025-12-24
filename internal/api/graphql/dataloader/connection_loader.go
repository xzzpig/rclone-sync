package dataloader

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vikstrous/dataloadgen"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/connection"
)

// ConnectionLoader batches and caches Connection loads.
type ConnectionLoader = dataloadgen.Loader[uuid.UUID, *ent.Connection]

type connectionReader struct {
	client *ent.Client
}

func (r *connectionReader) getConnections(ctx context.Context, ids []uuid.UUID) ([]*ent.Connection, []error) {
	connections, err := r.client.Connection.Query().
		Where(connection.IDIn(ids...)).
		All(ctx)
	if err != nil {
		errs := make([]error, len(ids))
		for i := range errs {
			errs[i] = err
		}
		return nil, errs
	}

	// Build a map for O(1) lookup
	connMap := make(map[uuid.UUID]*ent.Connection, len(connections))
	for _, c := range connections {
		connMap[c.ID] = c
	}

	// Return results in the same order as requested IDs
	result := make([]*ent.Connection, len(ids))
	errs := make([]error, len(ids))
	for i, id := range ids {
		if c, ok := connMap[id]; ok {
			result[i] = c
		} else {
			errs[i] = fmt.Errorf("connection not found: %s", id)
		}
	}

	return result, errs
}

// NewConnectionLoader creates a new ConnectionLoader.
func NewConnectionLoader(client *ent.Client) *ConnectionLoader {
	reader := &connectionReader{client: client}
	return dataloadgen.NewLoader(reader.getConnections)
}
