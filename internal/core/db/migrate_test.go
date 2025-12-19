package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
)

func init() {
	// Initialize config and logger for tests
	config.Cfg.App.Environment = "test"
	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)
}

func createTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "migrate_test_*.db")
	require.NoError(t, err)
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Open database connection
	db, err := sql.Open("sqlite3", tmpPath+"?_fk=1")
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.Remove(tmpPath)
	}

	return db, cleanup
}

func TestMigrate_FreshDatabase(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Execute migrations on a fresh database
	err := Migrate(db)
	require.NoError(t, err)

	// Verify tables were created
	tables := []string{"connections", "tasks", "jobs", "job_logs", "schema_migrations"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		assert.NoError(t, err, "Table %s should exist", table)
		assert.Equal(t, table, name)
	}
}

func TestMigrate_NoChange(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// First migration
	err := Migrate(db)
	require.NoError(t, err)

	// Second migration should report no change
	err = Migrate(db)
	require.NoError(t, err)
}

func TestGetMigrationStatus_FreshDatabase(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Check status before any migrations
	status, err := GetMigrationStatus(db)
	require.NoError(t, err)
	assert.Equal(t, uint(0), status.Version)
	assert.False(t, status.Dirty)
}

func TestGetMigrationStatus_AfterMigration(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Execute migrations
	err := Migrate(db)
	require.NoError(t, err)

	// Check status after migration
	status, err := GetMigrationStatus(db)
	require.NoError(t, err)
	assert.True(t, status.Version > 0, "Version should be greater than 0 after migration")
	assert.False(t, status.Dirty)
}

func TestGetPendingMigrations_FreshDatabase(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Check pending migrations before any migrations
	pending, err := GetPendingMigrations(db)
	require.NoError(t, err)
	assert.True(t, len(pending) > 0, "Should have pending migrations on fresh database")
}

func TestGetPendingMigrations_AfterMigration(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Execute migrations
	err := Migrate(db)
	require.NoError(t, err)

	// Check pending migrations after migration
	pending, err := GetPendingMigrations(db)
	require.NoError(t, err)
	assert.Equal(t, 0, len(pending), "Should have no pending migrations after migration")
}

func TestMigrate_DirtyDatabase(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Execute migrations first
	err := Migrate(db)
	require.NoError(t, err)

	// Simulate dirty state
	_, err = db.Exec("UPDATE schema_migrations SET dirty = 1")
	require.NoError(t, err)

	// Try to migrate again - should fail due to dirty state
	err = Migrate(db)
	assert.Error(t, err, "Migration should fail on dirty database")
}

func TestLogMigrationStatus(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Execute migrations
	err := Migrate(db)
	require.NoError(t, err)

	// This should not panic
	LogMigrationStatus(db)
}

func TestParseMigrationMode(t *testing.T) {
	tests := []struct {
		input    string
		expected MigrationMode
	}{
		{"versioned", MigrationModeVersioned},
		{"auto", MigrationModeAuto},
		{"", MigrationModeVersioned},          // default
		{"unknown", MigrationModeVersioned},   // unknown defaults to versioned
		{"VERSIONED", MigrationModeVersioned}, // case sensitive - defaults to versioned
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := ParseMigrationMode(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMigrationModeConstants(t *testing.T) {
	// Verify constant values are correct
	assert.Equal(t, MigrationMode("versioned"), MigrationModeVersioned)
	assert.Equal(t, MigrationMode("auto"), MigrationModeAuto)
}
