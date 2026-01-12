package rclone_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/db/query"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/rclone"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/rclone/rclone/backend/local"
	"github.com/rclone/rclone/fs/config"
)

// setupTestConfig initializes rclone configuration for testing using DBStorage
func setupTestConfig(t *testing.T) (*rclone.DBStorage, *query.ConnectionQuery) {
	t.Helper()

	// Create test database client
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	t.Cleanup(func() { client.Close() })

	// Create encryptor (plaintext mode for testing)
	encryptor, err := crypto.NewEncryptor("")
	require.NoError(t, err)

	// Create connection query
	connSvc := query.NewConnectionQuery(client, encryptor)

	// Create DBStorage and install it (use temp dir for cache)
	storage := rclone.NewDBStorage(connSvc, t.TempDir())
	storage.Install()

	return storage, connSvc
}

// createRemote creates or updates a remote with the given parameters.
func createRemote(remoteName string, params map[string]string) error {
	for key, value := range params {
		if value == "" {
			config.FileDeleteKey(remoteName, key)
			continue
		}
		config.FileSetValue(remoteName, key, value)
	}
	config.SaveConfig()
	return nil
}

// deleteRemote deletes a remote.
func deleteRemote(remoteName string) {
	config.DeleteRemote(remoteName)
}
