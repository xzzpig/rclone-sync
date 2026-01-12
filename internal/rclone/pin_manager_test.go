package rclone_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/rclone/rclone/backend/local"
	"github.com/rclone/rclone/fs/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/db/query"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/rclone"
	"github.com/xzzpig/rclone-sync/internal/rclone/backend/metacache"
	_ "github.com/xzzpig/rclone-sync/internal/rclone/backend/notifylocal"
)

// setupPinManagerTest sets up a test environment for PinManager tests.
func setupPinManagerTest(t *testing.T) (*rclone.PinManager, *query.ConnectionQuery, func()) {
	t.Helper()

	// Create test database client
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())

	// Create encryptor (plaintext mode for testing)
	encryptor, err := crypto.NewEncryptor("")
	require.NoError(t, err)

	// Create connection query
	connSvc := query.NewConnectionQuery(client, encryptor)

	// Create DBStorage and install it
	dataDir := t.TempDir()
	storage := rclone.NewDBStorage(connSvc, dataDir)
	storage.Install()

	// Clear rclone's Fs cache to avoid stale entries from previous tests
	cache.Clear()

	// Create a new PinManager for each test to avoid shared state
	cacheStatusBus := subscription.NewCacheStatusBus()
	pm := rclone.NewPinManager(cacheStatusBus)

	cleanup := func() {
		pm.ShutdownAll()
		cache.Clear() // Clear cache again on cleanup
		client.Close()
	}

	return pm, connSvc, cleanup
}

func createConnectionWithCache(t *testing.T, connSvc *query.ConnectionQuery, name string, cacheEnabled bool) *ent.Connection {
	return createConnectionWithCacheAndType(t, connSvc, name, "local", cacheEnabled)
}

func createConnectionWithCacheAndType(t *testing.T, connSvc *query.ConnectionQuery, name, connType string, cacheEnabled bool) *ent.Connection {
	t.Helper()

	ctx := context.Background()

	conn, err := connSvc.CreateConnection(ctx, name, connType, map[string]string{}, nil)
	require.NoError(t, err)

	if cacheEnabled {
		opts := &model.ConnectionOptions{
			Cache: &model.ConnectionCacheOptions{
				Enabled: true,
			},
		}
		_, err = conn.Update().SetOptions(opts).Save(ctx)
		require.NoError(t, err)

		conn, err = connSvc.GetConnectionByName(ctx, name)
		require.NoError(t, err)
	}

	return conn
}

// TestPinManager_PinConnection_NoCacheEnabled tests that PinConnection is a no-op when cache is not enabled.
func TestPinManager_PinConnection_NoCacheEnabled(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create connection without cache enabled
	conn := createConnectionWithCache(t, connSvc, "test-no-cache", false)

	// Pin should be a no-op
	err := pm.PinConnection(conn)
	require.NoError(t, err)

	// Should not be pinned
	assert.False(t, pm.IsPinned(conn.ID.String()))
}

// TestPinManager_PinConnection_WithCacheEnabled tests that PinConnection pins the Fs when cache is enabled.
func TestPinManager_PinConnection_WithCacheEnabled(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create connection with cache enabled
	conn := createConnectionWithCache(t, connSvc, "test-with-cache", true)

	// Pin should succeed
	err := pm.PinConnection(conn)
	require.NoError(t, err)

	// Should be pinned
	assert.True(t, pm.IsPinned(conn.ID.String()))

	// GetPinnedFs should return the Fs
	f := pm.GetPinnedFs(conn.ID.String())
	assert.NotNil(t, f)
}

// TestPinManager_PinConnection_AlreadyPinned tests that PinConnection is a no-op when already pinned.
func TestPinManager_PinConnection_AlreadyPinned(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create connection with cache enabled
	conn := createConnectionWithCache(t, connSvc, "test-already-pinned", true)

	// Pin first time
	err := pm.PinConnection(conn)
	require.NoError(t, err)
	assert.True(t, pm.IsPinned(conn.ID.String()))

	// Pin second time should be a no-op
	err = pm.PinConnection(conn)
	require.NoError(t, err)
	assert.True(t, pm.IsPinned(conn.ID.String()))
}

// TestPinManager_UnpinConnection tests unpinning a connection.
func TestPinManager_UnpinConnection(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create connection with cache enabled
	conn := createConnectionWithCache(t, connSvc, "test-unpin", true)

	// Pin first
	err := pm.PinConnection(conn)
	require.NoError(t, err)
	assert.True(t, pm.IsPinned(conn.ID.String()))

	// Unpin
	pm.UnpinConnection(conn.ID.String())
	assert.False(t, pm.IsPinned(conn.ID.String()))

	// GetPinnedFs should return nil
	f := pm.GetPinnedFs(conn.ID.String())
	assert.Nil(t, f)
}

