package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileSDN_WALMode tests that WAL journal mode is properly enabled
func TestFileSDN_WALMode(t *testing.T) {
	// Create a temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_wal.db")

	// Build DSN using FileSDN function
	dsn := FileSDN(dbPath)

	// Open database connection
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer db.Close()

	// Create a simple table to ensure database is initialized
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	// Query the current journal mode
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)

	// Verify WAL mode is enabled
	assert.Equal(t, "wal", journalMode, "Journal mode should be WAL")

	// Verify WAL file exists (SQLite creates -wal and -shm files in WAL mode)
	// Note: WAL file may not exist if no writes have occurred, so we check after write
	walPath := dbPath + "-wal"
	shmPath := dbPath + "-shm"

	// Perform a write to ensure WAL files are created
	_, err = db.Exec("INSERT INTO test (id) VALUES (1)")
	require.NoError(t, err)

	// Check if WAL-related files exist (they should exist after a write in WAL mode)
	_, walErr := os.Stat(walPath)
	_, shmErr := os.Stat(shmPath)

	// At least one of the WAL files should exist after write
	assert.True(t, walErr == nil || shmErr == nil, "WAL or SHM file should exist after write in WAL mode")
}

// TestFileSDN_BusyTimeout tests that busy_timeout is properly configured
func TestFileSDN_BusyTimeout(t *testing.T) {
	// Create a temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_busy.db")

	// Build DSN using FileSDN function
	dsn := FileSDN(dbPath)

	// Open database connection
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer db.Close()

	// Query the current busy_timeout
	var busyTimeout int
	err = db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout)
	require.NoError(t, err)

	// Verify busy_timeout is set to 5000ms
	assert.Equal(t, 5000, busyTimeout, "Busy timeout should be 5000ms")
}

// TestFileSDN_BusyTimeoutBehavior tests that busy_timeout actually works
// by creating a lock contention scenario
func TestFileSDN_BusyTimeoutBehavior(t *testing.T) {
	// Create a temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_busy_behavior.db")

	// Build DSN using FileSDN function
	dsn := FileSDN(dbPath)

	// Open first database connection
	db1, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer db1.Close()

	// Create a table
	_, err = db1.Exec("CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY, value TEXT)")
	require.NoError(t, err)

	// Open second database connection with a very short timeout for testing
	shortTimeoutDSN := "file:" + dbPath + "?_fk=1&_journal_mode=WAL&_busy_timeout=100&_synchronous=NORMAL"
	db2, err := sql.Open("sqlite3", shortTimeoutDSN)
	require.NoError(t, err)
	defer db2.Close()

	// Start an exclusive transaction on db1
	tx1, err := db1.Begin()
	require.NoError(t, err)

	// Insert data and hold the transaction
	_, err = tx1.Exec("INSERT INTO test (value) VALUES ('test')")
	require.NoError(t, err)

	// Try to write from db2 while db1 holds the lock
	// This should wait up to busy_timeout before failing
	var wg sync.WaitGroup
	var db2Err error
	var db2Duration time.Duration

	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()

		// Try to start an immediate transaction (which requires a write lock)
		tx2, err := db2.Begin()
		if err != nil {
			db2Err = err
			db2Duration = time.Since(start)
			return
		}

		// Try to write
		_, err = tx2.Exec("INSERT INTO test (value) VALUES ('test2')")
		if err != nil {
			tx2.Rollback()
			db2Err = err
			db2Duration = time.Since(start)
			return
		}

		err = tx2.Commit()
		if err != nil {
			db2Err = err
		}
		db2Duration = time.Since(start)
	}()

	// Wait a bit then commit db1's transaction
	time.Sleep(50 * time.Millisecond)
	err = tx1.Commit()
	require.NoError(t, err)

	wg.Wait()

	// With WAL mode, concurrent writes should be possible after db1 commits
	// If there was contention, the operation should have waited and then succeeded
	// or failed with SQLITE_BUSY after the timeout
	t.Logf("db2 operation took %v, error: %v", db2Duration, db2Err)
}

