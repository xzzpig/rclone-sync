package rclone

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/db/query"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
)

func setupStorageTest(t *testing.T) (*DBStorage, *query.ConnectionQuery, string) {
	t.Helper()

	// Create test database client
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	t.Cleanup(func() { client.Close() })

	// Create encryptor (plaintext mode for testing)
	encryptor, err := crypto.NewEncryptor("")
	require.NoError(t, err)

	// Create connection query
	connSvc := query.NewConnectionQuery(client, encryptor)

	// Create DBStorage (use temp dir for cache)
	dataDir := t.TempDir()
	storage := NewDBStorage(connSvc, dataDir)

	return storage, connSvc, dataDir
}

// T045: 单元测试：DBStorage.GetValue
func TestDBStorage_GetValue(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)
	ctx := context.Background()

	// Create a test connection
	config := map[string]string{
		"type":     "onedrive",
		"token":    `{"access_token":"test-token"}`,
		"drive_id": "abc123",
	}
	_, err := connSvc.CreateConnection(ctx, "test-remote", "onedrive", config, nil)
	require.NoError(t, err)

	t.Run("get existing key", func(t *testing.T) {
		value, ok := storage.GetValue("test-remote", "type")
		assert.True(t, ok)
		assert.Equal(t, "onedrive", value)
	})

	t.Run("get existing key - token", func(t *testing.T) {
		value, ok := storage.GetValue("test-remote", "token")
		assert.True(t, ok)
		assert.Equal(t, `{"access_token":"test-token"}`, value)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		value, ok := storage.GetValue("test-remote", "non-existing")
		assert.False(t, ok)
		assert.Empty(t, value)
	})

	t.Run("get key from non-existing section", func(t *testing.T) {
		value, ok := storage.GetValue("non-existing-remote", "type")
		assert.False(t, ok)
		assert.Empty(t, value)
	})
}

// T046: 单元测试：DBStorage.SetValue
func TestDBStorage_SetValue(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)
	ctx := context.Background()

	t.Run("set value on existing connection", func(t *testing.T) {
		// Create a test connection first
		config := map[string]string{
			"type":  "s3",
			"token": "old-token",
		}
		_, err := connSvc.CreateConnection(ctx, "update-remote", "s3", config, nil)
		require.NoError(t, err)

		// Update a value
		storage.SetValue("update-remote", "token", "new-token")

		// Verify the update
		value, ok := storage.GetValue("update-remote", "token")
		assert.True(t, ok)
		assert.Equal(t, "new-token", value)
	})

	t.Run("set new key on existing connection", func(t *testing.T) {
		// Add a new key
		storage.SetValue("update-remote", "new_key", "new_value")

		// Verify
		value, ok := storage.GetValue("update-remote", "new_key")
		assert.True(t, ok)
		assert.Equal(t, "new_value", value)
	})

	t.Run("set value creates new connection if not exists", func(t *testing.T) {
		// Set value on non-existing section (creates new connection)
		storage.SetValue("new-remote", "type", "gdrive")

		// Verify connection was created
		assert.True(t, storage.HasSection("new-remote"))
		value, ok := storage.GetValue("new-remote", "type")
		assert.True(t, ok)
		assert.Equal(t, "gdrive", value)
	})

	t.Run("token refresh scenario - simulates rclone token update", func(t *testing.T) {
		// Create OAuth connection
		config := map[string]string{
			"type":  "onedrive",
			"token": `{"access_token":"old","refresh_token":"xxx","expiry":"2024-01-01T00:00:00Z"}`,
		}
		_, err := connSvc.CreateConnection(ctx, "oauth-remote", "onedrive", config, nil)
		require.NoError(t, err)

		// Simulate rclone refreshing the token
		newToken := `{"access_token":"new","refresh_token":"xxx","expiry":"2025-01-01T00:00:00Z"}`
		storage.SetValue("oauth-remote", "token", newToken)

		// Verify token was updated
		value, ok := storage.GetValue("oauth-remote", "token")
		assert.True(t, ok)
		assert.Equal(t, newToken, value)
	})
}

