package rclone_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

func TestSyncEngine_RunTask_Upload(t *testing.T) {
	connService, taskService, jobService, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file in source
	testFilePath := filepath.Join(sourceDir, "upload_test.txt")
	err := os.WriteFile(testFilePath, []byte("upload content"), 0644)
	require.NoError(t, err)

	// Create a file in dest that should be removed (testing --delete-during implicit behavior of Sync)
	destGarbagePath := filepath.Join(destDir, "should_be_deleted.txt")
	err = os.WriteFile(destGarbagePath, []byte("garbage"), 0644)
	require.NoError(t, err)

	// 2. Create Connection and Task via ConnectionService (this goes to database)
	testConn, err := connService.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"})
	require.NoError(t, err)

	testTask, err := taskService.CreateTask(ctx,
		"TestUploadSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionUpload),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobService, nil, dataDir)

	// 4. Reload task with Connection edge before running
	testTask, err = taskService.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 5. Run the task - this should use DBStorage to read the connection config
	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	// 6. Verify results
	// Source file should exist in Dest
	destFilePath := filepath.Join(destDir, "upload_test.txt")
	_, err = os.Stat(destFilePath)
	assert.NoError(t, err, "Source file should differ to destination")

	// Dest garbage should be gone (Note: Sync uses --delete-during by default in many contexts,
	// but rclone's fs.Sync defaults depend on parameters.
	// In our implementation we passed 'false' for copyEmptySrcDirs.
	// Wait, rclone Sync naturally deletes extraneous files in dest ONLY if we configure it or use sync command.
	// rclonesync.Sync(ctx, fdst, fsrc, ...) acts like 'rclone sync', so it SHOULD delete.)
	// Correction: The rclonesync.Sync function signature is Sync(ctx, dest, src, bool).
	// It implements the sync command which makes dest identical to src, including deletions.
	_, err = os.Stat(destGarbagePath)
	assert.True(t, os.IsNotExist(err), "Extra file in destination should be deleted")

	// Verify Job Status
	jobs, err := jobService.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, string(model.JobStatusSuccess), string(jobs[0].Status))
}

func TestSyncEngine_RunTask_Download(t *testing.T) {
	connService, taskService, jobService, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file in Remote (simulated by destDir)
	remoteFilePath := filepath.Join(destDir, "download_test.txt")
	err := os.WriteFile(remoteFilePath, []byte("download content"), 0644)
	require.NoError(t, err)

	// Create garbage in Source that should be deleted
	sourceGarbagePath := filepath.Join(sourceDir, "local_garbage.txt")
	err = os.WriteFile(sourceGarbagePath, []byte("local garbage"), 0644)
	require.NoError(t, err)

	// 2. Create Connection and Task via ConnectionService (this goes to database)
	testConn, err := connService.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"})
	require.NoError(t, err)

	testTask, err := taskService.CreateTask(ctx,
		"TestDownloadSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionDownload),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobService, nil, dataDir)

	// 4. Reload task with Connection edge before running
	testTask, err = taskService.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 5. Run the task - this should use DBStorage to read the connection config
	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	// 6. Verify results
	// Remote file should exist in Source
	sourceTransferredPath := filepath.Join(sourceDir, "download_test.txt")
	_, err = os.Stat(sourceTransferredPath)
	assert.NoError(t, err, "Remote file should be downloaded to source")

	// Source garbage should be reduced
	_, err = os.Stat(sourceGarbagePath)
	assert.True(t, os.IsNotExist(err), "Extra file in source should be deleted")

	// Verify Job Status
	jobs, err := jobService.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, string(model.JobStatusSuccess), string(jobs[0].Status))
}

func TestSyncEngine_RunTask_Bidirectional(t *testing.T) {
	connService, taskService, jobService, _ := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. Setup test directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create a test file in source
	sourceFilePath := filepath.Join(sourceDir, "bidirectional_source.txt")
	err := os.WriteFile(sourceFilePath, []byte("source content"), 0644)
	require.NoError(t, err)

	// Create a test file in dest
	destFilePath := filepath.Join(destDir, "bidirectional_dest.txt")
	err = os.WriteFile(destFilePath, []byte("dest content"), 0644)
	require.NoError(t, err)

	// 2. Create Connection and Task via ConnectionService (this goes to database)
	testConn, err := connService.CreateConnection(ctx, "local", "local", map[string]string{"type": "local"})
	require.NoError(t, err)

	testTask, err := taskService.CreateTask(ctx,
		"TestBidirectionalSync",
		sourceDir,
		testConn.ID,
		destDir,
		string(model.SyncDirectionBidirectional),
		"",
		false,
		nil,
	)
	require.NoError(t, err)

	// 3. Setup SyncEngine
	dataDir := t.TempDir()
	syncEngine := rclone.NewSyncEngine(jobService, nil, dataDir)

	// 4. Reload task with Connection edge before running
	testTask, err = taskService.GetTaskWithConnection(ctx, testTask.ID)
	require.NoError(t, err)

	// 5. Run the task - this should use DBStorage to read the connection config
	err = syncEngine.RunTask(ctx, testTask, model.JobTriggerManual)
	require.NoError(t, err)

	// 6. Verify results
	// Source file should exist in Dest
	destSourceFilePath := filepath.Join(destDir, "bidirectional_source.txt")
	_, err = os.Stat(destSourceFilePath)
	assert.NoError(t, err, "Source file should be synced to destination")

	// Dest file should exist in Source
	sourceDestFilePath := filepath.Join(sourceDir, "bidirectional_dest.txt")
	_, err = os.Stat(sourceDestFilePath)
	assert.NoError(t, err, "Destination file should be synced to source")

	// Verify Job Status
	jobs, err := jobService.ListJobs(ctx, &testTask.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, string(model.JobStatusSuccess), string(jobs[0].Status))
}
