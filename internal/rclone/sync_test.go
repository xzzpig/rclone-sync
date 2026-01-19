package rclone

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rclone/rclone/fs/accounting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/hook"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
)

// MockJobQuery is a mock for query.JobQuery
// We only need to mock the methods used by pollStats
type MockJobQuery struct {
	mock.Mock
}

func TestMain(m *testing.M) {
	// Initialize logger for tests
	{ // logger init block
		logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug, nil)
	}
	m.Run()
}

func (m *MockJobQuery) CreateJob(ctx context.Context, taskID uuid.UUID, trigger model.JobTrigger) (*ent.Job, error) {
	args := m.Called(ctx, taskID, trigger)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobQuery) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string, errStr string) (*ent.Job, error) {
	args := m.Called(ctx, jobID, status, errStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobQuery) AddJobLogsBatch(ctx context.Context, jobID uuid.UUID, logs []*ent.JobLog) error {
	args := m.Called(ctx, jobID, logs)
	return args.Error(0)
}

func (m *MockJobQuery) UpdateJobStats(ctx context.Context, jobID uuid.UUID, files, bytes, filesDeleted, errorCount int64) (*ent.Job, error) {
	args := m.Called(ctx, jobID, files, bytes, filesDeleted, errorCount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobQuery) AddJobLog(ctx context.Context, jobID uuid.UUID, level, what, path string, size int64) (*ent.JobLog, error) {
	args := m.Called(ctx, jobID, level, what, path, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.JobLog), args.Error(1)
}

func (m *MockJobQuery) GetJob(ctx context.Context, jobID uuid.UUID) (*ent.Job, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobQuery) GetLastJobByTaskID(ctx context.Context, taskID uuid.UUID) (*ent.Job, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobQuery) ListJobs(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID, limit, offset int) ([]*ent.Job, error) {
	args := m.Called(ctx, taskID, connectionID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ent.Job), args.Error(1)
}

func (m *MockJobQuery) GetJobWithLogs(ctx context.Context, jobID uuid.UUID) (*ent.Job, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobQuery) CountJobs(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID) (int, error) {
	args := m.Called(ctx, taskID, connectionID)
	return args.Int(0), args.Error(1)
}

func (m *MockJobQuery) ListJobLogs(ctx context.Context, connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string, limit, offset int) ([]*ent.JobLog, error) {
	args := m.Called(ctx, connectionID, taskID, jobID, level, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ent.JobLog), args.Error(1)
}

func (m *MockJobQuery) CountJobLogs(ctx context.Context, connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string) (int, error) {
	args := m.Called(ctx, connectionID, taskID, jobID, level)
	return args.Int(0), args.Error(1)
}

func (m *MockJobQuery) DeleteJob(ctx context.Context, jobID uuid.UUID) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

// TestPollStatsLogic tests the logic of pollStats using a mocked JobQuery
func TestPollStatsLogic(t *testing.T) {
	// 1. Setup Mock
	mockJobQuery := new(MockJobQuery)
	jobID := uuid.New()

	// 2. Setup SyncEngine
	engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, nil)
	engine.logger = zap.NewNop() // Setup logger

	// 3. Setup Context with Stats
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = accounting.WithStatsGroup(ctx, jobID.String())
	stats := accounting.Stats(ctx)
	assert.NotNil(t, stats)

	// 4. Run loop
	var wg sync.WaitGroup
	wg.Go(func() {
		engine.pollStats(ctx, jobID, &ent.Task{ID: uuid.New()}, time.Now())
	})

	// Allow some time for the loop to run
	// time.Sleep(100 * time.Millisecond)

	// Cancel context to stop loop
	cancel()
	wg.Wait()
}

// TestGetJobProgress tests the GetJobProgress method of SyncEngine
func TestGetJobProgress(t *testing.T) {
	// Setup
	mockJobQuery := new(MockJobQuery)
	engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, nil)

	// Test case 1: Job ID exists in lastEvents
	jobID1 := uuid.New()
	expectedEvent := &model.JobProgressEvent{
		JobID:            jobID1,
		FilesTransferred: 10,
		BytesTransferred: 1024,
	}
	engine.lastEvents[jobID1] = expectedEvent

	progress := engine.GetJobProgress(jobID1)
	assert.NotNil(t, progress, "Should return non-nil for existing job ID")
	assert.Equal(t, expectedEvent, progress, "Should return the correct progress event")

	// Test case 2: Job ID does not exist in lastEvents
	jobID2 := uuid.New()
	progress = engine.GetJobProgress(jobID2)
	assert.Nil(t, progress, "Should return nil for non-existing job ID")

	// Test case 3: Empty lastEvents map
	engine.lastEvents = make(map[uuid.UUID]*model.JobProgressEvent)
	progress = engine.GetJobProgress(jobID1)
	assert.Nil(t, progress, "Should return nil when lastEvents is empty")
}