// T047: 单元测试：DBStorage.HasSection
func TestDBStorage_HasSection(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)
	ctx := context.Background()

	// Create a test connection
	config := map[string]string{"type": "local"}
	_, err := connSvc.CreateConnection(ctx, "existing-remote", "local", config, nil)
	require.NoError(t, err)

	t.Run("existing section returns true", func(t *testing.T) {
		assert.True(t, storage.HasSection("existing-remote"))
	})

	t.Run("non-existing section returns false", func(t *testing.T) {
		assert.False(t, storage.HasSection("non-existing-remote"))
	})

	t.Run("empty section name returns false", func(t *testing.T) {
		assert.False(t, storage.HasSection(""))
	})
}

func TestDBStorage_GetSectionList(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)
	ctx := context.Background()

	t.Run("empty database returns empty list", func(t *testing.T) {
		sections := storage.GetSectionList()
		assert.Empty(t, sections)
	})

	t.Run("returns all connection names", func(t *testing.T) {
		// Create multiple connections
		_, err := connSvc.CreateConnection(ctx, "remote-a", "s3", map[string]string{"type": "s3"}, nil)
		require.NoError(t, err)
		_, err = connSvc.CreateConnection(ctx, "remote-b", "gdrive", map[string]string{"type": "gdrive"}, nil)
		require.NoError(t, err)
		_, err = connSvc.CreateConnection(ctx, "remote-c", "onedrive", map[string]string{"type": "onedrive"}, nil)
		require.NoError(t, err)

		sections := storage.GetSectionList()
		assert.Len(t, sections, 3)
		assert.Contains(t, sections, "remote-a")
		assert.Contains(t, sections, "remote-b")
		assert.Contains(t, sections, "remote-c")
	})
}

func TestDBStorage_GetKeyList(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)
	ctx := context.Background()

	config := map[string]string{
		"type":     "onedrive",
		"token":    "xxx",
		"drive_id": "abc",
	}
	_, err := connSvc.CreateConnection(ctx, "keylist-remote", "onedrive", config, nil)
	require.NoError(t, err)

	t.Run("returns all keys for existing section", func(t *testing.T) {
		keys := storage.GetKeyList("keylist-remote")
		assert.Len(t, keys, 3)
		assert.Contains(t, keys, "type")
		assert.Contains(t, keys, "token")
		assert.Contains(t, keys, "drive_id")
	})

	t.Run("returns nil for non-existing section", func(t *testing.T) {
		keys := storage.GetKeyList("non-existing")
		assert.Nil(t, keys)
	})
}

func TestDBStorage_DeleteKey(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)
	ctx := context.Background()

	config := map[string]string{
		"type":       "s3",
		"key_id":     "xxx",
		"access_key": "yyy",
	}
	_, err := connSvc.CreateConnection(ctx, "delete-key-remote", "s3", config, nil)
	require.NoError(t, err)

	t.Run("delete existing key", func(t *testing.T) {
		result := storage.DeleteKey("delete-key-remote", "access_key")
		assert.True(t, result)

		// Verify key is deleted
		_, ok := storage.GetValue("delete-key-remote", "access_key")
		assert.False(t, ok)

		// Other keys should still exist
		value, ok := storage.GetValue("delete-key-remote", "key_id")
		assert.True(t, ok)
		assert.Equal(t, "xxx", value)
	})

	t.Run("delete non-existing key returns false", func(t *testing.T) {
		result := storage.DeleteKey("delete-key-remote", "non-existing-key")
		assert.False(t, result)
	})

	t.Run("delete from non-existing section returns false", func(t *testing.T) {
		result := storage.DeleteKey("non-existing-remote", "type")
		assert.False(t, result)
	})
}

func TestDBStorage_DeleteSection(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)
	ctx := context.Background()

	config := map[string]string{"type": "local"}
	_, err := connSvc.CreateConnection(ctx, "delete-section-remote", "local", config, nil)
	require.NoError(t, err)

	t.Run("delete existing section", func(t *testing.T) {
		assert.True(t, storage.HasSection("delete-section-remote"))

		storage.DeleteSection("delete-section-remote")

		assert.False(t, storage.HasSection("delete-section-remote"))
	})

	t.Run("delete non-existing section does not error", func(t *testing.T) {
		// Should not panic
		storage.DeleteSection("non-existing-remote")
	})
}