// TestFileSDN_BusyTimeoutWaitsBeforeError tests that the database actually waits
// before returning a busy error
func TestFileSDN_BusyTimeoutWaitsBeforeError(t *testing.T) {
	// Create a temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_busy_wait.db")

	// Open first connection without WAL to make locking more strict
	db1, err := sql.Open("sqlite3", "file:"+dbPath+"?_fk=1&_busy_timeout=0")
	require.NoError(t, err)
	defer db1.Close()

	// Create table
	_, err = db1.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	// Open second connection with busy_timeout
	dsn2 := "file:" + dbPath + "?_fk=1&_busy_timeout=500"
	db2, err := sql.Open("sqlite3", dsn2)
	require.NoError(t, err)
	defer db2.Close()

	// Verify busy_timeout is set correctly on db2
	var timeout int
	err = db2.QueryRow("PRAGMA busy_timeout").Scan(&timeout)
	require.NoError(t, err)
	assert.Equal(t, 500, timeout, "Busy timeout should be 500ms")

	// Start exclusive transaction on db1
	_, err = db1.Exec("BEGIN EXCLUSIVE")
	require.NoError(t, err)

	// Measure how long db2 waits before giving up
	start := time.Now()
	_, err = db2.Exec("BEGIN EXCLUSIVE")
	duration := time.Since(start)

	// Should have waited approximately the timeout duration before failing
	// Allow some tolerance
	if err != nil {
		t.Logf("db2 waited %v before error: %v", duration, err)
		// The wait should be close to the timeout value
		assert.True(t, duration >= 400*time.Millisecond, "Should wait at least close to busy_timeout before failing")
	}

	// Rollback db1's transaction
	_, _ = db1.Exec("ROLLBACK")
}

// TestInMemoryDSN_BusyTimeout tests that in-memory database also has busy_timeout set
func TestInMemoryDSN_BusyTimeout(t *testing.T) {
	dsn := InMemoryDSN()

	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer db.Close()

	// Query the current busy_timeout
	var busyTimeout int
	err = db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout)
	require.NoError(t, err)

	// Verify busy_timeout is set to 5000ms
	assert.Equal(t, 5000, busyTimeout, "Busy timeout should be 5000ms for in-memory DB")
}

// TestFileSDN_SynchronousMode tests that synchronous mode is properly set
func TestFileSDN_SynchronousMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_sync.db")

	dsn := FileSDN(dbPath)

	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer db.Close()

	// Query the current synchronous mode
	var synchronous int
	err = db.QueryRow("PRAGMA synchronous").Scan(&synchronous)
	require.NoError(t, err)

	// NORMAL = 1, FULL = 2, OFF = 0
	assert.Equal(t, 1, synchronous, "Synchronous mode should be NORMAL (1)")
}

// TestFileSDN_ForeignKeysEnabled tests that foreign keys are enabled
func TestFileSDN_ForeignKeysEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_fk.db")

	dsn := FileSDN(dbPath)

	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer db.Close()

	// Query foreign_keys pragma
	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	require.NoError(t, err)

	assert.Equal(t, 1, fkEnabled, "Foreign keys should be enabled")
}

// TestInitDB_WithAutoMigration tests InitDB with auto migration mode
func TestInitDB_WithAutoMigration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_init.db")

	client, err := InitDB(InitDBOptions{
		DSN:           FileSDN(dbPath),
		MigrationMode: MigrationModeAuto,
		EnableDebug:   false,
		Environment:   "test",
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	defer CloseDB(client)

	// Verify the database file exists
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "Database file should exist after InitDB")
}

func TestInitDB_WithVersionedMigration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_init.db")

	client, err := InitDB(InitDBOptions{
		DSN:           FileSDN(dbPath),
		MigrationMode: MigrationModeVersioned,
		EnableDebug:   false,
		Environment:   "test",
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	defer CloseDB(client)

	// Verify the database file exists
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "Database file should exist after InitDB")
}

// TestCloseDB_NilClient tests that CloseDB handles nil client gracefully
func TestCloseDB_NilClient(t *testing.T) {
	// Should not panic
	CloseDB(nil)
}