// TestGetConflictResolutionFromOptions tests all branches of getConflictResolutionFromOptions
func TestGetConflictResolutionFromOptions(t *testing.T) {
	tests := []struct {
		name            string
		options         *model.TaskSyncOptions
		expectedResolve string
		expectedLoser   string
	}{
		{
			name:            "nil options - default",
			options:         nil,
			expectedResolve: "newer",
			expectedLoser:   "num",
		},
		{
			name:            "empty options - default",
			options:         &model.TaskSyncOptions{},
			expectedResolve: "newer",
			expectedLoser:   "num",
		},
		{
			name: "resolution: newer",
			options: &model.TaskSyncOptions{
				ConflictResolution: func() *model.ConflictResolution { v := model.ConflictResolutionNewer; return &v }(),
			},
			expectedResolve: "newer",
			expectedLoser:   "num",
		},
		{
			name: "resolution: local",
			options: &model.TaskSyncOptions{
				ConflictResolution: func() *model.ConflictResolution { v := model.ConflictResolutionLocal; return &v }(),
			},
			expectedResolve: "path1",
			expectedLoser:   "delete",
		},
		{
			name: "resolution: remote",
			options: &model.TaskSyncOptions{
				ConflictResolution: func() *model.ConflictResolution { v := model.ConflictResolutionRemote; return &v }(),
			},
			expectedResolve: "path2",
			expectedLoser:   "delete",
		},
		{
			name: "resolution: both",
			options: &model.TaskSyncOptions{
				ConflictResolution: func() *model.ConflictResolution { v := model.ConflictResolutionBoth; return &v }(),
			},
			expectedResolve: "none",
			expectedLoser:   "num",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolve, loser := getConflictResolutionFromOptions(tt.options)
			assert.Equal(t, tt.expectedResolve, resolve.String())
			assert.Equal(t, tt.expectedLoser, loser.String())
		})
	}
}

// TestFailJob tests the failJob method
func TestFailJob(t *testing.T) {
	mockJobQuery := new(MockJobQuery)
	engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, nil)
	engine.logger = zap.NewNop()

	jobID := uuid.New()
	testErr := assert.AnError

	// Expect UpdateJobStatus to be called
	mockJobQuery.On("UpdateJobStatus", mock.Anything, jobID, string(model.JobStatusFailed), testErr.Error()).
		Return((*ent.Job)(nil), nil).Once()

	ctx := context.Background()
	engine.failJob(ctx, jobID, testErr)

	mockJobQuery.AssertExpectations(t)
}

// TestRemoteStatsIntegration tests that RemoteStats can be called on the accounting.Stats.
// This test verifies that the rclone accounting API works as expected for getting total stats.
func TestRemoteStatsIntegration(t *testing.T) {
	// Setup context with stats group
	ctx := context.Background()
	groupName := uuid.New().String()
	ctx = accounting.WithStatsGroup(ctx, groupName)

	stats := accounting.Stats(ctx)
	assert.NotNil(t, stats, "Stats should not be nil")

	// Call RemoteStats - this should work without error
	// RemoteStats(false) returns local stats without contacting remotes
	rc, err := stats.RemoteStats(false)
	assert.NoError(t, err, "RemoteStats should not return error")
	assert.NotNil(t, rc, "RemoteStats should return non-nil rc.Params")

	// Verify expected fields exist in the returned stats
	// The returned value is an rc.Params (map[string]interface{})
	// Check for expected keys: totalTransfers, totalBytes, transfers, bytes
	_, hasTotalTransfers := rc["totalTransfers"]
	_, hasTotalBytes := rc["totalBytes"]
	_, hasTransfers := rc["transfers"]
	_, hasBytes := rc["bytes"]

	// These fields should be present (they might be 0 but should exist)
	assert.True(t, hasTotalTransfers, "RemoteStats should contain 'totalTransfers' field")
	assert.True(t, hasTotalBytes, "RemoteStats should contain 'totalBytes' field")
	assert.True(t, hasTransfers, "RemoteStats should contain 'transfers' field")
	assert.True(t, hasBytes, "RemoteStats should contain 'bytes' field")
}

