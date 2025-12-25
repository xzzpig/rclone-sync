package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/db"
)

// setupTestDB creates a test database and returns the client
func setupTestDB(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	return client
}

// setupTestEncryptor creates a test encryptor with a fixed key
func setupTestEncryptor(t *testing.T) *crypto.Encryptor {
	encryptor, err := crypto.NewEncryptor("test-encryption-key-32-bytes!!")
	require.NoError(t, err)
	return encryptor
}

// T015: 单元测试：ConnectionService.CreateConnection
func TestConnectionService_CreateConnection(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Test data
	config := map[string]string{
		"type":       "onedrive",
		"token":      `{"access_token":"test_token","refresh_token":"test_refresh"}`,
		"drive_id":   "abc123",
		"drive_type": "personal",
	}

	// Create connection
	conn, err := service.CreateConnection(ctx, "my-onedrive", "onedrive", config)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, "my-onedrive", conn.Name)
	assert.Equal(t, "onedrive", conn.Type)
	assert.NotEmpty(t, conn.ID)
	assert.NotEmpty(t, conn.EncryptedConfig)
	assert.NotZero(t, conn.CreatedAt)
	assert.NotZero(t, conn.UpdatedAt)

	// Verify connection can be retrieved
	retrieved, err := service.GetConnectionByName(ctx, "my-onedrive")
	require.NoError(t, err)
	assert.Equal(t, conn.ID, retrieved.ID)
	assert.Equal(t, "my-onedrive", retrieved.Name)
	assert.Equal(t, "onedrive", retrieved.Type)

	// Verify config is encrypted (can be decrypted)
	decryptedConfig, err := encryptor.DecryptConfig(retrieved.EncryptedConfig)
	require.NoError(t, err)
	assert.Equal(t, "onedrive", decryptedConfig["type"])
	assert.Contains(t, decryptedConfig["token"], "test_token")
}