func TestDBStorage_LoadSave(t *testing.T) {
	storage, _, _ := setupStorageTest(t)

	t.Run("Load always returns nil", func(t *testing.T) {
		err := storage.Load()
		assert.NoError(t, err)
	})

	t.Run("Save always returns nil", func(t *testing.T) {
		err := storage.Save()
		assert.NoError(t, err)
	})
}

func TestDBStorage_Serialize(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)
	ctx := context.Background()

	t.Run("empty database returns empty JSON", func(t *testing.T) {
		result, err := storage.Serialize()
		assert.NoError(t, err)
		assert.Equal(t, "{}", result)
	})

	t.Run("serializes all connections to JSON", func(t *testing.T) {
		_, err := connSvc.CreateConnection(ctx, "ser-remote", "s3", map[string]string{"type": "s3", "key": "value"}, nil)
		require.NoError(t, err)

		result, err := storage.Serialize()
		assert.NoError(t, err)
		assert.Contains(t, result, `"ser-remote"`)
		assert.Contains(t, result, `"type"`)
		assert.Contains(t, result, `"s3"`)
	})
}

// TestDBStorage_SetValue_UpdateType tests SetValue when updating the type field
func TestDBStorage_SetValue_UpdateType(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)
	ctx := context.Background()

	// Create initial connection
	_, err := connSvc.CreateConnection(ctx, "type-update-remote", "s3", map[string]string{
		"type": "s3",
		"key":  "value",
	}, nil)
	require.NoError(t, err)

	// Update the type field
	storage.SetValue("type-update-remote", "type", "alias")

	// Verify type was updated
	value, ok := storage.GetValue("type-update-remote", "type")
	assert.True(t, ok)
	assert.Equal(t, "alias", value)
}

// TestDBStorage_GetSectionList_Empty tests GetSectionList when query returns error
func TestDBStorage_GetSectionList_Empty(t *testing.T) {
	storage, _, _ := setupStorageTest(t)

	// Initially empty
	sections := storage.GetSectionList()
	assert.Empty(t, sections)
}

// TestDBStorage_Install tests the Install method
func TestDBStorage_Install(t *testing.T) {
	storage, _, _ := setupStorageTest(t)

	// Should not panic
	assert.NotPanics(t, func() {
		storage.Install()
	})
}

// createConnectionWithCacheOptions creates a connection with optional cache configuration
func createConnectionWithCacheOptions(t *testing.T, connSvc *query.ConnectionQuery, name string, cacheEnabled bool, infoAge, changeNotifyPoll *string) string {
	t.Helper()

	ctx := context.Background()

	// Create connection
	conn, err := connSvc.CreateConnection(ctx, name, "local", map[string]string{"type": "local"}, nil)
	require.NoError(t, err)

	// Update with cache options if enabled
	if cacheEnabled {
		opts := &model.ConnectionOptions{
			Cache: &model.ConnectionCacheOptions{
				Enabled:          true,
				InfoAge:          infoAge,
				ChangeNotifyPoll: changeNotifyPoll,
			},
		}
		_, err = conn.Update().SetOptions(opts).Save(ctx)
		require.NoError(t, err)
	}

	return conn.ID.String()
}

// TestDBStorage_HasSection_CacheSuffix tests HasSection with -cache suffix handling
func TestDBStorage_HasSection_CacheSuffix(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)

	// Create a connection with cache enabled
	createConnectionWithCacheOptions(t, connSvc, "myremote", true, nil, nil)

	// Create a connection without cache enabled
	createConnectionWithCacheOptions(t, connSvc, "nocache-remote", false, nil, nil)

	t.Run("cache suffix returns true when cache is enabled", func(t *testing.T) {
		// "myremote-cache" should exist because "myremote" has cache enabled
		assert.True(t, storage.HasSection("myremote-cache"))
	})

	t.Run("cache suffix returns false when cache is not enabled", func(t *testing.T) {
		// "nocache-remote-cache" should not exist because cache is not enabled
		assert.False(t, storage.HasSection("nocache-remote-cache"))
	})

	t.Run("cache suffix returns false when base connection does not exist", func(t *testing.T) {
		// "nonexistent-cache" should not exist
		assert.False(t, storage.HasSection("nonexistent-cache"))
	})

	t.Run("base connection still accessible", func(t *testing.T) {
		// "myremote" should still work normally
		assert.True(t, storage.HasSection("myremote"))
		assert.True(t, storage.HasSection("nocache-remote"))
	})
}

