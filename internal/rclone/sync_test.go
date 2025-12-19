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

	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
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
		logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)
	}
	m.Run()
}

func (m *MockJobService) CreateJob(ctx context.Context, taskID uuid.UUID, trigger job.Trigger) (*ent.Job, error) {
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

func (m *MockJobService) UpdateJobStats(ctx context.Context, jobID uuid.UUID, files, bytes int64) (*ent.Job, error) {
	args := m.Called(ctx, jobID, files, bytes)
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

// TestPollStatsLogic tests the logic of pollStats using a mocked JobService
func TestPollStatsLogic(t *testing.T) {
	// 1. Setup Mock
	mockJobService := new(MockJobService)
	jobID := uuid.New()

	// 2. Setup SyncEngine
	engine := NewSyncEngine(mockJobService, t.TempDir())
	engine.logger = zap.NewNop() // Setup logger

	// Inject job into activeJobs (normally done in RunTask)
	engine.statsMu.Lock()
	engine.activeJobs[jobID] = JobProgress{}
	engine.statsMu.Unlock()

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
	engine := NewSyncEngine(mockJobService, t.TempDir())

	// Test case 1: Job ID exists in activeJobs
	jobID1 := uuid.New()
	expectedProgress1 := JobProgress{Transfers: 10, Bytes: 1024}
	engine.activeJobs[jobID1] = expectedProgress1

	progress, ok := engine.GetJobProgress(jobID1)
	assert.True(t, ok, "Should return true for existing job ID")
	assert.Equal(t, expectedProgress1, progress, "Should return the correct progress")

	// Test case 2: Job ID does not exist in activeJobs
	jobID2 := uuid.New()
	progress, ok = engine.GetJobProgress(jobID2)
	assert.False(t, ok, "Should return false for non-existing job ID")
	assert.Equal(t, JobProgress{}, progress, "Should return zero-value progress for non-existing job")

	// Test case 3: Empty activeJobs map
	engine.activeJobs = make(map[uuid.UUID]JobProgress)
	progress, ok = engine.GetJobProgress(jobID1)
	assert.False(t, ok, "Should return false when activeJobs is empty")
	assert.Equal(t, JobProgress{}, progress, "Should return zero-value progress when activeJobs is empty")
}

// TestGetConflictResolutionFromOptions tests all branches of getConflictResolutionFromOptions
func TestGetConflictResolutionFromOptions(t *testing.T) {
	tests := []struct {
		name            string
		options         map[string]any
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
			options:         map[string]any{},
			expectedResolve: "newer",
			expectedLoser:   "num",
		},
		{
			name: "resolution: newer",
			options: map[string]any{
				"conflict_resolution": "newer",
			},
			expectedResolve: "newer",
			expectedLoser:   "num",
		},
		{
			name: "resolution: local",
			options: map[string]any{
				"conflict_resolution": "local",
			},
			expectedResolve: "path1",
			expectedLoser:   "delete",
		},
		{
			name: "resolution: remote",
			options: map[string]any{
				"conflict_resolution": "remote",
			},
			expectedResolve: "path2",
			expectedLoser:   "delete",
		},
		{
			name: "resolution: both",
			options: map[string]any{
				"conflict_resolution": "both",
			},
			expectedResolve: "none",
			expectedLoser:   "num",
		},
		{
			name: "resolution: unknown - default to newer",
			options: map[string]any{
				"conflict_resolution": "unknown",
			},
			expectedResolve: "newer",
			expectedLoser:   "num",
		},
		{
			name: "resolution: wrong type - default",
			options: map[string]any{
				"conflict_resolution": 123,
			},
			expectedResolve: "newer",
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
	engine := NewSyncEngine(mockJobService, t.TempDir())
	engine.logger = zap.NewNop()

	jobID := uuid.New()
	testErr := assert.AnError

	// Expect UpdateJobStatus to be called
	mockJobService.On("UpdateJobStatus", mock.Anything, jobID, "failed", testErr.Error()).
		Return((*ent.Job)(nil), nil).Once()

	ctx := context.Background()
	engine.failJob(ctx, jobID, testErr)

	mockJobService.AssertExpectations(t)
}