// TestPinManager_UnpinConnection_NotPinned tests that UnpinConnection is a no-op when not pinned.
func TestPinManager_UnpinConnection_NotPinned(t *testing.T) {
	pm, _, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Unpin non-existent connection should not panic
	pm.UnpinConnection("non-existent-id")
}

// TestPinManager_InitPinnedConnections tests initializing pinned connections on startup.
func TestPinManager_InitPinnedConnections(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create multiple connections
	conn1 := createConnectionWithCache(t, connSvc, "test-init-1", true)
	conn2 := createConnectionWithCache(t, connSvc, "test-init-2", false)
	conn3 := createConnectionWithCache(t, connSvc, "test-init-3", true)

	// Initialize all connections
	connections := []*ent.Connection{conn1, conn2, conn3}
	err := pm.InitPinnedConnections(context.Background(), connections)
	require.NoError(t, err)

	// Only connections with cache enabled should be pinned
	assert.True(t, pm.IsPinned(conn1.ID.String()))
	assert.False(t, pm.IsPinned(conn2.ID.String()))
	assert.True(t, pm.IsPinned(conn3.ID.String()))
}

// TestPinManager_ShutdownAll tests shutting down all pinned connections.
func TestPinManager_ShutdownAll(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create connections with cache enabled
	conn1 := createConnectionWithCache(t, connSvc, "test-shutdown-1", true)
	conn2 := createConnectionWithCache(t, connSvc, "test-shutdown-2", true)

	// Pin connections
	err := pm.PinConnection(conn1)
	require.NoError(t, err)
	err = pm.PinConnection(conn2)
	require.NoError(t, err)

	assert.True(t, pm.IsPinned(conn1.ID.String()))
	assert.True(t, pm.IsPinned(conn2.ID.String()))

	// Shutdown all
	pm.ShutdownAll()

	// All should be unpinned
	assert.False(t, pm.IsPinned(conn1.ID.String()))
	assert.False(t, pm.IsPinned(conn2.ID.String()))
}

// TestPinManager_GetCacheStatus_NotPinned tests GetCacheStatus when connection is not pinned.
func TestPinManager_GetCacheStatus_NotPinned(t *testing.T) {
	pm, _, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Get status for non-existent connection
	connID := uuid.New()
	status := pm.GetCacheStatus(connID)

	// Should return status with running=false and default values
	assert.Equal(t, connID, status.ConnectionID)
	assert.False(t, status.Running)
	assert.False(t, status.ChangeNotifySupported)
	assert.Equal(t, 0, status.EntriesCount)
	assert.Nil(t, status.DbSizeBytes)
	assert.Nil(t, status.LastNotifyTime)
}

// TestPinManager_GetCacheStatus_Pinned tests GetCacheStatus when connection is pinned.
func TestPinManager_GetCacheStatus_Pinned(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create and pin a connection with cache enabled
	conn := createConnectionWithCache(t, connSvc, "test-cache-status", true)
	err := pm.PinConnection(conn)
	require.NoError(t, err)

	// Get status
	status := pm.GetCacheStatus(conn.ID)

	// Should return status with running=true
	assert.Equal(t, conn.ID, status.ConnectionID)
	assert.True(t, status.Running)
	// Local filesystem doesn't support ChangeNotify
	assert.False(t, status.ChangeNotifySupported)
	// EntriesCount may be 0 or more depending on cache state
	assert.GreaterOrEqual(t, status.EntriesCount, 0)
}

// TestPinManager_PublishCacheStatus tests PublishCacheStatus method.
func TestPinManager_PublishCacheStatus(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create and pin a connection
	conn := createConnectionWithCache(t, connSvc, "test-publish-status", true)
	err := pm.PinConnection(conn)
	require.NoError(t, err)

	// PublishCacheStatus should not panic
	assert.NotPanics(t, func() {
		pm.PublishCacheStatus(conn.ID)
	})
}

