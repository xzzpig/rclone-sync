package rclone

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/rclone/rclone/cmd/bisync"
	"github.com/rclone/rclone/cmd/bisync/bilib"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/accounting"
	rclonesync "github.com/rclone/rclone/fs/sync"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
	"github.com/xzzpig/rclone-sync/internal/core/ent/joblog"
	taskent "github.com/xzzpig/rclone-sync/internal/core/ent/task"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/api/sse"
)

type JobProgress struct {
	Transfers int64
	Bytes     int64
}

type SyncEngine struct {
	jobService ports.JobService
	logger     *zap.Logger
	workDir    string
	statsMu    sync.RWMutex
	activeJobs map[uuid.UUID]JobProgress
}

func NewSyncEngine(jobService ports.JobService, dataDir string) *SyncEngine {
	workDir := filepath.Join(dataDir, "bisync_state")
	return &SyncEngine{
		jobService: jobService,
		logger:     logger.L.Named("sync-engine"),
		workDir:    workDir,
		activeJobs: make(map[uuid.UUID]JobProgress),
	}
}

func (e *SyncEngine) GetJobProgress(jobID uuid.UUID) (JobProgress, bool) {
	e.statsMu.RLock()
	defer e.statsMu.RUnlock()
	p, ok := e.activeJobs[jobID]
	return p, ok
}

// RunTask executes a sync task using the appropriate method based on task.Direction.
// Supports bidirectional sync using bisync, and one-way sync (upload/download) using rclone sync.
func (e *SyncEngine) RunTask(ctx context.Context, task *ent.Task, trigger string) error {
	// 1. Create Job record
	jobEntity, err := e.jobService.CreateJob(ctx, task.ID, trigger)
	if err != nil {
		return errors.Join(errs.ErrSystem, errors.New("failed to create job"), err)
	}

	// Initialize in-memory progress
	e.statsMu.Lock()
	e.activeJobs[jobEntity.ID] = JobProgress{}
	e.statsMu.Unlock()

	// Ensure cleanup when done
	defer func() {
		e.statsMu.Lock()
		delete(e.activeJobs, jobEntity.ID)
		e.statsMu.Unlock()
	}()

	e.logger.Info("Starting sync task", zap.String("task", task.Name), zap.String("job_id", jobEntity.ID.String()))

	// 2. Update Job status to running
	_, err = e.jobService.UpdateJobStatus(ctx, jobEntity.ID, string(job.StatusRunning), "")
	if err != nil {
		return errors.Join(errs.ErrSystem, errors.New("failed to update job status"), err)
	}

	// 3. Prepare Rclone context with stats group
	// We use the job ID as the stats group key to isolate stats for this job
	statsCtx, statsCancel := context.WithCancel(ctx)
	defer statsCancel()

	// Initialize stats for this context
	statsCtx = accounting.WithStatsGroup(statsCtx, jobEntity.ID.String())
	accounting.Stats(statsCtx).SetMaxCompletedTransfers(-1) // Unlimited buffer, we manage it manually

	// 4. Start stats poller
	// This runs in the background and collects transfer events
	var wg sync.WaitGroup
	wg.Go(func() {
		e.pollStats(statsCtx, jobEntity.ID)
	})

	// 5. Create Fs objects
	// Note: We assume remotes are already configured in rclone.conf
	// For local paths, NewFs works directly. For remotes, it uses the config.
	fSrc, err := fs.NewFs(statsCtx, task.SourcePath)
	if err != nil {
		e.failJob(ctx, jobEntity.ID, err)
		return err
	}

	f2Path := fmt.Sprintf("%s:%s", task.RemoteName, task.RemotePath)
	fDst, err := fs.NewFs(statsCtx, f2Path)
	if err != nil {
		e.failJob(ctx, jobEntity.ID, err)
		return err
	}

	// 8. Run sync based on task direction
	var syncErr error
	switch task.Direction {
	case taskent.DirectionBidirectional:
		syncErr = e.runBidirectional(statsCtx, fSrc, fDst)
	case taskent.DirectionUpload:
		syncErr = e.runOneWay(statsCtx, fSrc, fDst)
	case taskent.DirectionDownload:
		syncErr = e.runOneWay(statsCtx, fDst, fSrc)
	default:
		syncErr = fmt.Errorf("unsupported sync direction: %s", task.Direction)
	}

	// 9. Wait for poller to finish (it stops when statsCtx is cancelled or done)
	// We cancel statsCtx after sync returns to stop the poller loop
	statsCancel()
	wg.Wait()

	// 10. Finalize Job
	if syncErr != nil {
		// Check if the error is due to context cancellation
		if errors.Is(ctx.Err(), context.Canceled) {
			e.logger.Info("Sync task cancelled", zap.Stringer("job_id", jobEntity.ID))
			if _, updateErr := e.jobService.UpdateJobStatus(ctx, jobEntity.ID, string(job.StatusCancelled), "Task cancelled by user or shutdown"); updateErr != nil {
				e.logger.Error("Failed to update job status to cancelled", zap.Error(updateErr))
			}
			return syncErr
		}

		e.logger.Error("Sync operation failed", zap.Error(syncErr))
		if _, updateErr := e.jobService.UpdateJobStatus(ctx, jobEntity.ID, string(job.StatusFailed), syncErr.Error()); updateErr != nil {
			e.logger.Error("Failed to update job status to failed", zap.Error(updateErr))
		}
		return syncErr
	}

	// Update final stats
	s := accounting.Stats(statsCtx)
	if s != nil {
		if _, updateErr := e.jobService.UpdateJobStats(ctx, jobEntity.ID, s.GetTransfers(), s.GetBytes()); updateErr != nil {
			e.logger.Error("Failed to update final job stats", zap.Error(updateErr))
		}
	}

	if _, updateErr := e.jobService.UpdateJobStatus(ctx, jobEntity.ID, string(job.StatusSuccess), ""); updateErr != nil {
		e.logger.Error("Failed to update job status to success", zap.Error(updateErr))
	}
	e.logger.Info("Sync task completed successfully", zap.Stringer("job_id", jobEntity.ID))

	return nil
}