// TestGetTotalStats tests the getTotalStats helper function.
func TestGetTotalStats(t *testing.T) {
	// Setup context with stats group
	ctx := context.Background()
	groupName := uuid.New().String()
	ctx = accounting.WithStatsGroup(ctx, groupName)

	stats := accounting.Stats(ctx)
	assert.NotNil(t, stats, "Stats should not be nil")

	// Get total stats using the helper function
	totalTransfers, totalBytes, filesChecked := getTotalStats(stats)

	// Initially, all should be 0 (no transfers started)
	assert.Equal(t, int64(0), totalTransfers, "Initial totalTransfers should be 0")
	assert.Equal(t, int64(0), totalBytes, "Initial totalBytes should be 0")
	assert.Equal(t, 0, filesChecked, "Initial filesChecked should be 0")
}

// TestGetTotalStats_NilStats tests getTotalStats with nil stats.
func TestGetTotalStats_NilStats(t *testing.T) {
	totalTransfers, totalBytes, filesChecked := getTotalStats(nil)
	assert.Equal(t, int64(0), totalTransfers, "totalTransfers should be 0 for nil stats")
	assert.Equal(t, int64(0), totalBytes, "totalBytes should be 0 for nil stats")
	assert.Equal(t, 0, filesChecked, "filesChecked should be 0 for nil stats")
}

// TestApplyFilterRules tests the applyFilterRules helper function.
func TestApplyFilterRules(t *testing.T) {
	tests := []struct {
		name        string
		rules       []string
		expectErr   bool
		errContains string
	}{
		{
			name:      "empty rules - returns original context",
			rules:     nil,
			expectErr: false,
		},
		{
			name:      "empty slice - returns original context",
			rules:     []string{},
			expectErr: false,
		},
		{
			name:      "valid exclude rule",
			rules:     []string{"- *.tmp"},
			expectErr: false,
		},
		{
			name:      "valid include rule",
			rules:     []string{"+ *.go"},
			expectErr: false,
		},
		{
			name:      "multiple valid rules",
			rules:     []string{"- node_modules/**", "- .git/**", "+ **"},
			expectErr: false,
		},
		{
			name:        "invalid rule - missing prefix",
			rules:       []string{"*.tmp"},
			expectErr:   true,
			errContains: "error_filter_rule_invalid",
		},
		{
			name:        "invalid rule - wrong prefix",
			rules:       []string{"* *.tmp"},
			expectErr:   true,
			errContains: "error_filter_rule_invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			newCtx, err := applyFilterRules(ctx, tt.rules)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, newCtx)
				// When no rules, context should be unchanged
				if len(tt.rules) == 0 {
					assert.Equal(t, ctx, newCtx)
				}
			}
		})
	}
}

