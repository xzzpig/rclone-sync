package rclone

import (
	"context"
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
	"github.com/xzzpig/rclone-sync/internal/core/logger"
)

// MockJobService is a mock for services.JobService
// We only need to mock the methods used by pollStats
type MockJobService struct {
	mock.Mock
}

func TestMain(m *testing.M) {
	// Initialize logger for tests
	{ // logger init block
		logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug, nil)
	}
	m.Run()
}

func (m *MockJobService) CreateJob(ctx context.Context, taskID uuid.UUID, trigger model.JobTrigger) (*ent.Job, error) {
	args := m.Called(ctx, taskID, trigger)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string, errStr string) (*ent.Job, error) {
	args := m.Called(ctx, jobID, status, errStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobService) AddJobLogsBatch(ctx context.Context, jobID uuid.UUID, logs []*ent.JobLog) error {
	args := m.Called(ctx, jobID, logs)
	return args.Error(0)
}

func (m *MockJobService) UpdateJobStats(ctx context.Context, jobID uuid.UUID, files, bytes, filesDeleted, errorCount int64) (*ent.Job, error) {
	args := m.Called(ctx, jobID, files, bytes, filesDeleted, errorCount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobService) AddJobLog(ctx context.Context, jobID uuid.UUID, level, what, path string, size int64) (*ent.JobLog, error) {
	args := m.Called(ctx, jobID, level, what, path, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.JobLog), args.Error(1)
}

func (m *MockJobService) GetJob(ctx context.Context, jobID uuid.UUID) (*ent.Job, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobService) GetLastJobByTaskID(ctx context.Context, taskID uuid.UUID) (*ent.Job, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobService) ListJobs(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID, limit, offset int) ([]*ent.Job, error) {
	args := m.Called(ctx, taskID, connectionID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ent.Job), args.Error(1)
}

func (m *MockJobService) GetJobWithLogs(ctx context.Context, jobID uuid.UUID) (*ent.Job, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Job), args.Error(1)
}

func (m *MockJobService) CountJobs(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID) (int, error) {
	args := m.Called(ctx, taskID, connectionID)
	return args.Int(0), args.Error(1)
}

func (m *MockJobService) ListJobLogs(ctx context.Context, connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string, limit, offset int) ([]*ent.JobLog, error) {
	args := m.Called(ctx, connectionID, taskID, jobID, level, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ent.JobLog), args.Error(1)
}

func (m *MockJobService) CountJobLogs(ctx context.Context, connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string) (int, error) {
	args := m.Called(ctx, connectionID, taskID, jobID, level)
	return args.Int(0), args.Error(1)
}

func (m *MockJobService) DeleteJob(ctx context.Context, jobID uuid.UUID) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

// TestPollStatsLogic tests the logic of pollStats using a mocked JobService
func TestPollStatsLogic(t *testing.T) {
	// 1. Setup Mock
	mockJobService := new(MockJobService)
	jobID := uuid.New()

	// 2. Setup SyncEngine
	engine := NewSyncEngine(mockJobService, nil, nil, t.TempDir(), false, 0)
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
	mockJobService := new(MockJobService)
	engine := NewSyncEngine(mockJobService, nil, nil, t.TempDir(), false, 0)

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
	mockJobService := new(MockJobService)
	engine := NewSyncEngine(mockJobService, nil, nil, t.TempDir(), false, 0)
	engine.logger = zap.NewNop()

	jobID := uuid.New()
	testErr := assert.AnError

	// Expect UpdateJobStatus to be called
	mockJobService.On("UpdateJobStatus", mock.Anything, jobID, string(model.JobStatusFailed), testErr.Error()).
		Return((*ent.Job)(nil), nil).Once()

	ctx := context.Background()
	engine.failJob(ctx, jobID, testErr)

	mockJobService.AssertExpectations(t)
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
	totalTransfers, totalBytes := getTotalStats(stats)

	// Initially, both should be 0 (no transfers started)
	assert.Equal(t, int64(0), totalTransfers, "Initial totalTransfers should be 0")
	assert.Equal(t, int64(0), totalBytes, "Initial totalBytes should be 0")
}

// TestGetTotalStats_NilStats tests getTotalStats with nil stats.
func TestGetTotalStats_NilStats(t *testing.T) {
	totalTransfers, totalBytes := getTotalStats(nil)
	assert.Equal(t, int64(0), totalTransfers, "totalTransfers should be 0 for nil stats")
	assert.Equal(t, int64(0), totalBytes, "totalBytes should be 0 for nil stats")
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

// TestShouldDeleteEmptyJob tests the shouldDeleteEmptyJob helper function.
func TestShouldDeleteEmptyJob(t *testing.T) {
	tests := []struct {
		name                string
		autoDeleteEmptyJobs bool
		status              model.JobStatus
		filesTransferred    int
		bytesTransferred    int64
		filesDeleted        int
		errorCount          int
		expectedDelete      bool
	}{
		{
			name:                "enabled+success+no_activity->delete",
			autoDeleteEmptyJobs: true,
			status:              model.JobStatusSuccess,
			filesTransferred:    0,
			bytesTransferred:    0,
			filesDeleted:        0,
			errorCount:          0,
			expectedDelete:      true,
		},
		{
			name:                "enabled+success+has_activity->keep",
			autoDeleteEmptyJobs: true,
			status:              model.JobStatusSuccess,
			filesTransferred:    5,
			bytesTransferred:    1024,
			filesDeleted:        0,
			errorCount:          0,
			expectedDelete:      false,
		},
		{
			name:                "enabled+success+only_bytes_transferred->keep",
			autoDeleteEmptyJobs: true,
			status:              model.JobStatusSuccess,
			filesTransferred:    0,
			bytesTransferred:    1024,
			filesDeleted:        0,
			errorCount:          0,
			expectedDelete:      false,
		},
		{
			name:                "enabled+success+only_files_transferred->keep",
			autoDeleteEmptyJobs: true,
			status:              model.JobStatusSuccess,
			filesTransferred:    5,
			bytesTransferred:    0,
			filesDeleted:        0,
			errorCount:          0,
			expectedDelete:      false,
		},
		{
			name:                "enabled+success+only_files_deleted->keep",
			autoDeleteEmptyJobs: true,
			status:              model.JobStatusSuccess,
			filesTransferred:    0,
			bytesTransferred:    0,
			filesDeleted:        3,
			errorCount:          0,
			expectedDelete:      false,
		},
		{
			name:                "enabled+success+only_errors->keep",
			autoDeleteEmptyJobs: true,
			status:              model.JobStatusSuccess,
			filesTransferred:    0,
			bytesTransferred:    0,
			filesDeleted:        0,
			errorCount:          2,
			expectedDelete:      false,
		},
		{
			name:                "enabled+success+deletes_and_errors->keep",
			autoDeleteEmptyJobs: true,
			status:              model.JobStatusSuccess,
			filesTransferred:    0,
			bytesTransferred:    0,
			filesDeleted:        1,
			errorCount:          1,
			expectedDelete:      false,
		},
		{
			name:                "enabled+failed+no_activity->keep",
			autoDeleteEmptyJobs: true,
			status:              model.JobStatusFailed,
			filesTransferred:    0,
			bytesTransferred:    0,
			filesDeleted:        0,
			errorCount:          0,
			expectedDelete:      false,
		},
		{
			name:                "enabled+cancelled+no_activity->keep",
			autoDeleteEmptyJobs: true,
			status:              model.JobStatusCancelled,
			filesTransferred:    0,
			bytesTransferred:    0,
			filesDeleted:        0,
			errorCount:          0,
			expectedDelete:      false,
		},
		{
			name:                "disabled+success+no_activity->keep",
			autoDeleteEmptyJobs: false,
			status:              model.JobStatusSuccess,
			filesTransferred:    0,
			bytesTransferred:    0,
			filesDeleted:        0,
			errorCount:          0,
			expectedDelete:      false,
		},
		{
			name:                "disabled+success+has_activity->keep",
			autoDeleteEmptyJobs: false,
			status:              model.JobStatusSuccess,
			filesTransferred:    5,
			bytesTransferred:    1024,
			filesDeleted:        0,
			errorCount:          0,
			expectedDelete:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldDelete := shouldDeleteEmptyJob(tt.autoDeleteEmptyJobs, tt.status, tt.filesTransferred, tt.bytesTransferred, tt.filesDeleted, tt.errorCount)
			assert.Equal(t, tt.expectedDelete, shouldDelete, "shouldDeleteEmptyJob result mismatch")
		})
	}
}