func (e *SyncEngine) failJob(ctx context.Context, jobID uuid.UUID, err error) {
	e.logger.Error("Job failed during setup", zap.Error(err))
	_, _ = e.jobService.UpdateJobStatus(ctx, jobID, string(job.StatusFailed), err.Error())
}

// runBidirectional executes a bidirectional sync using bisync.
func (e *SyncEngine) runBidirectional(ctx context.Context, f1, f2 fs.Fs) error {
	// Determine Resync necessity
	// Calculate base path and listing file names to check if they exist
	basePath := bilib.BasePath(ctx, e.workDir, f1, f2)
	listing1 := basePath + ".path1.lst"
	listing2 := basePath + ".path2.lst"

	resync := false
	if !bilib.FileExists(listing1) || !bilib.FileExists(listing2) {
		e.logger.Info("Listing files not found, forcing Resync")
		resync = true
	}

	// Prepare Bisync options
	opt := &bisync.Options{
		Resync:      resync,
		Workdir:     e.workDir,
		NoCleanup:   true, // Keep workdir for state
		CheckAccess: false,
		// TODO: Map other options from task.Options JSON
	}

	// Run Bisync
	return bisync.Bisync(ctx, f1, f2, opt)
}

// runOneWay executes a one-way sync using rclone sync.
func (e *SyncEngine) runOneWay(ctx context.Context, fSrc, fDst fs.Fs) error {
	return rclonesync.Sync(ctx, fDst, fSrc, true)
}