// TestDBStorage_HasSection_EdgeCase_NameEndingWithCache tests connections whose name ends with -cache
func TestDBStorage_HasSection_EdgeCase_NameEndingWithCache(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)

	// Create a connection whose name literally ends with "-cache"
	createConnectionWithCacheOptions(t, connSvc, "my-backup-cache", true, nil, nil)

	t.Run("connection name ending with -cache is found as-is", func(t *testing.T) {
		// Should find "my-backup-cache" as a real connection, not as cache for "my-backup"
		assert.True(t, storage.HasSection("my-backup-cache"))
	})

	t.Run("cache suffix for connection ending with -cache works", func(t *testing.T) {
		// "my-backup-cache-cache" should exist because "my-backup-cache" has cache enabled
		assert.True(t, storage.HasSection("my-backup-cache-cache"))
	})

	t.Run("nonexistent base connection returns false", func(t *testing.T) {
		// "my-backup" does not exist, so "my-backup-cache" should be found as real connection
		// but we should verify the logic works correctly
		assert.False(t, storage.HasSection("my-backup"))
	})
}

// TestDBStorage_GetValue_CacheSuffix tests GetValue with -cache suffix handling
func TestDBStorage_GetValue_CacheSuffix(t *testing.T) {
	storage, connSvc, dataDir := setupStorageTest(t)

	infoAge := "12h"
	changeNotifyPoll := "30s"

	// Create a connection with cache enabled and custom options
	connID := createConnectionWithCacheOptions(t, connSvc, "test-remote", true, &infoAge, &changeNotifyPoll)

	t.Run("cache suffix returns metacache type", func(t *testing.T) {
		value, ok := storage.GetValue("test-remote-cache", "type")
		assert.True(t, ok)
		assert.Equal(t, "metacache", value)
	})

	t.Run("cache suffix returns remote pointing to base connection", func(t *testing.T) {
		value, ok := storage.GetValue("test-remote-cache", "remote")
		assert.True(t, ok)
		assert.Equal(t, "test-remote:", value)
	})

	t.Run("cache suffix returns info_age from options", func(t *testing.T) {
		value, ok := storage.GetValue("test-remote-cache", "info_age")
		assert.True(t, ok)
		assert.Equal(t, "12h", value)
	})

	t.Run("cache suffix returns change_notify_poll from options", func(t *testing.T) {
		value, ok := storage.GetValue("test-remote-cache", "change_notify_poll")
		assert.True(t, ok)
		assert.Equal(t, "30s", value)
	})

	t.Run("cache suffix returns db_path", func(t *testing.T) {
		value, ok := storage.GetValue("test-remote-cache", "db_path")
		assert.True(t, ok)
		expectedPath := filepath.Join(dataDir, "cache", connID+".db")
		assert.Equal(t, expectedPath, value)
	})

	t.Run("cache suffix returns false for unknown key", func(t *testing.T) {
		value, ok := storage.GetValue("test-remote-cache", "unknown_key")
		assert.False(t, ok)
		assert.Empty(t, value)
	})

	t.Run("base connection values still accessible", func(t *testing.T) {
		value, ok := storage.GetValue("test-remote", "type")
		assert.True(t, ok)
		assert.Equal(t, "local", value)
	})
}

// TestDBStorage_GetValue_CacheSuffix_DefaultOptions tests GetValue when cache options use defaults
func TestDBStorage_GetValue_CacheSuffix_DefaultOptions(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)

	// Create a connection with cache enabled but no custom options (defaults)
	createConnectionWithCacheOptions(t, connSvc, "default-options", true, nil, nil)

	t.Run("info_age returns false when not set (use backend default)", func(t *testing.T) {
		_, ok := storage.GetValue("default-options-cache", "info_age")
		assert.False(t, ok)
	})

	t.Run("change_notify_poll returns false when not set (use backend default)", func(t *testing.T) {
		_, ok := storage.GetValue("default-options-cache", "change_notify_poll")
		assert.False(t, ok)
	})
}

// TestDBStorage_GetValue_CacheSuffix_NotEnabled tests GetValue when cache is not enabled
func TestDBStorage_GetValue_CacheSuffix_NotEnabled(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)

	// Create a connection without cache enabled
	createConnectionWithCacheOptions(t, connSvc, "no-cache", false, nil, nil)

	t.Run("cache suffix returns false when cache not enabled", func(t *testing.T) {
		_, ok := storage.GetValue("no-cache-cache", "type")
		assert.False(t, ok)
	})
}