// T016: 单元测试：重复名称创建失败
func TestConnectionService_CreateConnection_DuplicateName(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	config := map[string]string{
		"type": "s3",
	}

	// Create first connection
	_, err := service.CreateConnection(ctx, "duplicate-test", "s3", config)
	require.NoError(t, err)

	// Try to create second connection with same name
	_, err = service.CreateConnection(ctx, "duplicate-test", "s3", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestConnectionService_CreateConnection_InvalidName(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	config := map[string]string{
		"type": "s3",
	}

	tests := []struct {
		name        string
		connName    string
		expectError bool
	}{
		// Invalid names according to rclone's CheckConfigName
		{"empty name", "", true},
		{"starts with hyphen", "-test", true},
		{"starts with space", " test", true},
		{"ends with space", "test ", true},
		{"only invalid chars", "!!!", true},
		{"contains invalid char", "my#test", true},

		// Valid names according to rclone's CheckConfigName
		{"starts with number", "123test", false},
		{"contains space in middle", "my test", false},
		{"contains @ symbol", "my@test", false},
		{"contains + symbol", "my+test", false},
		{"contains dot", "my.test", false},
		{"valid name with underscore", "my_test", false},
		{"valid name with hyphen", "my-test", false},
		{"valid name alphanumeric", "myTest123", false},
		{"very long name", "this_is_a_very_long_connection_name_that_exceeds_the_previous_64_character_limit_but_is_now_allowed", false},
		{"unicode name", "测试连接", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateConnection(ctx, tt.connName, "s3", config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConnectionService_CreateConnection_EmptyConfig(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Empty config should be allowed (some providers have minimal config)
	conn, err := service.CreateConnection(ctx, "minimal-conn", "local", map[string]string{})
	require.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestConnectionService_CreateConnection_PlaintextMode(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	// Create encryptor with empty key (plaintext mode)
	encryptor, err := crypto.NewEncryptor("")
	require.NoError(t, err)

	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	config := map[string]string{
		"type":  "s3",
		"token": "sensitive_data",
	}

	conn, err := service.CreateConnection(ctx, "plaintext-conn", "s3", config)
	require.NoError(t, err)

	// In plaintext mode, config should still be retrievable
	decryptedConfig, err := encryptor.DecryptConfig(conn.EncryptedConfig)
	require.NoError(t, err)
	assert.Equal(t, "sensitive_data", decryptedConfig["token"])
}

// T023: 单元测试：ConnectionService.ListConnections
func TestConnectionService_ListConnections(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Initially empty
	connections, err := service.ListConnections(ctx)
	require.NoError(t, err)
	assert.Empty(t, connections)

	// Create multiple connections
	config1 := map[string]string{"type": "s3", "region": "us-east-1"}
	config2 := map[string]string{"type": "onedrive", "drive_type": "personal"}
	config3 := map[string]string{"type": "dropbox"}

	conn1, err := service.CreateConnection(ctx, "my-s3", "s3", config1)
	require.NoError(t, err)

	conn2, err := service.CreateConnection(ctx, "my-onedrive", "onedrive", config2)
	require.NoError(t, err)

	conn3, err := service.CreateConnection(ctx, "my-dropbox", "dropbox", config3)
	require.NoError(t, err)

	// List all connections
	connections, err = service.ListConnections(ctx)
	require.NoError(t, err)
	assert.Len(t, connections, 3)

	// Verify connections are returned (order may vary)
	names := make(map[string]bool)
	types := make(map[string]string)
	for _, conn := range connections {
		names[conn.Name] = true
		types[conn.Name] = conn.Type
		assert.NotEmpty(t, conn.ID)
		assert.NotEmpty(t, conn.EncryptedConfig)
		assert.NotZero(t, conn.CreatedAt)
		assert.NotZero(t, conn.UpdatedAt)
	}

	assert.True(t, names["my-s3"])
	assert.True(t, names["my-onedrive"])
	assert.True(t, names["my-dropbox"])
	assert.Equal(t, "s3", types["my-s3"])
	assert.Equal(t, "onedrive", types["my-onedrive"])
	assert.Equal(t, "dropbox", types["my-dropbox"])

	// Verify IDs match
	assert.Contains(t, []string{conn1.ID.String(), conn2.ID.String(), conn3.ID.String()}, connections[0].ID.String())
}

// 单元测试：ConnectionService.ListConnectionNames (优化查询)
func TestConnectionService_ListConnectionNames(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Initially empty
	names, err := service.ListConnectionNames(ctx)
	require.NoError(t, err)
	assert.Empty(t, names)

	// Create multiple connections
	config1 := map[string]string{"type": "s3", "region": "us-east-1"}
	config2 := map[string]string{"type": "onedrive", "drive_type": "personal"}
	config3 := map[string]string{"type": "dropbox"}

	_, err = service.CreateConnection(ctx, "my-s3", "s3", config1)
	require.NoError(t, err)

	_, err = service.CreateConnection(ctx, "my-onedrive", "onedrive", config2)
	require.NoError(t, err)

	_, err = service.CreateConnection(ctx, "my-dropbox", "dropbox", config3)
	require.NoError(t, err)

	// List connection names
	names, err = service.ListConnectionNames(ctx)
	require.NoError(t, err)
	assert.Len(t, names, 3)

	// Verify only names are returned (sorted alphabetically)
	assert.Equal(t, []string{"my-dropbox", "my-onedrive", "my-s3"}, names)
}

// T024: 单元测试：ConnectionService.GetConnectionByName
func TestConnectionService_GetConnectionByName(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create a connection
	config := map[string]string{
		"type":       "onedrive",
		"token":      `{"access_token":"test_token"}`,
		"drive_type": "personal",
	}

	created, err := service.CreateConnection(ctx, "test-connection", "onedrive", config)
	require.NoError(t, err)

	// Get by name
	conn, err := service.GetConnectionByName(ctx, "test-connection")
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, created.ID, conn.ID)
	assert.Equal(t, "test-connection", conn.Name)
	assert.Equal(t, "onedrive", conn.Type)
	assert.NotEmpty(t, conn.EncryptedConfig)
	assert.NotZero(t, conn.CreatedAt)
	assert.NotZero(t, conn.UpdatedAt)

	// Verify config is still encrypted
	assert.NotContains(t, string(conn.EncryptedConfig), "test_token")

	// Get non-existent connection
	_, err = service.GetConnectionByName(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConnectionService_GetConnectionByName_CaseSensitive(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create connection with specific case
	config := map[string]string{"type": "s3"}
	_, err := service.CreateConnection(ctx, "MyConnection", "s3", config)
	require.NoError(t, err)

	// Should find exact match
	conn, err := service.GetConnectionByName(ctx, "MyConnection")
	require.NoError(t, err)
	assert.Equal(t, "MyConnection", conn.Name)

	// Should not find different case
	_, err = service.GetConnectionByName(ctx, "myconnection")
	assert.Error(t, err)

	_, err = service.GetConnectionByName(ctx, "MYCONNECTION")
	assert.Error(t, err)
}

// T033: 单元测试：ConnectionService.UpdateConnection
func TestConnectionService_UpdateConnection(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create initial connection
	initialConfig := map[string]string{
		"type":       "s3",
		"region":     "us-east-1",
		"access_key": "old_access_key",
		"secret_key": "old_secret_key",
	}

	conn, err := service.CreateConnection(ctx, "my-s3", "s3", initialConfig)
	require.NoError(t, err)
	initialUpdatedAt := conn.UpdatedAt

	// Update configuration
	updatedConfig := map[string]string{
		"type":       "s3",
		"region":     "us-west-2",      // Changed region
		"access_key": "new_access_key", // Changed access key
		"secret_key": "new_secret_key", // Changed secret key
		"bucket":     "my-bucket",      // Added new field
	}

	err = service.UpdateConnection(ctx, conn.ID, nil, nil, updatedConfig)
	require.NoError(t, err)

	// Retrieve updated connection to verify
	updated, err := service.GetConnectionByName(ctx, "my-s3")
	require.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, conn.ID, updated.ID)
	assert.Equal(t, "my-s3", updated.Name)
	assert.Equal(t, "s3", updated.Type) // Type should not change
	assert.NotEmpty(t, updated.EncryptedConfig)

	// UpdatedAt should change
	assert.True(t, updated.UpdatedAt.After(initialUpdatedAt))

	// Verify new config is properly encrypted and stored
	decryptedConfig, err := encryptor.DecryptConfig(updated.EncryptedConfig)
	require.NoError(t, err)
	assert.Equal(t, "us-west-2", decryptedConfig["region"])
	assert.Equal(t, "new_access_key", decryptedConfig["access_key"])
	assert.Equal(t, "new_secret_key", decryptedConfig["secret_key"])
	assert.Equal(t, "my-bucket", decryptedConfig["bucket"])

	// Verify old values are replaced
	assert.NotEqual(t, "us-east-1", decryptedConfig["region"])
	assert.NotEqual(t, "old_access_key", decryptedConfig["access_key"])

	// Retrieve and verify persistence
	retrieved, err := service.GetConnectionByName(ctx, "my-s3")
	require.NoError(t, err)

	retrievedConfig, err := encryptor.DecryptConfig(retrieved.EncryptedConfig)
	require.NoError(t, err)
	assert.Equal(t, "us-west-2", retrievedConfig["region"])
	assert.Equal(t, "new_access_key", retrievedConfig["access_key"])
	assert.Equal(t, "my-bucket", retrievedConfig["bucket"])
}

func TestConnectionService_UpdateConnection_NonExistent(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Try to update non-existent connection with random UUID
	config := map[string]string{"type": "s3"}
	fakeID := uuid.New()
	err := service.UpdateConnection(ctx, fakeID, nil, nil, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConnectionService_UpdateConnection_EmptyConfig(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create connection
	initialConfig := map[string]string{
		"type": "local",
		"path": "/data",
	}

	conn, err := service.CreateConnection(ctx, "my-local", "local", initialConfig)
	require.NoError(t, err)

	// Update with empty config (should be allowed for some providers)
	emptyConfig := map[string]string{}
	err = service.UpdateConnection(ctx, conn.ID, nil, nil, emptyConfig)
	require.NoError(t, err)

	// Retrieve and verify config is empty
	updated, err := service.GetConnectionByName(ctx, "my-local")
	require.NoError(t, err)
	decryptedConfig, err := encryptor.DecryptConfig(updated.EncryptedConfig)
	require.NoError(t, err)
	delete(decryptedConfig, "type") // Remove type for comparison
	assert.Empty(t, decryptedConfig)
}

func TestConnectionService_UpdateConnection_PartialUpdate(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create connection with multiple fields
	initialConfig := map[string]string{
		"type":       "onedrive",
		"token":      `{"access_token":"token1"}`,
		"drive_id":   "drive123",
		"drive_type": "personal",
	}

	conn, err := service.CreateConnection(ctx, "my-onedrive", "onedrive", initialConfig)
	require.NoError(t, err)

	// Update only token field (note: this is a full config replacement, not partial merge)
	updatedConfig := map[string]string{
		"type":  "onedrive",
		"token": `{"access_token":"token2"}`,
	}

	err = service.UpdateConnection(ctx, conn.ID, nil, nil, updatedConfig)
	require.NoError(t, err)

	// Retrieve and verify new config completely replaces old config
	updated, err := service.GetConnectionByName(ctx, "my-onedrive")
	require.NoError(t, err)
	decryptedConfig, err := encryptor.DecryptConfig(updated.EncryptedConfig)
	require.NoError(t, err)
	assert.Contains(t, decryptedConfig["token"], "token2")

	// Old fields should not exist (full replacement)
	assert.NotContains(t, decryptedConfig, "drive_id")
	assert.NotContains(t, decryptedConfig, "drive_type")
}

func TestConnectionService_UpdateConnection_PlaintextMode(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	// Plaintext encryptor
	encryptor, err := crypto.NewEncryptor("")
	require.NoError(t, err)

	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create and update in plaintext mode
	initialConfig := map[string]string{
		"type":  "s3",
		"token": "old_token",
	}

	conn, err := service.CreateConnection(ctx, "plaintext-s3", "s3", initialConfig)
	require.NoError(t, err)

	updatedConfig := map[string]string{
		"type":  "s3",
		"token": "new_token",
	}

	err = service.UpdateConnection(ctx, conn.ID, nil, nil, updatedConfig)
	require.NoError(t, err)

	// Retrieve and verify update worked in plaintext mode
	updated, err := service.GetConnectionByName(ctx, "plaintext-s3")
	require.NoError(t, err)
	decryptedConfig, err := encryptor.DecryptConfig(updated.EncryptedConfig)
	require.NoError(t, err)
	assert.Equal(t, "new_token", decryptedConfig["token"])
}

// T038: 单元测试：ConnectionService.DeleteConnectionByName
func TestConnectionService_DeleteConnectionByName(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create a connection
	config := map[string]string{
		"type":  "s3",
		"token": "test_token",
	}

	conn, err := service.CreateConnection(ctx, "test-delete", "s3", config)
	require.NoError(t, err)

	// Verify connection exists
	retrieved, err := service.GetConnectionByName(ctx, "test-delete")
	require.NoError(t, err)
	assert.Equal(t, conn.ID, retrieved.ID)

	// Delete the connection
	err = service.DeleteConnectionByName(ctx, "test-delete")
	require.NoError(t, err)

	// Verify connection no longer exists
	_, err = service.GetConnectionByName(ctx, "test-delete")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// List connections should be empty
	connections, err := service.ListConnections(ctx)
	require.NoError(t, err)
	assert.Empty(t, connections)
}

func TestConnectionService_DeleteConnectionByName_NonExistent(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Try to delete non-existent connection
	err := service.DeleteConnectionByName(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// T039: 单元测试：级联删除关联 Task
func TestConnectionService_DeleteConnection_CascadeDeleteTasks(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	connService := NewConnectionService(client, encryptor)
	taskService := NewTaskService(client)

	ctx := context.Background()

	// Create a connection
	config := map[string]string{
		"type": "s3",
	}

	conn, err := connService.CreateConnection(ctx, "test-cascade", "s3", config)
	require.NoError(t, err)

	// Create multiple tasks associated with this connection
	task1, err := taskService.CreateTask(ctx, "task1", "/source1", conn.ID, "/dest1", string(model.SyncDirectionBidirectional), "0 0 * * *", false, nil)
	require.NoError(t, err)

	task2, err := taskService.CreateTask(ctx, "task2", "/source2", conn.ID, "/dest2", string(model.SyncDirectionUpload), "0 1 * * *", false, nil)
	require.NoError(t, err)

	task3, err := taskService.CreateTask(ctx, "task3", "/source3", conn.ID, "/dest3", string(model.SyncDirectionDownload), "0 2 * * *", false, nil)
	require.NoError(t, err)

	// Verify tasks exist
	tasks, err := taskService.ListAllTasks(ctx)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Delete the connection (should cascade delete all tasks)
	err = connService.DeleteConnectionByName(ctx, "test-cascade")
	require.NoError(t, err)

	// Verify connection is deleted
	_, err = connService.GetConnectionByName(ctx, "test-cascade")
	assert.Error(t, err)

	// Verify all associated tasks are deleted
	tasks, err = taskService.ListAllTasks(ctx)
	require.NoError(t, err)
	assert.Empty(t, tasks)

	// Verify specific tasks no longer exist
	_, err = taskService.GetTask(ctx, task1.ID)
	assert.Error(t, err)

	_, err = taskService.GetTask(ctx, task2.ID)
	assert.Error(t, err)

	_, err = taskService.GetTask(ctx, task3.ID)
	assert.Error(t, err)
}

func TestConnectionService_HasAssociatedTasks(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	connService := NewConnectionService(client, encryptor)
	taskService := NewTaskService(client)

	ctx := context.Background()

	// Create a connection
	config := map[string]string{"type": "s3"}
	conn, err := connService.CreateConnection(ctx, "test-has-tasks", "s3", config)
	require.NoError(t, err)

	// Initially should have no tasks
	hasTasks, err := connService.HasAssociatedTasks(ctx, conn.ID)
	require.NoError(t, err)
	assert.False(t, hasTasks)

	// Create a task
	_, err = taskService.CreateTask(ctx, "task1", "/source", conn.ID, "/dest", string(model.SyncDirectionBidirectional), "0 0 * * *", false, nil)
	require.NoError(t, err)

	// Now should have tasks
	hasTasks, err = connService.HasAssociatedTasks(ctx, conn.ID)
	require.NoError(t, err)
	assert.True(t, hasTasks)
}

func TestConnectionService_DeleteConnection_MultipleConnections(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	connService := NewConnectionService(client, encryptor)
	taskService := NewTaskService(client)

	ctx := context.Background()

	// Create multiple connections
	config := map[string]string{"type": "s3"}
	conn1, err := connService.CreateConnection(ctx, "conn1", "s3", config)
	require.NoError(t, err)

	conn2, err := connService.CreateConnection(ctx, "conn2", "s3", config)
	require.NoError(t, err)

	// Create tasks for each connection
	task1, err := taskService.CreateTask(ctx, "task1", "/s1", conn1.ID, "/d1", string(model.SyncDirectionBidirectional), "0 0 * * *", false, nil)
	require.NoError(t, err)

	task2, err := taskService.CreateTask(ctx, "task2", "/s2", conn2.ID, "/d2", string(model.SyncDirectionBidirectional), "0 0 * * *", false, nil)
	require.NoError(t, err)

	// Delete conn1
	err = connService.DeleteConnectionByName(ctx, "conn1")
	require.NoError(t, err)

	// Verify conn1 and task1 are deleted
	_, err = connService.GetConnectionByName(ctx, "conn1")
	assert.Error(t, err)

	_, err = taskService.GetTask(ctx, task1.ID)
	assert.Error(t, err)

	// Verify conn2 and task2 still exist
	conn2Retrieved, err := connService.GetConnectionByName(ctx, "conn2")
	require.NoError(t, err)
	assert.Equal(t, conn2.ID, conn2Retrieved.ID)

	task2Retrieved, err := taskService.GetTask(ctx, task2.ID)
	require.NoError(t, err)
	assert.Equal(t, task2.ID, task2Retrieved.ID)
}

// Additional tests for GetConnectionByID
func TestConnectionService_GetConnectionByID(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create a connection
	config := map[string]string{
		"type":  "s3",
		"token": "test_token",
	}

	created, err := service.CreateConnection(ctx, "test-by-id", "s3", config)
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		conn, err := service.GetConnectionByID(ctx, created.ID)
		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Equal(t, created.ID, conn.ID)
		assert.Equal(t, "test-by-id", conn.Name)
		assert.Equal(t, "s3", conn.Type)
		assert.NotEmpty(t, conn.EncryptedConfig)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := service.GetConnectionByID(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// Tests for GetConnectionConfig
func TestConnectionService_GetConnectionConfig(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create a connection with config
	config := map[string]string{
		"type":       "onedrive",
		"token":      `{"access_token":"secret_token"}`,
		"drive_type": "personal",
	}

	_, err := service.CreateConnection(ctx, "test-get-config", "onedrive", config)
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		decrypted, err := service.GetConnectionConfig(ctx, "test-get-config")
		require.NoError(t, err)
		assert.Equal(t, "onedrive", decrypted["type"])
		assert.Contains(t, decrypted["token"], "secret_token")
		assert.Equal(t, "personal", decrypted["drive_type"])
	})

	t.Run("ConnectionNotFound", func(t *testing.T) {
		_, err := service.GetConnectionConfig(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// Tests for GetConnectionConfigByID
func TestConnectionService_GetConnectionConfigByID(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create a connection with config
	config := map[string]string{
		"type":   "dropbox",
		"token":  "secret_dropbox_token",
		"folder": "/sync",
	}

	created, err := service.CreateConnection(ctx, "test-get-config-by-id", "dropbox", config)
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		decrypted, err := service.GetConnectionConfigByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "dropbox", decrypted["type"])
		assert.Equal(t, "secret_dropbox_token", decrypted["token"])
		assert.Equal(t, "/sync", decrypted["folder"])
	})

	t.Run("ConnectionNotFound", func(t *testing.T) {
		_, err := service.GetConnectionConfigByID(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// Tests for DeleteConnectionByID
func TestConnectionService_DeleteConnectionByID(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create a connection
	config := map[string]string{"type": "local"}

	conn, err := service.CreateConnection(ctx, "test-delete-by-id", "local", config)
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		// Verify connection exists
		retrieved, err := service.GetConnectionByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, conn.ID, retrieved.ID)

		// Delete the connection
		err = service.DeleteConnectionByID(ctx, conn.ID)
		require.NoError(t, err)

		// Verify connection no longer exists
		_, err = service.GetConnectionByID(ctx, conn.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("NotFound", func(t *testing.T) {
		// Try to delete non-existent connection
		err := service.DeleteConnectionByID(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// Tests for UpdateConnection with name and type changes
func TestConnectionService_UpdateConnection_NameChange(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create initial connection
	initialConfig := map[string]string{
		"type":   "s3",
		"region": "us-east-1",
	}

	conn, err := service.CreateConnection(ctx, "old-name", "s3", initialConfig)
	require.NoError(t, err)

	t.Run("SuccessfulNameChange", func(t *testing.T) {
		newName := "new-name"
		updatedConfig := map[string]string{
			"type":   "s3",
			"region": "us-east-1",
		}

		err = service.UpdateConnection(ctx, conn.ID, &newName, nil, updatedConfig)
		require.NoError(t, err)

		// Verify name changed
		updated, err := service.GetConnectionByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "new-name", updated.Name)

		// Old name should not exist
		_, err = service.GetConnectionByName(ctx, "old-name")
		assert.Error(t, err)

		// New name should exist
		byName, err := service.GetConnectionByName(ctx, "new-name")
		require.NoError(t, err)
		assert.Equal(t, conn.ID, byName.ID)
	})

	t.Run("NameConflict", func(t *testing.T) {
		// Create another connection
		config2 := map[string]string{"type": "local"}
		conn2, err := service.CreateConnection(ctx, "existing-name", "local", config2)
		require.NoError(t, err)

		// Try to rename to existing name
		existingName := "existing-name"
		err = service.UpdateConnection(ctx, conn.ID, &existingName, nil, initialConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")

		// Verify conn2 still exists
		_, err = service.GetConnectionByID(ctx, conn2.ID)
		require.NoError(t, err)
	})
}

func TestConnectionService_UpdateConnection_TypeChange(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Create initial connection
	initialConfig := map[string]string{
		"type": "s3",
	}

	conn, err := service.CreateConnection(ctx, "type-change-test", "s3", initialConfig)
	require.NoError(t, err)

	// Update type
	newType := "onedrive"
	updatedConfig := map[string]string{
		"type": "onedrive",
	}

	err = service.UpdateConnection(ctx, conn.ID, nil, &newType, updatedConfig)
	require.NoError(t, err)

	// Verify type changed
	updated, err := service.GetConnectionByID(ctx, conn.ID)
	require.NoError(t, err)
	assert.Equal(t, "onedrive", updated.Type)
}

// Test ValidateConnectionName directly
func TestValidateConnectionName(t *testing.T) {
	tests := []struct {
		name        string
		connName    string
		expectError bool
	}{
		{"empty name", "", true},
		{"valid simple name", "myconn", false},
		{"valid with hyphen", "my-conn", false},
		{"valid with underscore", "my_conn", false},
		{"valid with number", "conn123", false},
		{"valid with space", "my conn", false},
		{"valid with dot", "my.conn", false},
		{"starts with hyphen", "-myconn", true},
		{"contains invalid char", "my#conn", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConnectionName(tt.connName)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test HasAssociatedTasks with non-existent connection
func TestConnectionService_HasAssociatedTasks_NotFound(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Try to check tasks for non-existent connection
	_, err := service.HasAssociatedTasks(ctx, uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Tests for ListConnectionsPaginated
func TestConnectionService_ListConnectionsPaginated(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	// Initially empty
	conns, total, err := service.ListConnectionsPaginated(ctx, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, conns)
	assert.Equal(t, 0, total)

	// Create multiple connections
	for i := 0; i < 5; i++ {
		config := map[string]string{"type": "local"}
		_, err := service.CreateConnection(ctx, "conn-paginated-"+uuid.NewString()[:8], "local", config)
		require.NoError(t, err)
	}

	t.Run("FirstPage", func(t *testing.T) {
		conns, total, err := service.ListConnectionsPaginated(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, conns, 2)
		assert.Equal(t, 5, total)
	})

	t.Run("SecondPage", func(t *testing.T) {
		conns, total, err := service.ListConnectionsPaginated(ctx, 2, 2)
		require.NoError(t, err)
		assert.Len(t, conns, 2)
		assert.Equal(t, 5, total)
	})

	t.Run("LastPage", func(t *testing.T) {
		conns, total, err := service.ListConnectionsPaginated(ctx, 2, 4)
		require.NoError(t, err)
		assert.Len(t, conns, 1)
		assert.Equal(t, 5, total)
	})

	t.Run("OffsetBeyondTotal", func(t *testing.T) {
		conns, total, err := service.ListConnectionsPaginated(ctx, 10, 100)
		require.NoError(t, err)
		assert.Empty(t, conns)
		assert.Equal(t, 5, total)
	})

	t.Run("LargeLimit", func(t *testing.T) {
		conns, total, err := service.ListConnectionsPaginated(ctx, 100, 0)
		require.NoError(t, err)
		assert.Len(t, conns, 5)
		assert.Equal(t, 5, total)
	})
}

// Tests for CountAssociatedTasks
func TestConnectionService_CountAssociatedTasks(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	connService := NewConnectionService(client, encryptor)
	taskService := NewTaskService(client)

	ctx := context.Background()

	// Create a connection
	config := map[string]string{"type": "local"}
	conn, err := connService.CreateConnection(ctx, "count-tasks-conn", "local", config)
	require.NoError(t, err)

	t.Run("ZeroTasks", func(t *testing.T) {
		count, err := connService.CountAssociatedTasks(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("OneTasks", func(t *testing.T) {
		// Create one task
		_, err := taskService.CreateTask(ctx, "task1", "/src", conn.ID, "/dst", string(model.SyncDirectionBidirectional), "", false, nil)
		require.NoError(t, err)

		count, err := connService.CountAssociatedTasks(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("MultipleTasks", func(t *testing.T) {
		// Create more tasks
		_, err := taskService.CreateTask(ctx, "task2", "/src2", conn.ID, "/dst2", string(model.SyncDirectionUpload), "", false, nil)
		require.NoError(t, err)
		_, err = taskService.CreateTask(ctx, "task3", "/src3", conn.ID, "/dst3", string(model.SyncDirectionDownload), "", false, nil)
		require.NoError(t, err)

		count, err := connService.CountAssociatedTasks(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := connService.CountAssociatedTasks(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// Test CreateConnection with empty type
func TestConnectionService_CreateConnection_EmptyType(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	encryptor := setupTestEncryptor(t)
	service := NewConnectionService(client, encryptor)

	ctx := context.Background()

	config := map[string]string{"key": "value"}
	_, err := service.CreateConnection(ctx, "valid-name", "", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "type cannot be empty")
}
