package subscription

import (
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
)

// CacheStatusBus is a specialized event bus for ConnectionCacheStatus.
type CacheStatusBus = GenericEventBus[*model.ConnectionCacheStatus]

// CacheStatusSubscriber is a subscriber for ConnectionCacheStatus.
type CacheStatusSubscriber = GenericSubscriber[*model.ConnectionCacheStatus]

// NewCacheStatusBus creates a new ConnectionCacheStatus event bus.
func NewCacheStatusBus() *CacheStatusBus {
	return NewGenericEventBus[*model.ConnectionCacheStatus](100)
}

// CacheStatusFilter creates a filter function for ConnectionCacheStatus based on connection ID.
func CacheStatusFilter(connectionID uuid.UUID) func(*model.ConnectionCacheStatus) bool {
	return func(event *model.ConnectionCacheStatus) bool {
		return event.ConnectionID == connectionID
	}
}
