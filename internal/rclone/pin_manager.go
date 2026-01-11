package rclone

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/cache"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/rclone/backend/metacache"
	"go.uber.org/zap"
)

// PinManager manages pinned MetaCache Fs instances for connections with cache enabled.
// When a connection has cache enabled, the MetaCache Fs is pinned to keep it alive
// so that ChangeNotify subscriptions remain active even when no sync tasks are running.
type PinManager struct {
	mu             sync.RWMutex
	pinnedFs       map[string]fs.Fs
	cacheStatusBus *subscription.CacheStatusBus
	timers         map[string]*time.Timer
	logger         *zap.Logger
}

// NewPinManager creates a new PinManager instance.
func NewPinManager(cacheStatusBus *subscription.CacheStatusBus) *PinManager {
	return &PinManager{
		pinnedFs:       make(map[string]fs.Fs),
		cacheStatusBus: cacheStatusBus,
		timers:         make(map[string]*time.Timer),
		logger:         logger.Named("pin_manager"),
	}
}

// PinConnection pins the MetaCache Fs for a connection if cache is enabled.
func (pm *PinManager) PinConnection(conn *ent.Connection) error {
	if conn.Options == nil || conn.Options.Cache == nil || !conn.Options.Cache.Enabled {
		pm.logger.Debug("Cache not enabled for connection, skipping pin",
			zap.String("connection_id", conn.ID.String()),
			zap.String("connection_name", conn.Name))
		return nil
	}

	connID := conn.ID.String()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, ok := pm.pinnedFs[connID]; ok {
		pm.logger.Debug("Connection already pinned",
			zap.String("connection_id", connID),
			zap.String("connection_name", conn.Name))
		return nil
	}

	remotePath := conn.Name + CacheSuffix + ":"
	f, err := cache.Get(context.Background(), remotePath)
	if err != nil {
		pm.logger.Error("Failed to get MetaCache Fs for connection",
			zap.String("connection_id", connID),
			zap.String("connection_name", conn.Name),
			zap.Error(err))
		return err
	}

	cache.Pin(f)
	pm.pinnedFs[connID] = f

	updateFn := func() {
		pm.mu.Lock()
		defer pm.mu.Unlock()
		if _, ok := pm.timers[connID]; ok {
			return
		}
		pm.timers[connID] = time.AfterFunc(5*time.Second, func() {
			pm.mu.Lock()
			delete(pm.timers, connID)
			pm.mu.Unlock()
			if pm.IsPinned(connID) {
				pm.publishCacheStatus(conn.ID)
			}
		})
	}

	// Subscribe to cache store changes
	cacheName := GetCacheRemoteName(conn.Name)
	if store := metacache.GetCacheStoreIfExists(cacheName); store != nil {
		store.SetOnChange(updateFn)
	}

	if notifyer, ok := f.(fs.ChangeNotifier); ok {
		pm.logger.Info("Subscribing to ChangeNotify for real-time status updates",
			zap.String("connection_id", connID))

		notifyer.ChangeNotify(context.Background(), func(path string, _ fs.EntryType) {
			pm.logger.Debug("Received ChangeNotify",
				zap.String("connection_id", connID),
				zap.String("path", path))
			updateFn()
		}, nil)
	}

	pm.logger.Info("Pinned MetaCache Fs for connection",
		zap.String("connection_id", connID),
		zap.String("connection_name", conn.Name))

	return nil
}

// publishCacheStatus fetches current cache status and publishes it to the bus.
func (pm *PinManager) publishCacheStatus(connID uuid.UUID) {
	status := pm.GetCacheStatus(connID)
	pm.cacheStatusBus.Publish(status)
}

// PublishCacheStatus fetches current cache status and publishes it to the bus.
func (pm *PinManager) PublishCacheStatus(connID uuid.UUID) {
	pm.publishCacheStatus(connID)
}