// TestGetSyncOptionsFromTask tests the getSyncOptionsFromTask helper function.
func TestGetSyncOptionsFromTask(t *testing.T) {
	tests := []struct {
		name     string
		options  *model.TaskSyncOptions
		expected SyncOptions
	}{
		{
			name:     "nil options - returns empty SyncOptions",
			options:  nil,
			expected: SyncOptions{},
		},
		{
			name:     "empty options - returns empty SyncOptions",
			options:  &model.TaskSyncOptions{},
			expected: SyncOptions{},
		},
		{
			name: "filters only",
			options: &model.TaskSyncOptions{
				Filters: []string{"- *.tmp", "+ **"},
			},
			expected: SyncOptions{
				Filters: []string{"- *.tmp", "+ **"},
			},
		},
		{
			name: "noDelete only",
			options: &model.TaskSyncOptions{
				NoDelete: func() *bool { v := true; return &v }(),
			},
			expected: SyncOptions{
				NoDelete: true,
			},
		},
		{
			name: "transfers only",
			options: &model.TaskSyncOptions{
				Transfers: func() *int { v := 8; return &v }(),
			},
			expected: SyncOptions{
				Transfers: 8,
			},
		},
		{
			name: "all options combined",
			options: &model.TaskSyncOptions{
				Filters:   []string{"- node_modules/**", "+ **"},
				NoDelete:  func() *bool { v := true; return &v }(),
				Transfers: func() *int { v := 32; return &v }(),
			},
			expected: SyncOptions{
				Filters:   []string{"- node_modules/**", "+ **"},
				NoDelete:  true,
				Transfers: 32,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSyncOptionsFromTask(tt.options)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDetermineTransfers tests the determineTransfers helper function.
func TestDetermineTransfers(t *testing.T) {
	tests := []struct {
		name             string
		taskTransfers    int
		defaultTransfers int
		expected         int
	}{
		{
			name:             "task-level value takes priority",
			taskTransfers:    8,
			defaultTransfers: 16,
			expected:         8,
		},
		{
			name:             "global config used when task is 0",
			taskTransfers:    0,
			defaultTransfers: 16,
			expected:         16,
		},
		{
			name:             "built-in default when both are 0",
			taskTransfers:    0,
			defaultTransfers: 0,
			expected:         DefaultTransfers,
		},
		{
			name:             "built-in default when both are negative",
			taskTransfers:    -1,
			defaultTransfers: -1,
			expected:         DefaultTransfers,
		},
		{
			name:             "task-level edge case: 1",
			taskTransfers:    1,
			defaultTransfers: 64,
			expected:         1,
		},
		{
			name:             "task-level edge case: 64",
			taskTransfers:    64,
			defaultTransfers: 4,
			expected:         64,
		},
		{
			name:             "global config edge case: 1",
			taskTransfers:    0,
			defaultTransfers: 1,
			expected:         1,
		},
		{
			name:             "global config edge case: 64",
			taskTransfers:    0,
			defaultTransfers: 64,
			expected:         64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineTransfers(tt.taskTransfers, tt.defaultTransfers)
			assert.Equal(t, tt.expected, result, "determineTransfers result mismatch")
		})
	}
}

// TestGetTransferDirection tests the getTransferDirection helper function.
func TestGetTransferDirection(t *testing.T) {
	tests := []struct {
		name           string
		srcFs          string
		taskSourcePath string
		expected       model.TransferDirection
	}{
		{
			name:           "srcFs equals taskSourcePath -> UPLOAD",
			srcFs:          "/home/user/data",
			taskSourcePath: "/home/user/data",
			expected:       model.TransferDirectionUpload,
		},
		{
			name:           "srcFs differs from taskSourcePath -> DOWNLOAD",
			srcFs:          "remote:/backup",
			taskSourcePath: "/home/user/data",
			expected:       model.TransferDirectionDownload,
		},
		{
			name:           "empty srcFs with non-empty taskSourcePath -> DOWNLOAD",
			srcFs:          "",
			taskSourcePath: "/home/user/data",
			expected:       model.TransferDirectionDownload,
		},
		{
			name:           "both empty -> UPLOAD (equal strings)",
			srcFs:          "",
			taskSourcePath: "",
			expected:       model.TransferDirectionUpload,
		},
		{
			name:           "srcFs with trailing slash differs -> DOWNLOAD",
			srcFs:          "/home/user/data/",
			taskSourcePath: "/home/user/data",
			expected:       model.TransferDirectionDownload,
		},
		{
			name:           "remote source matches task source -> UPLOAD",
			srcFs:          "gdrive:/Documents",
			taskSourcePath: "gdrive:/Documents",
			expected:       model.TransferDirectionUpload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTransferDirection(tt.srcFs, tt.taskSourcePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsEmptyJob tests the isEmptyJob helper function.
func TestIsEmptyJob(t *testing.T) {
	tests := []struct {
		name             string
		status           model.JobStatus
		filesTransferred int
		bytesTransferred int64
		filesDeleted     int
		errorCount       int
		expected         bool
	}{
		{
			name:             "success+no_activity->empty",
			status:           model.JobStatusSuccess,
			filesTransferred: 0,
			bytesTransferred: 0,
			filesDeleted:     0,
			errorCount:       0,
			expected:         true,
		},
		{
			name:             "success+has_files->not_empty",
			status:           model.JobStatusSuccess,
			filesTransferred: 5,
			bytesTransferred: 1024,
			filesDeleted:     0,
			errorCount:       0,
			expected:         false,
		},
		{
			name:             "success+only_bytes->not_empty",
			status:           model.JobStatusSuccess,
			filesTransferred: 0,
			bytesTransferred: 1024,
			filesDeleted:     0,
			errorCount:       0,
			expected:         false,
		},
		{
			name:             "success+only_deletes->not_empty",
			status:           model.JobStatusSuccess,
			filesTransferred: 0,
			bytesTransferred: 0,
			filesDeleted:     3,
			errorCount:       0,
			expected:         false,
		},
		{
			name:             "success+only_errors->not_empty",
			status:           model.JobStatusSuccess,
			filesTransferred: 0,
			bytesTransferred: 0,
			filesDeleted:     0,
			errorCount:       2,
			expected:         false,
		},
		{
			name:             "failed+no_activity->not_empty",
			status:           model.JobStatusFailed,
			filesTransferred: 0,
			bytesTransferred: 0,
			filesDeleted:     0,
			errorCount:       0,
			expected:         false,
		},
		{
			name:             "cancelled+no_activity->not_empty",
			status:           model.JobStatusCancelled,
			filesTransferred: 0,
			bytesTransferred: 0,
			filesDeleted:     0,
			errorCount:       0,
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &ent.Job{
				Status:           tt.status,
				FilesTransferred: tt.filesTransferred,
				BytesTransferred: tt.bytesTransferred,
				FilesDeleted:     tt.filesDeleted,
				ErrorCount:       tt.errorCount,
			}
			result := isEmptyJob(job)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// MockHookExecutor is a mock for ports.HookExecutor
type MockHookExecutor struct {
	mock.Mock
}

func (m *MockHookExecutor) Execute(ctx context.Context, task *ent.Task, job *ent.Job, event model.HookEvent, syncErr error) error {
	args := m.Called(ctx, task, job, event, syncErr)
	return args.Error(0)
}

// TestExecuteHooks tests the executeHooks method of SyncEngine
func TestExecuteHooks(t *testing.T) {
	t.Run("hookExecutor nil - does nothing", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, nil)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()

		// Should not panic and return gracefully
		engine.executeHooks(context.Background(), task, job, model.HookEventOnSuccess, nil)
	})

	t.Run("hookExecutor executes successfully", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		mockHookExecutor := new(MockHookExecutor)

		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, mockHookExecutor)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()

		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnSuccess, nil).Return(nil).Once()

		engine.executeHooks(context.Background(), task, job, model.HookEventOnSuccess, nil)

		mockHookExecutor.AssertExpectations(t)
	})

	t.Run("hookExecutor executes with syncErr", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		mockHookExecutor := new(MockHookExecutor)

		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, mockHookExecutor)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()
		syncErr := errors.New("sync failed")

		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnFailure, syncErr).Return(nil).Once()

		engine.executeHooks(context.Background(), task, job, model.HookEventOnFailure, syncErr)

		mockHookExecutor.AssertExpectations(t)
	})

	t.Run("hookExecutor returns error - logs warning", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		mockHookExecutor := new(MockHookExecutor)

		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, mockHookExecutor)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()
		hookErr := errors.New("hook execution failed")

		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnEnd, nil).Return(hookErr).Once()

		// Should not panic, just log warning
		engine.executeHooks(context.Background(), task, job, model.HookEventOnEnd, nil)

		mockHookExecutor.AssertExpectations(t)
	})

	t.Run("all hook events are supported", func(t *testing.T) {
		events := []model.HookEvent{
			model.HookEventOnStart,
			model.HookEventOnSuccess,
			model.HookEventOnFailure,
			model.HookEventOnEnd,
		}

		for _, event := range events {
			t.Run(string(event), func(t *testing.T) {
				mockJobQuery := new(MockJobQuery)
				mockHookExecutor := new(MockHookExecutor)

				engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, mockHookExecutor)
				engine.logger = zap.NewNop()

				task := createTestTaskForHook()
				job := createTestJobForHook()

				mockHookExecutor.On("Execute", mock.Anything, task, job, event, nil).Return(nil).Once()

				engine.executeHooks(context.Background(), task, job, event, nil)

				mockHookExecutor.AssertExpectations(t)
			})
		}
	})
}

