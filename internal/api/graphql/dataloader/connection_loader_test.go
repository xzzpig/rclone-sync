package dataloader_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/dataloader"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// setupConnectionTestDB creates an in-memory database with connection service for testing.
func setupConnectionTestDB(t *testing.T) (*ent.Client, *services.ConnectionService) {
	t.Helper()

	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)

	client, err := db.InitDB(db.InitDBOptions{
		DSN:           "file:ent?mode=memory&cache=shared&_fk=1",
		MigrationMode: db.MigrationModeAuto,
	})
	require.NoError(t, err)

	encryptor, err := crypto.NewEncryptor("test-encryption-key-32-bytes!!")
	require.NoError(t, err)

	connectionService := services.NewConnectionService(client, encryptor)

	// Install DBStorage for rclone configuration
	storage := rclone.NewDBStorage(connectionService)
	storage.Install()

	t.Cleanup(func() {
		client.Close()
	})

	return client, connectionService
}

func TestConnectionLoader_Load_ExistingConnection(t *testing.T) {
	client, connectionService := setupConnectionTestDB(t)
	ctx := context.Background()

	// Create a test connection
	conn, err := connectionService.CreateConnection(ctx, "test-connection", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	// Create loader
	loader := dataloader.NewConnectionLoader(client)

	// Load the connection
	result, err := loader.Load(ctx, conn.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, conn.ID, result.ID)
	assert.Equal(t, "test-connection", result.Name)
}

func TestConnectionLoader_Load_NonExistentConnection(t *testing.T) {
	client, _ := setupConnectionTestDB(t)
	ctx := context.Background()

	// Create loader
	loader := dataloader.NewConnectionLoader(client)

	// Try to load a non-existent connection
	nonExistentID := uuid.New()
	result, err := loader.Load(ctx, nonExistentID)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection not found")
}

func TestConnectionLoader_LoadAll_MultipleConnections(t *testing.T) {
	client, connectionService := setupConnectionTestDB(t)
	ctx := context.Background()

	// Create multiple test connections
	conn1, err := connectionService.CreateConnection(ctx, "connection-1", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	conn2, err := connectionService.CreateConnection(ctx, "connection-2", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	conn3, err := connectionService.CreateConnection(ctx, "connection-3", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	// Create loader
	loader := dataloader.NewConnectionLoader(client)

	// Load all connections in batch
	ids := []uuid.UUID{conn1.ID, conn2.ID, conn3.ID}
	results, err := loader.LoadAll(ctx, ids)

	require.NoError(t, err)
	require.Len(t, results, 3)

	// Check results are in correct order
	assert.Equal(t, conn1.ID, results[0].ID)
	assert.Equal(t, conn2.ID, results[1].ID)
	assert.Equal(t, conn3.ID, results[2].ID)
}

func TestConnectionLoader_LoadAll_MixedExistingAndNonExistent(t *testing.T) {
	client, connectionService := setupConnectionTestDB(t)
	ctx := context.Background()

	// Create one test connection
	conn1, err := connectionService.CreateConnection(ctx, "connection-1", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	nonExistentID := uuid.New()

	// Create loader
	loader := dataloader.NewConnectionLoader(client)

	// Load batch with mix of existing and non-existing
	// When one fails, LoadAll returns an error
	ids := []uuid.UUID{conn1.ID, nonExistentID}
	results, err := loader.LoadAll(ctx, ids)

	// LoadAll returns error if any item fails
	assert.Error(t, err)

	// Results will still have the length matching input
	require.Len(t, results, 2)

	// First should be present
	assert.NotNil(t, results[0])
	assert.Equal(t, conn1.ID, results[0].ID)
}

func TestConnectionLoader_LoadAll_PreservesOrder(t *testing.T) {
	client, connectionService := setupConnectionTestDB(t)
	ctx := context.Background()

	// Create connections in specific order
	var connections []*ent.Connection
	for i := 0; i < 5; i++ {
		conn, err := connectionService.CreateConnection(ctx, "connection-"+string(rune('A'+i)), "local", map[string]string{
			"type": "local",
		})
		require.NoError(t, err)
		connections = append(connections, conn)
	}

	// Create loader
	loader := dataloader.NewConnectionLoader(client)

	// Request in reverse order
	ids := []uuid.UUID{
		connections[4].ID,
		connections[2].ID,
		connections[0].ID,
		connections[3].ID,
		connections[1].ID,
	}

	results, err := loader.LoadAll(ctx, ids)

	require.NoError(t, err)
	require.Len(t, results, 5)

	// Results should match request order
	assert.Equal(t, connections[4].ID, results[0].ID)
	assert.Equal(t, connections[2].ID, results[1].ID)
	assert.Equal(t, connections[0].ID, results[2].ID)
	assert.Equal(t, connections[3].ID, results[3].ID)
	assert.Equal(t, connections[1].ID, results[4].ID)
}

func TestConnectionLoader_LoadAll_EmptySlice(t *testing.T) {
	client, _ := setupConnectionTestDB(t)
	ctx := context.Background()

	// Create loader
	loader := dataloader.NewConnectionLoader(client)

	// Load with empty slice
	results, err := loader.LoadAll(ctx, []uuid.UUID{})

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestConnectionLoader_LoadAll_DuplicateIDs(t *testing.T) {
	client, connectionService := setupConnectionTestDB(t)
	ctx := context.Background()

	// Create a test connection
	conn, err := connectionService.CreateConnection(ctx, "test-connection", "local", map[string]string{
		"type": "local",
	})
	require.NoError(t, err)

	// Create loader
	loader := dataloader.NewConnectionLoader(client)

	// Load same ID multiple times
	ids := []uuid.UUID{conn.ID, conn.ID, conn.ID}
	results, err := loader.LoadAll(ctx, ids)

	require.NoError(t, err)
	require.Len(t, results, 3)

	// All should return the same connection
	for _, result := range results {
		assert.NotNil(t, result)
		assert.Equal(t, conn.ID, result.ID)
	}
}
