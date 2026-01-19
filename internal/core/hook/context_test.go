package hook

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildContext(t *testing.T) {
	taskID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	jobID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	startTime := time.Date(2025, 1, 17, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 17, 10, 1, 30, 0, time.UTC)
	duration := 90 * time.Second

	ctx := BuildContext(
		taskID, "test-task", "/local/path", "remote:backup", "UPLOAD",
		jobID, "SUCCESS", "MANUAL", startTime, endTime,
		"ON_SUCCESS", "", duration,
		100, 1048576, 5, 0,
	)

	require.NotNil(t, ctx)

	assert.Equal(t, taskID, ctx.Task.ID)
	assert.Equal(t, "test-task", ctx.Task.Name)
	assert.Equal(t, "/local/path", ctx.Task.SourcePath)
	assert.Equal(t, "remote:backup", ctx.Task.RemotePath)
	assert.Equal(t, "UPLOAD", ctx.Task.Direction)

	assert.Equal(t, jobID, ctx.Job.ID)
	assert.Equal(t, "SUCCESS", ctx.Job.Status)
	assert.Equal(t, "MANUAL", ctx.Job.Trigger)
	assert.Equal(t, startTime, ctx.Job.StartTime)
	assert.Equal(t, endTime, ctx.Job.EndTime)

	assert.Equal(t, "ON_SUCCESS", ctx.Event)
	assert.Equal(t, "", ctx.Error)
	assert.Equal(t, duration, ctx.Duration)

	assert.Equal(t, int64(100), ctx.Stats.FilesTransferred)
	assert.Equal(t, int64(1048576), ctx.Stats.BytesTransferred)
	assert.Equal(t, int64(5), ctx.Stats.FilesDeleted)
	assert.Equal(t, int64(0), ctx.Stats.ErrorCount)

	assert.NotNil(t, ctx.Env)
	_, hasPath := ctx.Env["PATH"]
	assert.True(t, hasPath || len(ctx.Env) > 0)
}

func TestBuildContext_EnvVariables(t *testing.T) {
	testKey := "RCLONE_SYNC_TEST_VAR"
	testValue := "test_value_12345"
	err := os.Setenv(testKey, testValue)
	require.NoError(t, err)
	defer os.Unsetenv(testKey)

	ctx := BuildContext(
		uuid.New(), "task", "/src", "dst:", "BIDIRECTIONAL",
		uuid.New(), "RUNNING", "REALTIME", time.Now(), time.Time{},
		"ON_START", "", 0,
		0, 0, 0, 0,
	)

	require.NotNil(t, ctx)
	assert.Equal(t, testValue, ctx.Env[testKey])
}