// TestExecuteOnStartHooks tests the executeOnStartHooks method of SyncEngine
func TestExecuteOnStartHooks(t *testing.T) {
	t.Run("hookExecutor nil - returns nil", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, nil)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()

		err := engine.executeOnStartHooks(context.Background(), task, job)
		assert.NoError(t, err)
	})

	t.Run("hookExecutor executes successfully - returns nil", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		mockHookExecutor := new(MockHookExecutor)

		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, mockHookExecutor)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()

		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnStart, nil).Return(nil).Once()

		err := engine.executeOnStartHooks(context.Background(), task, job)
		assert.NoError(t, err)
		mockHookExecutor.AssertExpectations(t)
	})

	t.Run("CancelError - updates status to CANCELLED and calls on_end", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		mockHookExecutor := new(MockHookExecutor)

		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, mockHookExecutor)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()

		hookID := uuid.New()
		cancelErr := &hook.CancelError{HookID: hookID, Cause: errors.New("pre-condition not met")}

		// First call: on_start returns CancelError
		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnStart, nil).
			Return(cancelErr).Once()

		// Job status should be updated to CANCELLED
		mockJobQuery.On("UpdateJobStatus", mock.Anything, job.ID, string(model.JobStatusCancelled), cancelErr.Error()).
			Return((*ent.Job)(nil), nil).Once()

		// on_end hook should be called with the cancelErr
		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnEnd, cancelErr).
			Return(nil).Once()

		err := engine.executeOnStartHooks(context.Background(), task, job)

		assert.Error(t, err)
		var returnedCancelErr *hook.CancelError
		assert.True(t, errors.As(err, &returnedCancelErr))
		assert.Equal(t, hookID, returnedCancelErr.HookID)

		mockJobQuery.AssertExpectations(t)
		mockHookExecutor.AssertExpectations(t)
	})

	t.Run("FatalError - updates status to FAILED and calls on_failure and on_end", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		mockHookExecutor := new(MockHookExecutor)

		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, mockHookExecutor)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()

		hookID := uuid.New()
		fatalErr := &hook.FatalError{HookID: hookID, Cause: errors.New("critical failure")}

		// First call: on_start returns FatalError
		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnStart, nil).
			Return(fatalErr).Once()

		// Job status should be updated to FAILED
		mockJobQuery.On("UpdateJobStatus", mock.Anything, job.ID, string(model.JobStatusFailed), fatalErr.Error()).
			Return((*ent.Job)(nil), nil).Once()

		// on_failure hook should be called with the fatalErr
		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnFailure, fatalErr).
			Return(nil).Once()

		// on_end hook should be called with the fatalErr
		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnEnd, fatalErr).
			Return(nil).Once()

		err := engine.executeOnStartHooks(context.Background(), task, job)

		assert.Error(t, err)
		var returnedFatalErr *hook.FatalError
		assert.True(t, errors.As(err, &returnedFatalErr))
		assert.Equal(t, hookID, returnedFatalErr.HookID)

		mockJobQuery.AssertExpectations(t)
		mockHookExecutor.AssertExpectations(t)
	})

	t.Run("other error (IGNORE mode) - logs warning and returns nil", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		mockHookExecutor := new(MockHookExecutor)

		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, mockHookExecutor)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()

		// Return a generic error (not CancelError or FatalError)
		genericErr := errors.New("transient hook failure")

		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnStart, nil).
			Return(genericErr).Once()

		err := engine.executeOnStartHooks(context.Background(), task, job)

		// Should return nil (IGNORE mode behavior)
		assert.NoError(t, err)

		// No status update or additional hooks should be called
		mockJobQuery.AssertNotCalled(t, "UpdateJobStatus")
		mockHookExecutor.AssertExpectations(t)
	})

	t.Run("CancelError - UpdateJobStatus fails - still returns CancelError", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		mockHookExecutor := new(MockHookExecutor)

		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, mockHookExecutor)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()

		hookID := uuid.New()
		cancelErr := &hook.CancelError{HookID: hookID, Cause: errors.New("cancelled")}

		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnStart, nil).
			Return(cancelErr).Once()

		// UpdateJobStatus fails
		mockJobQuery.On("UpdateJobStatus", mock.Anything, job.ID, string(model.JobStatusCancelled), cancelErr.Error()).
			Return((*ent.Job)(nil), errors.New("db error")).Once()

		// on_end should still be called
		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnEnd, cancelErr).
			Return(nil).Once()

		err := engine.executeOnStartHooks(context.Background(), task, job)

		// Should still return the original cancel error
		assert.Error(t, err)
		var returnedCancelErr *hook.CancelError
		assert.True(t, errors.As(err, &returnedCancelErr))

		mockJobQuery.AssertExpectations(t)
		mockHookExecutor.AssertExpectations(t)
	})

	t.Run("FatalError - UpdateJobStatus fails - still returns FatalError", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		mockHookExecutor := new(MockHookExecutor)

		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 0, mockHookExecutor)
		engine.logger = zap.NewNop()

		task := createTestTaskForHook()
		job := createTestJobForHook()

		hookID := uuid.New()
		fatalErr := &hook.FatalError{HookID: hookID, Cause: errors.New("fatal")}

		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnStart, nil).
			Return(fatalErr).Once()

		// UpdateJobStatus fails
		mockJobQuery.On("UpdateJobStatus", mock.Anything, job.ID, string(model.JobStatusFailed), fatalErr.Error()).
			Return((*ent.Job)(nil), errors.New("db error")).Once()

		// on_failure and on_end should still be called
		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnFailure, fatalErr).
			Return(nil).Once()
		mockHookExecutor.On("Execute", mock.Anything, task, job, model.HookEventOnEnd, fatalErr).
			Return(nil).Once()

		err := engine.executeOnStartHooks(context.Background(), task, job)

		// Should still return the original fatal error
		assert.Error(t, err)
		var returnedFatalErr *hook.FatalError
		assert.True(t, errors.As(err, &returnedFatalErr))

		mockJobQuery.AssertExpectations(t)
		mockHookExecutor.AssertExpectations(t)
	})
}

