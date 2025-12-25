package rclone

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/services"

	_ "github.com/mattn/go-sqlite3"
)

func setupStorageTest(t *testing.T) (*DBStorage, *services.ConnectionService) {
	t.Helper()

	// Create test database client
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	t.Cleanup(func() { client.Close() })

	// Create encryptor (plaintext mode for testing)
	encryptor, err := crypto.NewEncryptor("")
	require.NoError(t, err)

	// Create connection service
	connSvc := services.NewConnectionService(client, encryptor)

	// Create DBStorage
	storage := NewDBStorage(connSvc)

	return storage, connSvc
}

// T045: 单元测试：DBStorage.GetValue
func TestDBStorage_GetValue(t *testing.T) {
	storage, connSvc := setupStorageTest(t)
	ctx := context.Background()

	// Create a test connection
	config := map[string]string{
		"type":     "onedrive",
		"token":    `{"access_token":"test-token"}`,
		"drive_id": "abc123",
	}
	_, err := connSvc.CreateConnection(ctx, "test-remote", "onedrive", config)
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
	storage, connSvc := setupStorageTest(t)
	ctx := context.Background()

	t.Run("set value on existing connection", func(t *testing.T) {
		// Create a test connection first
		config := map[string]string{
			"type":  "s3",
			"token": "old-token",
		}
		_, err := connSvc.CreateConnection(ctx, "update-remote", "s3", config)
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
		_, err := connSvc.CreateConnection(ctx, "oauth-remote", "onedrive", config)
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
	storage, connSvc := setupStorageTest(t)
	ctx := context.Background()

	// Create a test connection
	config := map[string]string{"type": "local"}
	_, err := connSvc.CreateConnection(ctx, "existing-remote", "local", config)
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
	storage, connSvc := setupStorageTest(t)
	ctx := context.Background()

	t.Run("empty database returns empty list", func(t *testing.T) {
		sections := storage.GetSectionList()
		assert.Empty(t, sections)
	})

	t.Run("returns all connection names", func(t *testing.T) {
		// Create multiple connections
		_, err := connSvc.CreateConnection(ctx, "remote-a", "s3", map[string]string{"type": "s3"})
		require.NoError(t, err)
		_, err = connSvc.CreateConnection(ctx, "remote-b", "gdrive", map[string]string{"type": "gdrive"})
		require.NoError(t, err)
		_, err = connSvc.CreateConnection(ctx, "remote-c", "onedrive", map[string]string{"type": "onedrive"})
		require.NoError(t, err)

		sections := storage.GetSectionList()
		assert.Len(t, sections, 3)
		assert.Contains(t, sections, "remote-a")
		assert.Contains(t, sections, "remote-b")
		assert.Contains(t, sections, "remote-c")
	})
}

func TestDBStorage_GetKeyList(t *testing.T) {
	storage, connSvc := setupStorageTest(t)
	ctx := context.Background()

	config := map[string]string{
		"type":     "onedrive",
		"token":    "xxx",
		"drive_id": "abc",
	}
	_, err := connSvc.CreateConnection(ctx, "keylist-remote", "onedrive", config)
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
	storage, connSvc := setupStorageTest(t)
	ctx := context.Background()

	config := map[string]string{
		"type":       "s3",
		"key_id":     "xxx",
		"access_key": "yyy",
	}
	_, err := connSvc.CreateConnection(ctx, "delete-key-remote", "s3", config)
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
	storage, connSvc := setupStorageTest(t)
	ctx := context.Background()

	config := map[string]string{"type": "local"}
	_, err := connSvc.CreateConnection(ctx, "delete-section-remote", "local", config)
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
	storage, _ := setupStorageTest(t)

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
	storage, connSvc := setupStorageTest(t)
	ctx := context.Background()

	t.Run("empty database returns empty JSON", func(t *testing.T) {
		result, err := storage.Serialize()
		assert.NoError(t, err)
		assert.Equal(t, "{}", result)
	})

	t.Run("serializes all connections to JSON", func(t *testing.T) {
		_, err := connSvc.CreateConnection(ctx, "ser-remote", "s3", map[string]string{"type": "s3", "key": "value"})
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
	storage, connSvc := setupStorageTest(t)
	ctx := context.Background()

	// Create initial connection
	_, err := connSvc.CreateConnection(ctx, "type-update-remote", "s3", map[string]string{
		"type": "s3",
		"key":  "value",
	})
	require.NoError(t, err)

	// Update the type field
	storage.SetValue("type-update-remote", "type", "alias")

	// Verify type was updated
	value, ok := storage.GetValue("type-update-remote", "type")
	assert.True(t, ok)
	assert.Equal(t, "alias", value)
}

// TestDBStorage_GetSectionList_Empty tests GetSectionList when service returns error
func TestDBStorage_GetSectionList_Empty(t *testing.T) {
	storage, _ := setupStorageTest(t)

	// Initially empty
	sections := storage.GetSectionList()
	assert.Empty(t, sections)
}

// TestDBStorage_Install tests the Install method
func TestDBStorage_Install(t *testing.T) {
	storage, _ := setupStorageTest(t)

	// Should not panic
	assert.NotPanics(t, func() {
		storage.Install()
	})
}