// GetCacheStatus returns the current cache status for a connection.
func (pm *PinManager) GetCacheStatus(connID uuid.UUID) *model.ConnectionCacheStatus {
	connIDStr := connID.String()
	running := pm.IsPinned(connIDStr)

	var changeNotifySupported bool
	var entriesCount int
	var dbSizeBytes *int64
	var lastNotifyTime *time.Time

	var store *metacache.CacheStore
	if f := pm.GetPinnedFs(connIDStr); f != nil {
		changeNotifySupported = f.Features().ChangeNotify != nil
		store = metacache.GetCacheStoreIfExists(f.Name())
	}
	if store != nil {
		count, _ := store.GetEntriesCount()
		entriesCount = int(count)
		size, _ := store.GetDBSize()
		dbSizeBytes = &size
		t := store.GetLastNotifyTime()
		if !t.IsZero() {
			lastNotifyTime = &t
		}
	}

	return &model.ConnectionCacheStatus{
		ConnectionID:          connID,
		Running:               running,
		ChangeNotifySupported: changeNotifySupported,
		EntriesCount:          entriesCount,
		DbSizeBytes:           dbSizeBytes,
		LastNotifyTime:        lastNotifyTime,
	}
}

// UnpinConnection unpins the MetaCache Fs for a connection and releases resources.
func (pm *PinManager) UnpinConnection(connID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	f, ok := pm.pinnedFs[connID]
	if !ok {
		pm.logger.Debug("Connection not pinned, skipping unpin",
			zap.String("connection_id", connID))
		return
	}

	if shutdowner, ok := f.(fs.Shutdowner); ok {
		ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()
		if err := shutdowner.Shutdown(ctx); err != nil {
			pm.logger.Warn("Error shutting down MetaCache Fs",
				zap.String("connection_id", connID),
				zap.Error(err))
		}
	}

	if timer, ok := pm.timers[connID]; ok {
		timer.Stop()
		delete(pm.timers, connID)
	}

	if mcFs, ok := f.(*metacache.Fs); ok {
		if store := mcFs.GetCacheStore(); store != nil {
			store.SetOnChange(nil)
		}
	}

	cache.Unpin(f)
	delete(pm.pinnedFs, connID)

	pm.logger.Info("Unpinned MetaCache Fs for connection",
		zap.String("connection_id", connID))
}

// IsPinned returns true if the connection has a pinned MetaCache Fs.
func (pm *PinManager) IsPinned(connID string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, ok := pm.pinnedFs[connID]
	return ok
}

// GetPinnedFs returns the pinned Fs for a connection, or nil if not pinned.
func (pm *PinManager) GetPinnedFs(connID string) fs.Fs {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.pinnedFs[connID]
}

// InitPinnedConnections initializes pinned Fs instances for all connections with cache enabled.
func (pm *PinManager) InitPinnedConnections(_ context.Context, connections []*ent.Connection) error {
	pm.logger.Info("Initializing pinned connections", zap.Int("count", len(connections)))

	var pinCount int
	for _, conn := range connections {
		if conn.Options == nil || conn.Options.Cache == nil || !conn.Options.Cache.Enabled {
			continue
		}

		if err := pm.PinConnection(conn); err != nil {
			pm.logger.Warn("Failed to pin connection during initialization",
				zap.String("connection_id", conn.ID.String()),
				zap.String("connection_name", conn.Name),
				zap.Error(err))
			continue
		}
		pinCount++
	}

	pm.logger.Info("Completed pinned connection initialization",
		zap.Int("pinned_count", pinCount),
		zap.Int("total_connections", len(connections)))

	return nil
}

// ShutdownAll unpins all connections and releases resources.
func (pm *PinManager) ShutdownAll() {
	pm.mu.Lock()
	connIDs := make([]string, 0, len(pm.pinnedFs))
	for connID := range pm.pinnedFs {
		connIDs = append(connIDs, connID)
	}
	pm.mu.Unlock()

	pm.logger.Info("Shutting down all pinned connections", zap.Int("count", len(connIDs)))

	for _, connID := range connIDs {
		pm.UnpinConnection(connID)
	}
}

// ShutdownTimeout is the maximum time to wait for a Fs shutdown.
const ShutdownTimeout = 60 * time.Second