// pollStats monitors the rclone stats and persists logs to the database.
//
// WARNING: This method uses UNSAFE REFLECTION to access private fields ('mu' and 'startedTransfers')
// of the rclone accounting.StatsInfo struct. This is necessary because rclone does not expose
// completed transfers or individual transfer details via its public API in a way that allows
// granular logging per file transfer after the fact, or easy consumption of such events.
//
// NOTE: Since we use rclone as a dependency, the version is stable, fixed, and controllable.
// Unit tests ensure that this logic works correctly with the current rclone version.
//
// Risks:
//  1. Compatibility: If rclone changes the internal structure of StatsInfo (renames fields, changes types),
//     this code WILL PANIC or fail silently. This is verified by TestPollStatsReflection in sync_test.go.
//  2. Concurrency: We manually lock the mutex ('mu') acquired via reflection. If rclone changes its locking
//     strategy, this could lead to race conditions or deadlocks.
//
// Future: If rclone adds a proper event bus or callback system for transfers, this should be replaced immediately.
func (e *SyncEngine) pollStats(ctx context.Context, jobID uuid.UUID) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final flush to capture any remaining completed transfers
			e.processStats(ctx, jobID)
			return
		case <-ticker.C:
			e.processStats(ctx, jobID)
		}
	}
}

// processStats is the core logic for polling rclone stats, creating logs, and updating progress.
func (e *SyncEngine) processStats(ctx context.Context, jobID uuid.UUID) {
	s := accounting.Stats(ctx)
	if s == nil {
		return
	}

	statsInnerMu, transfers, err := getStatsInternals(s)
	if err != nil {
		e.logger.Debug("Failed to get stats internals", zap.Error(err))
		return
	}

	statsInnerMu.Lock()

	var transfersToRemove []*accounting.Transfer
	var logsToSave []*ent.JobLog

	// Find completed transfers
	for _, tr := range *transfers {
		if tr.IsDone() {
			snapshot := tr.Snapshot()

			// Create log entry
			if snapshot.Bytes > 0 {
				logsToSave = append(logsToSave, &ent.JobLog{
					Level:   joblog.LevelInfo,
					Message: fmt.Sprintf("Transferred %s (%d bytes)", snapshot.Name, snapshot.Bytes),
					Path:    snapshot.Name,
					Time:    time.Now(),
				})
			}

			transfersToRemove = append(transfersToRemove, tr)
		}
	}

	statsInnerMu.Unlock()

	// Persist logs
	if len(logsToSave) > 0 {
		// Use a separate context for DB operations to ensure they complete even if the job context is cancelling
		dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := e.jobService.AddJobLogsBatch(dbCtx, jobID, logsToSave)
		if err != nil {
			e.logger.Error("Failed to save job logs", zap.Error(err))
		}
		cancel()
	}

	// Remove processed transfers from stats
	for _, tr := range transfersToRemove {
		s.RemoveTransfer(tr)
	}

	// Update in-memory job progress for realtime monitoring
	e.statsMu.Lock()
	e.activeJobs[jobID] = JobProgress{
		Transfers: s.GetTransfers(),
		Bytes:     s.GetBytes(),
	}
	e.statsMu.Unlock()

	// Broadcast progress update
	sse.GetBroker().Submit(sse.Event{
		Type: "job_progress",
		Data: gin.H{
			"job_id":            jobID.String(),
			"files_transferred": s.GetTransfers(),
			"bytes_transferred": s.GetBytes(),
		},
	})
}

// getStatsInternals uses unsafe reflection to access private fields of rclone's StatsInfo.
// It returns the mutex and the slice of started transfers.
func getStatsInternals(s *accounting.StatsInfo) (*sync.RWMutex, *[]*accounting.Transfer, error) {
	statsVal := reflect.ValueOf(s).Elem()

	// Access 'mu'
	muField := statsVal.FieldByName("mu")
	if !muField.IsValid() {
		return nil, nil, errors.New("field 'mu' not found in accounting.StatsInfo")
	}
	muPtr := unsafe.Pointer(muField.UnsafeAddr())
	mu := (*sync.RWMutex)(muPtr)

	// Access 'startedTransfers'
	transfersField := statsVal.FieldByName("startedTransfers")
	if !transfersField.IsValid() {
		return nil, nil, errors.New("field 'startedTransfers' not found in accounting.StatsInfo")
	}
	transfersPtr := unsafe.Pointer(transfersField.UnsafeAddr())
	transfers := (*[]*accounting.Transfer)(transfersPtr)

	return mu, transfers, nil
}
