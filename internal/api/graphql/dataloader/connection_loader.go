package dataloader

import (
	"context"

	"github.com/google/uuid"
	"github.com/vikstrous/dataloadgen"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/connection"
)

// ConnectionLoader batches and caches Connection loads.
type ConnectionLoader = dataloadgen.Loader[uuid.UUID, *ent.Connection]

// NewConnectionLoader creates a new ConnectionLoader.
func NewConnectionLoader(client *ent.Client) *ConnectionLoader {
	return NewGenericLoader(
		func(ctx context.Context, ids []uuid.UUID) ([]*ent.Connection, error) {
			return client.Connection.Query().Where(connection.IDIn(ids...)).All(ctx)
		},
		func(c *ent.Connection) uuid.UUID { return c.ID },
		"connection",
	)
}