// TestNewSyncEngineWithHookExecutor tests that NewSyncEngine properly initializes with hookExecutor
func TestNewSyncEngineWithHookExecutor(t *testing.T) {
	t.Run("hookExecutor nil is allowed", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 4, nil)

		assert.NotNil(t, engine)
		assert.Nil(t, engine.hookExecutor)
	})

	t.Run("hookExecutor is stored correctly", func(t *testing.T) {
		mockJobQuery := new(MockJobQuery)
		mockHookExecutor := new(MockHookExecutor)

		engine := NewSyncEngine(mockJobQuery, nil, nil, t.TempDir(), false, 4, mockHookExecutor)

		assert.NotNil(t, engine)
		assert.NotNil(t, engine.hookExecutor)
		assert.Equal(t, mockHookExecutor, engine.hookExecutor)
	})
}

// Helper functions for hook tests
func createTestTaskForHook() *ent.Task {
	return &ent.Task{
		ID:           uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Name:         "test-task-hook",
		SourcePath:   "/local/path",
		RemotePath:   "remote:backup",
		Direction:    model.SyncDirectionUpload,
		ConnectionID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
	}
}

func createTestJobForHook() *ent.Job {
	return &ent.Job{
		ID:               uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		TaskID:           uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Status:           model.JobStatusRunning,
		Trigger:          model.JobTriggerManual,
		StartTime:        time.Date(2026, 1, 17, 10, 0, 0, 0, time.UTC),
		EndTime:          time.Time{},
		FilesTransferred: 0,
		BytesTransferred: 0,
		FilesDeleted:     0,
		ErrorCount:       0,
	}
}