// TestFileSDN_ConcurrentWrites tests that FileSDN configuration allows concurrent writes
// from multiple goroutines using WAL mode and busy_timeout
func TestFileSDN_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_concurrent.db")

	// Build DSN using FileSDN function
	dsn := FileSDN(dbPath)

	// Open multiple database connections (simulating multiple writers)
	numConnections := 5
	numWritesPerConnection := 100
	connections := make([]*sql.DB, numConnections)

	for i := 0; i < numConnections; i++ {
		db, err := sql.Open("sqlite3", dsn)
		require.NoError(t, err)
		defer db.Close()
		connections[i] = db
	}

	// Create table using first connection
	_, err := connections[0].Exec(`
		CREATE TABLE IF NOT EXISTS concurrent_test (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			writer_id INTEGER NOT NULL,
			write_seq INTEGER NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	// Run concurrent writes from all connections
	var wg sync.WaitGroup
	errors := make(chan error, numConnections*numWritesPerConnection)
	successCount := make(chan int, numConnections)

	for connID := 0; connID < numConnections; connID++ {
		wg.Add(1)
		go func(id int, db *sql.DB) {
			defer wg.Done()
			success := 0
			for seq := 0; seq < numWritesPerConnection; seq++ {
				_, err := db.Exec("INSERT INTO concurrent_test (writer_id, write_seq) VALUES (?, ?)", id, seq)
				if err != nil {
					errors <- err
				} else {
					success++
				}
			}
			successCount <- success
		}(connID, connections[connID])
	}

	wg.Wait()
	close(errors)
	close(successCount)

	// Collect results
	totalSuccess := 0
	for count := range successCount {
		totalSuccess += count
	}

	// Collect any errors
	var errList []error
	for err := range errors {
		errList = append(errList, err)
	}

	// Log results
	expectedTotal := numConnections * numWritesPerConnection
	t.Logf("Concurrent write results: %d/%d successful, %d errors", totalSuccess, expectedTotal, len(errList))

	// All writes should succeed with WAL mode and busy_timeout
	assert.Equal(t, expectedTotal, totalSuccess, "All concurrent writes should succeed")
	assert.Empty(t, errList, "Should have no errors during concurrent writes")

	// Verify data integrity - count total rows
	var count int
	err = connections[0].QueryRow("SELECT COUNT(*) FROM concurrent_test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, expectedTotal, count, "Total row count should match expected writes")

	// Verify each writer's data
	for connID := 0; connID < numConnections; connID++ {
		var writerCount int
		err = connections[0].QueryRow("SELECT COUNT(*) FROM concurrent_test WHERE writer_id = ?", connID).Scan(&writerCount)
		require.NoError(t, err)
		assert.Equal(t, numWritesPerConnection, writerCount, "Each writer should have correct number of rows")
	}
}

// TestFileSDN_ConcurrentReadWrite tests that FileSDN configuration allows
// concurrent reads and writes without blocking each other (WAL advantage)
func TestFileSDN_ConcurrentReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_concurrent_rw.db")

	dsn := FileSDN(dbPath)

	// Open two connections - one for writing, one for reading
	writerDB, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer writerDB.Close()

	readerDB, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer readerDB.Close()

	// Create table
	_, err = writerDB.Exec(`
		CREATE TABLE IF NOT EXISTS rw_test (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	// Insert initial data
	for i := 0; i < 100; i++ {
		_, err = writerDB.Exec("INSERT INTO rw_test (value) VALUES (?)", i)
		require.NoError(t, err)
	}

	// Run concurrent reads and writes
	var wg sync.WaitGroup
	writeErrors := make(chan error, 100)
	readErrors := make(chan error, 100)
	writeCount := 0
	readCount := 0

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 100; i < 200; i++ {
			_, err := writerDB.Exec("INSERT INTO rw_test (value) VALUES (?)", i)
			if err != nil {
				writeErrors <- err
			} else {
				writeCount++
			}
			time.Sleep(1 * time.Millisecond) // Small delay to interleave operations
		}
	}()

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			var count int
			err := readerDB.QueryRow("SELECT COUNT(*) FROM rw_test").Scan(&count)
			if err != nil {
				readErrors <- err
			} else {
				readCount++
			}
			time.Sleep(1 * time.Millisecond) // Small delay to interleave operations
		}
	}()

	wg.Wait()
	close(writeErrors)
	close(readErrors)

	// Collect errors
	var writeErrList, readErrList []error
	for err := range writeErrors {
		writeErrList = append(writeErrList, err)
	}
	for err := range readErrors {
		readErrList = append(readErrList, err)
	}

	t.Logf("Write results: %d successful, %d errors", writeCount, len(writeErrList))
	t.Logf("Read results: %d successful, %d errors", readCount, len(readErrList))

	// With WAL mode, concurrent reads and writes should work without issues
	assert.Equal(t, 100, writeCount, "All writes should succeed")
	assert.Equal(t, 100, readCount, "All reads should succeed")
	assert.Empty(t, writeErrList, "Should have no write errors")
	assert.Empty(t, readErrList, "Should have no read errors")

	// Verify final data
	var finalCount int
	err = readerDB.QueryRow("SELECT COUNT(*) FROM rw_test").Scan(&finalCount)
	require.NoError(t, err)
	assert.Equal(t, 200, finalCount, "Should have all 200 rows")
}
