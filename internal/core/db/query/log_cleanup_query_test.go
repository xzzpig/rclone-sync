package query

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
)

// Helper to create a test connection query
func createTestConnQuery(t *testing.T, client *ent.Client) *ConnectionQuery {
	t.Helper()
	encryptor, err := crypto.NewEncryptor("test-secret-key-32-bytes-long!!")
	require.NoError(t, err)
	return NewConnectionQuery(client, encryptor)
}

func TestLogCleanupQuery(t *testing.T) {
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	defer client.Close()

	jobQuery := NewJobQuery(client)
	taskQuery := NewTaskQuery(client)
	connQuery := createTestConnQuery(t, client)
	ctx := context.Background()

	// Create test connection with logs
	testConn, err := connQuery.CreateConnection(ctx, "test-cleanup-conn-"+uuid.NewString(), "local", map[string]string{
		"type": "local",
	}, nil)
	require.NoError(t, err)

	task, err := taskQuery.CreateTask(ctx, "Cleanup Test Task "+uuid.NewString(), "/l", testConn.ID, "/r", string(model.SyncDirectionBidirectional), "", false, true, nil)
	require.NoError(t, err)

	job, err := jobQuery.CreateJob(ctx, task.ID, model.JobTriggerManual)
	require.NoError(t, err)

	// Add logs
	for i := 0; i < 15; i++ {
		_, err := jobQuery.AddJobLog(ctx, job.ID, string(model.LogLevelInfo), string(model.LogActionUpload), "/file"+string(rune('0'+i)), int64(i*100))
		require.NoError(t, err)
		time.Sleep(time.Millisecond)
	}

	t.Run("NewLogCleanupQuery", func(t *testing.T) {
		svc := NewLogCleanupQuery(client, 1000)
		assert.NotNil(t, svc)
	})

	t.Run("CleanupLogs", func(t *testing.T) {
		svc := NewLogCleanupQuery(client, 5)

		// Run cleanup
		err := svc.CleanupLogs(ctx)
		assert.NoError(t, err)

		// Verify logs were cleaned up for testConn
		count, err := jobQuery.CountJobLogs(ctx, &testConn.ID, nil, nil, "")
		assert.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("CleanupLogsForConnection", func(t *testing.T) {
		// Create another connection with logs
		conn2, err := connQuery.CreateConnection(ctx, "cleanup-test-2-"+uuid.NewString(), "local", map[string]string{
			"type": "local",
		}, nil)
		require.NoError(t, err)

		task2, err := taskQuery.CreateTask(ctx, "Cleanup Test Task 2 "+uuid.NewString(), "/l", conn2.ID, "/r", string(model.SyncDirectionBidirectional), "", false, true, nil)
		require.NoError(t, err)

		job2, err := jobQuery.CreateJob(ctx, task2.ID, model.JobTriggerManual)
		require.NoError(t, err)

		for i := 0; i < 10; i++ {
			_, err := jobQuery.AddJobLog(ctx, job2.ID, string(model.LogLevelInfo), string(model.LogActionDownload), "/file2-"+string(rune('0'+i)), int64(i*200))
			require.NoError(t, err)
			time.Sleep(time.Millisecond)
		}

		svc := NewLogCleanupQuery(client, 3)

		// Cleanup specific connection
		err = svc.CleanupLogsForConnection(ctx, conn2.ID)
		assert.NoError(t, err)

		// Verify only 3 logs remain for conn2
		count, err := jobQuery.CountJobLogs(ctx, &conn2.ID, nil, nil, "")
		assert.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("StartAndStop", func(t *testing.T) {
		svc := NewLogCleanupQuery(client, 1000)

		// Start with a cron schedule
		err := svc.Start("@every 1h")
		assert.NoError(t, err)

		// Stop should not error
		svc.Stop()
	})

	t.Run("Start_InvalidSchedule", func(t *testing.T) {
		svc := NewLogCleanupQuery(client, 1000)

		// Start with invalid schedule
		err := svc.Start("invalid-schedule")
		assert.Error(t, err)
	})

	t.Run("Stop_NotStarted", func(t *testing.T) {
		svc := NewLogCleanupQuery(client, 1000)

		// Stop without start should not panic
		svc.Stop()
	})
}

func TestLogCleanupQuery_Integration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", db.InMemoryDSN())
	defer client.Close()

	jobQuery := NewJobQuery(client)
	taskQuery := NewTaskQuery(client)
	connQuery := createTestConnQuery(t, client)
	ctx := context.Background()

	// Create multiple connections with different log counts
	var connectionIDs []uuid.UUID
	for i := 0; i < 3; i++ {
		conn, err := connQuery.CreateConnection(ctx, "integration-conn-"+uuid.NewString(), "local", map[string]string{
			"type": "local",
		}, nil)
		require.NoError(t, err)
		connectionIDs = append(connectionIDs, conn.ID)

		task, err := taskQuery.CreateTask(ctx, "Integration Task "+uuid.NewString(), "/l", conn.ID, "/r", string(model.SyncDirectionBidirectional), "", false, true, nil)
		require.NoError(t, err)

		job, err := jobQuery.CreateJob(ctx, task.ID, model.JobTriggerManual)
		require.NoError(t, err)

		// Add different number of logs per connection
		logCount := (i + 1) * 5 // 5, 10, 15 logs
		for j := 0; j < logCount; j++ {
			_, err := jobQuery.AddJobLog(ctx, job.ID, string(model.LogLevelInfo), string(model.LogActionUpload), "/file", int64(j*100))
			require.NoError(t, err)
			time.Sleep(time.Millisecond)
		}
	}

	t.Run("CleanupLogs_AllConnections", func(t *testing.T) {
		svc := NewLogCleanupQuery(client, 3)

		// Cleanup all
		err := svc.CleanupLogs(ctx)
		assert.NoError(t, err)

		// Verify each connection has at most 3 logs
		for _, connID := range connectionIDs {
			count, err := jobQuery.CountJobLogs(ctx, &connID, nil, nil, "")
			assert.NoError(t, err)
			assert.LessOrEqual(t, count, 3)
		}
	})
}