// TestDBStorage_GetValue_EdgeCase_NameEndingWithCache tests connections whose name ends with -cache
func TestDBStorage_GetValue_EdgeCase_NameEndingWithCache(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)

	// Create a connection whose name literally ends with "-cache"
	createConnectionWithCacheOptions(t, connSvc, "my-data-cache", true, nil, nil)

	t.Run("connection name ending with -cache returns its own type", func(t *testing.T) {
		// Should return "local" (the real connection type), not "metacache"
		value, ok := storage.GetValue("my-data-cache", "type")
		assert.True(t, ok)
		assert.Equal(t, "local", value)
	})

	t.Run("cache suffix for connection ending with -cache returns metacache", func(t *testing.T) {
		// "my-data-cache-cache" should return metacache type
		value, ok := storage.GetValue("my-data-cache-cache", "type")
		assert.True(t, ok)
		assert.Equal(t, "metacache", value)
	})

	t.Run("cache suffix for connection ending with -cache returns correct remote", func(t *testing.T) {
		// "my-data-cache-cache" remote should point to "my-data-cache:"
		value, ok := storage.GetValue("my-data-cache-cache", "remote")
		assert.True(t, ok)
		assert.Equal(t, "my-data-cache:", value)
	})
}

// TestDBStorage_GetKeyList_CacheSuffix tests GetKeyList with -cache suffix handling
func TestDBStorage_GetKeyList_CacheSuffix(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)

	// Create a connection with cache enabled
	createConnectionWithCacheOptions(t, connSvc, "keylist-test", true, nil, nil)

	// Create a connection without cache enabled
	createConnectionWithCacheOptions(t, connSvc, "keylist-nocache", false, nil, nil)

	t.Run("cache suffix returns metacache keys when cache is enabled", func(t *testing.T) {
		keys := storage.GetKeyList("keylist-test-cache")
		assert.NotNil(t, keys)
		assert.Contains(t, keys, "type")
		assert.Contains(t, keys, "remote")
		assert.Contains(t, keys, "info_age")
		assert.Contains(t, keys, "change_notify_poll")
		assert.Contains(t, keys, "db_path")
	})

	t.Run("cache suffix returns nil when cache is not enabled", func(t *testing.T) {
		keys := storage.GetKeyList("keylist-nocache-cache")
		assert.Nil(t, keys)
	})

	t.Run("cache suffix returns nil when base connection does not exist", func(t *testing.T) {
		keys := storage.GetKeyList("nonexistent-cache")
		assert.Nil(t, keys)
	})

	t.Run("base connection returns its own keys", func(t *testing.T) {
		keys := storage.GetKeyList("keylist-test")
		assert.NotNil(t, keys)
		assert.Contains(t, keys, "type")
		// Should NOT contain metacache-specific keys
		assert.NotContains(t, keys, "remote")
		assert.NotContains(t, keys, "db_path")
	})
}

// TestDBStorage_GetKeyList_EdgeCase_NameEndingWithCache tests connections whose name ends with -cache
func TestDBStorage_GetKeyList_EdgeCase_NameEndingWithCache(t *testing.T) {
	storage, connSvc, _ := setupStorageTest(t)

	// Create a connection whose name literally ends with "-cache"
	createConnectionWithCacheOptions(t, connSvc, "backup-cache", true, nil, nil)

	t.Run("connection name ending with -cache returns its own keys", func(t *testing.T) {
		keys := storage.GetKeyList("backup-cache")
		assert.NotNil(t, keys)
		assert.Contains(t, keys, "type")
		// Should NOT contain metacache-specific keys for the base connection
		assert.NotContains(t, keys, "remote")
		assert.NotContains(t, keys, "db_path")
	})

	t.Run("cache suffix for connection ending with -cache returns metacache keys", func(t *testing.T) {
		keys := storage.GetKeyList("backup-cache-cache")
		assert.NotNil(t, keys)
		assert.Contains(t, keys, "type")
		assert.Contains(t, keys, "remote")
		assert.Contains(t, keys, "db_path")
	})
}