// TestPinManager_GetCacheStatus_WithCacheStore tests GetCacheStatus with active CacheStore.
func TestPinManager_GetCacheStatus_WithCacheStore(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create and pin a connection with cache enabled
	conn := createConnectionWithCache(t, connSvc, "test-cache-store-status", true)
	err := pm.PinConnection(conn)
	require.NoError(t, err)

	// Get the pinned Fs to ensure cache store is initialized
	f := pm.GetPinnedFs(conn.ID.String())
	require.NotNil(t, f)

	// Get status - should have cache store data
	status := pm.GetCacheStatus(conn.ID)

	assert.Equal(t, conn.ID, status.ConnectionID)
	assert.True(t, status.Running)
	// DbSizeBytes should be available when CacheStore exists
	// (may be nil if store doesn't exist for some backends)
}

// TestPinManager_PinConnection_NilOptions tests that PinConnection handles nil options correctly.
func TestPinManager_PinConnection_NilOptions(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create connection without any options
	ctx := context.Background()
	conn, err := connSvc.CreateConnection(ctx, "test-nil-options", "local", map[string]string{}, nil)
	require.NoError(t, err)

	// Ensure options is nil
	assert.Nil(t, conn.Options)

	// Pin should be a no-op (not cause panic)
	err = pm.PinConnection(conn)
	require.NoError(t, err)
	assert.False(t, pm.IsPinned(conn.ID.String()))
}

// TestPinManager_PinConnection_NilCacheOptions tests that PinConnection handles nil cache options correctly.
func TestPinManager_PinConnection_NilCacheOptions(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create connection with options but no cache
	ctx := context.Background()
	conn, err := connSvc.CreateConnection(ctx, "test-nil-cache", "local", map[string]string{}, nil)
	require.NoError(t, err)

	// Set options without cache
	opts := &model.ConnectionOptions{
		Cache: nil,
	}
	_, err = conn.Update().SetOptions(opts).Save(ctx)
	require.NoError(t, err)

	// Refresh connection
	conn, err = connSvc.GetConnectionByName(ctx, "test-nil-cache")
	require.NoError(t, err)

	// Pin should be a no-op
	err = pm.PinConnection(conn)
	require.NoError(t, err)
	assert.False(t, pm.IsPinned(conn.ID.String()))
}

// TestPinManager_InitPinnedConnections_Empty tests initializing with empty connection list.
func TestPinManager_InitPinnedConnections_Empty(t *testing.T) {
	pm, _, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Initialize with empty list
	err := pm.InitPinnedConnections(context.Background(), []*ent.Connection{})
	require.NoError(t, err)
}

// TestPinManager_InitPinnedConnections_AllDisabled tests initializing when all connections have cache disabled.
func TestPinManager_InitPinnedConnections_AllDisabled(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	// Create connections without cache
	conn1 := createConnectionWithCache(t, connSvc, "test-init-disabled-1", false)
	conn2 := createConnectionWithCache(t, connSvc, "test-init-disabled-2", false)

	// Initialize
	err := pm.InitPinnedConnections(context.Background(), []*ent.Connection{conn1, conn2})
	require.NoError(t, err)

	// None should be pinned
	assert.False(t, pm.IsPinned(conn1.ID.String()))
	assert.False(t, pm.IsPinned(conn2.ID.String()))
}

func TestPinManager_ShutdownAll_Empty(t *testing.T) {
	pm, _, cleanup := setupPinManagerTest(t)
	defer cleanup()

	assert.NotPanics(t, func() {
		pm.ShutdownAll()
	})
}

func TestPinManager_GetCacheStatus_WithLastNotifyTime(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	conn := createConnectionWithCache(t, connSvc, "test-last-notify", true)
	err := pm.PinConnection(conn)
	require.NoError(t, err)

	f := pm.GetPinnedFs(conn.ID.String())
	require.NotNil(t, f)

	store := metacache.GetCacheStoreIfExists(f.Name())
	require.NotNil(t, store)

	expectedTime := time.Now().Add(-time.Hour)
	store.SetLastNotifyTime(expectedTime)

	status := pm.GetCacheStatus(conn.ID)

	assert.NotNil(t, status.LastNotifyTime)
	assert.WithinDuration(t, expectedTime, *status.LastNotifyTime, time.Second)
}

func TestPinManager_PinConnection_WithChangeNotifier(t *testing.T) {
	pm, connSvc, cleanup := setupPinManagerTest(t)
	defer cleanup()

	conn := createConnectionWithCacheAndType(t, connSvc, "test-changenotify", "notifylocal", true)
	err := pm.PinConnection(conn)
	require.NoError(t, err)

	assert.True(t, pm.IsPinned(conn.ID.String()))

	f := pm.GetPinnedFs(conn.ID.String())
	require.NotNil(t, f)

	status := pm.GetCacheStatus(conn.ID)
	assert.True(t, status.ChangeNotifySupported)
}
