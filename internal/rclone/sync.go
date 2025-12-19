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

// JobProgress represents the current progress of a sync job.
type JobProgress struct {
	Transfers int64
	Bytes     int64
}

// SyncEngine handles file synchronization operations using rclone.
type SyncEngine struct {
	jobService ports.JobService
	logger     *zap.Logger
	workDir    string
	statsMu    sync.RWMutex
	activeJobs map[uuid.UUID]JobProgress
}

// NewSyncEngine creates a new SyncEngine instance.
func NewSyncEngine(jobService ports.JobService, dataDir string) *SyncEngine {
	workDir := filepath.Join(dataDir, "bisync_state")
	return &SyncEngine{
		jobService: jobService,
		logger:     logger.Named("sync.engine"),
		workDir:    workDir,
		activeJobs: make(map[uuid.UUID]JobProgress),
	}
}

// GetJobProgress returns the current progress of a running job.
func (e *SyncEngine) GetJobProgress(jobID uuid.UUID) (JobProgress, bool) {
	e.statsMu.RLock()
	defer e.statsMu.RUnlock()
	p, ok := e.activeJobs[jobID]
	return p, ok
}

// getConflictResolutionFromOptions extracts conflict resolution setting from task options.
// Returns the default value (PreferNewer) if not specified.
func getConflictResolutionFromOptions(options map[string]any) (bisync.Prefer, bisync.ConflictLoserAction) {
	conflictResolve := bisync.PreferNewer
	conflictLoser := bisync.ConflictLoserNumber // Default: rename conflicting files

	if options == nil {
		return conflictResolve, conflictLoser
	}

	resolution, ok := options["conflict_resolution"].(string)
	if !ok {
		return conflictResolve, conflictLoser
	}

	switch resolution {
	case "newer":
		// Keep newer file, rename the older one
		conflictResolve = bisync.PreferNewer
		conflictLoser = bisync.ConflictLoserNumber
	case "local":
		// Keep local (path1), delete remote
		conflictResolve = bisync.PreferPath1
		conflictLoser = bisync.ConflictLoserDelete
	case "remote":
		// Keep remote (path2), delete local
		conflictResolve = bisync.PreferPath2
		conflictLoser = bisync.ConflictLoserDelete
	case "both":
		// Keep both files, rename them with conflict suffix
		conflictResolve = bisync.PreferNone
		conflictLoser = bisync.ConflictLoserNumber
	default:
		// Default to newer
		conflictResolve = bisync.PreferNewer
		conflictLoser = bisync.ConflictLoserNumber
	}

	return conflictResolve, conflictLoser
}

// RunTask executes a sync task using the appropriate method based on task.Direction.
// Supports bidirectional sync using bisync, and one-way sync (upload/download) using rclone sync.
func (e *SyncEngine) RunTask(ctx context.Context, task *ent.Task, trigger job.Trigger) error {
	// Get connection name from task's connection edge (needed throughout function)
	if task.Edges.Connection == nil {
		return errs.ConstError("task connection edge not loaded")
	}
	connectionName := task.Edges.Connection.Name

	// 1. Create Job record
	jobEntity, err := e.jobService.CreateJob(ctx, task.ID, trigger)
	if err != nil {
		return errors.Join(errs.ErrSystem, errs.ConstError("failed to create job"), err)
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
		return errors.Join(errs.ErrSystem, errs.ConstError("failed to update job status"), err)
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
		e.pollStats(statsCtx, jobEntity.ID, task, jobEntity.StartTime)
	})

	// 5. Create Fs objects
	// Note: We assume remotes are already configured in rclone.conf
	// For local paths, NewFs works directly. For remotes, it uses the config.
	fSrc, err := fs.NewFs(statsCtx, task.SourcePath)
	if err != nil {
		e.failJob(ctx, jobEntity.ID, err)
		return err
	}

	f2Path := fmt.Sprintf("%s:%s", connectionName, task.RemotePath)
	fDst, err := fs.NewFs(statsCtx, f2Path)
	if err != nil {
		e.failJob(ctx, jobEntity.ID, err)
		return err
	}

	// 8. Run sync based on task direction
	var syncErr error
	switch task.Direction {
	case taskent.DirectionBidirectional:
		syncErr = e.runBidirectional(statsCtx, task, fSrc, fDst)
	case taskent.DirectionUpload:
		syncErr = e.runOneWay(statsCtx, fSrc, fDst)
	case taskent.DirectionDownload:
		syncErr = e.runOneWay(statsCtx, fDst, fSrc)
	default:
		syncErr = fmt.Errorf("unsupported sync direction: %s", task.Direction) //nolint:err113
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
			// Use a fresh context for DB operations since the original context is cancelled
			dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
			if _, updateErr := e.jobService.UpdateJobStatus(dbCtx, jobEntity.ID, string(job.StatusCancelled), "Task cancelled by user or shutdown"); updateErr != nil {
				e.logger.Error("Failed to update job status to cancelled", zap.Error(updateErr))
			}
			dbCancel()
			// Broadcast cancellation
			s := accounting.Stats(statsCtx)
			var files, bytes int64
			if s != nil {
				files, bytes = s.GetTransfers(), s.GetBytes()
			}
			e.broadcastJobUpdate(jobEntity.ID, task.ID, connectionName, string(job.StatusCancelled), jobEntity.StartTime, time.Now(), files, bytes)
			return syncErr
		}

		if _, updateErr := e.jobService.AddJobLog(ctx, jobEntity.ID, string(joblog.LevelError), string(joblog.WhatError), syncErr.Error(), 0); updateErr != nil {
			e.logger.Error("Failed to add job log for sync failure", zap.Error(updateErr))
		}

		e.logger.Error("Sync operation failed", zap.Error(syncErr))
		if _, updateErr := e.jobService.UpdateJobStatus(ctx, jobEntity.ID, string(job.StatusFailed), syncErr.Error()); updateErr != nil {
			e.logger.Error("Failed to update job status to failed", zap.Error(updateErr))
		}
		// Broadcast failure
		s := accounting.Stats(statsCtx)
		var files, bytes int64
		if s != nil {
			files, bytes = s.GetTransfers(), s.GetBytes()
		}
		e.broadcastJobUpdate(jobEntity.ID, task.ID, connectionName, string(job.StatusFailed), jobEntity.StartTime, time.Now(), files, bytes)
		return syncErr
	}

	// Update final stats
	s := accounting.Stats(statsCtx)
	var files, bytes int64
	if s != nil {
		files, bytes = s.GetTransfers(), s.GetBytes()
		if _, updateErr := e.jobService.UpdateJobStats(ctx, jobEntity.ID, files, bytes); updateErr != nil {
			e.logger.Error("Failed to update final job stats", zap.Error(updateErr))
		}
	}

	if _, updateErr := e.jobService.UpdateJobStatus(ctx, jobEntity.ID, string(job.StatusSuccess), ""); updateErr != nil {
		e.logger.Error("Failed to update job status to success", zap.Error(updateErr))
	}

	// Broadcast success
	e.broadcastJobUpdate(jobEntity.ID, task.ID, connectionName, string(job.StatusSuccess), jobEntity.StartTime, time.Now(), files, bytes)

	e.logger.Info("Sync task completed successfully", zap.Stringer("job_id", jobEntity.ID))

	return nil
}

func (e *SyncEngine) failJob(ctx context.Context, jobID uuid.UUID, err error) {
	e.logger.Error("Job failed during setup", zap.Error(err))
	_, _ = e.jobService.UpdateJobStatus(ctx, jobID, string(job.StatusFailed), err.Error())
}

// runBidirectional executes a bidirectional sync using bisync.
func (e *SyncEngine) runBidirectional(ctx context.Context, task *ent.Task, f1, f2 fs.Fs) error {
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

	// Get conflict resolution settings from task options
	conflictResolve, conflictLoser := getConflictResolutionFromOptions(task.Options)
	e.logger.Debug("Conflict resolution settings",
		zap.String("conflict_resolve", conflictResolve.String()),
		zap.String("conflict_loser", conflictLoser.String()),
	)

	// Prepare Bisync options
	opt := &bisync.Options{
		Resync:          resync,
		Workdir:         e.workDir,
		NoCleanup:       true, // Keep workdir for state
		Force:           true, // TODO: Expose as task option
		CheckAccess:     false,
		ConflictResolve: conflictResolve,
		ConflictLoser:   conflictLoser,
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
func (e *SyncEngine) pollStats(ctx context.Context, jobID uuid.UUID, task *ent.Task, startTime time.Time) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final stats update
			e.processStats(ctx, jobID, task, startTime)
			return
		case <-ticker.C:
			e.processStats(ctx, jobID, task, startTime)
		}
	}
}

// processStats is the core logic for polling rclone stats, creating logs, and updating progress.
func (e *SyncEngine) processStats(ctx context.Context, jobID uuid.UUID, task *ent.Task, startTime time.Time) {
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
			transfersToRemove = append(transfersToRemove, tr)

			// Handle failed transfers
			if snapshot.Error != nil {
				logsToSave = append(logsToSave, &ent.JobLog{
					Level: joblog.LevelError,
					What:  joblog.WhatError,
					Path:  snapshot.Name + ": " + snapshot.Error.Error(), //TODO: better error handling
					Size:  snapshot.Size,
					Time:  snapshot.CompletedAt,
				})
				continue
			}

			// Categorize and log based on operation type (snapshot.What)
			switch snapshot.What {
			case "deleting":
				// Log file deletion operations
				logsToSave = append(logsToSave, &ent.JobLog{
					Level: joblog.LevelInfo,
					What:  joblog.WhatDelete,
					Path:  snapshot.Name,
					Size:  snapshot.Size,
					Time:  snapshot.CompletedAt,
				})
			case "moving":
				// Log file move operations
				logsToSave = append(logsToSave, &ent.JobLog{
					Level: joblog.LevelInfo,
					What:  joblog.WhatMove,
					Path:  snapshot.Name,
					Size:  snapshot.Size,
					Time:  snapshot.CompletedAt,
				})
			case "checking", "hashing", "listing", "listing file - Path1", "listing file - Path2":
				// Skip pure check operations (e.g., MD5 verification, listing)
				continue
			case "transferring":
				what := joblog.WhatUpload
				if snapshot.SrcFs != task.SourcePath {
					what = joblog.WhatDownload
				}
				// Log successful transfers (including 0-byte files)
				logsToSave = append(logsToSave, &ent.JobLog{
					Level: joblog.LevelInfo,
					What:  what,
					Path:  snapshot.Name,
					Size:  snapshot.Size,
					Time:  snapshot.CompletedAt,
				})
			default:
				// Unknown operation type
				logsToSave = append(logsToSave, &ent.JobLog{
					Level: joblog.LevelWarning,
					What:  joblog.WhatUnknown,
					Size:  snapshot.Size,
					Time:  time.Now(),
				})
			}
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
	// Get connection name for broadcast
	connectionName := ""
	if task.Edges.Connection != nil {
		connectionName = task.Edges.Connection.Name
	}
	e.broadcastJobUpdate(jobID, task.ID, connectionName, string(job.StatusRunning), startTime, time.Time{}, s.GetTransfers(), s.GetBytes())
}

func (e *SyncEngine) broadcastJobUpdate(jobID, taskID uuid.UUID, remoteName, status string, startTime, endTime time.Time, files, bytes int64) {
	data := gin.H{
		"id":                jobID.String(), // Map job_id to id for frontend compatibility
		"job_id":            jobID.String(),
		"task_id":           taskID.String(),
		"files_transferred": files,
		"bytes_transferred": bytes,
		"remote_name":       remoteName,
		"status":            status,
		"start_time":        startTime,
	}

	if !endTime.IsZero() {
		data["end_time"] = endTime
	}

	sse.GetBroker().Submit(sse.Event{
		Type: "job_progress",
		Data: data,
	})
}

// getStatsInternals uses unsafe reflection to access private fields of rclone's StatsInfo.
// It returns the mutex and the slice of started transfers.
func getStatsInternals(s *accounting.StatsInfo) (*sync.RWMutex, *[]*accounting.Transfer, error) {
	statsVal := reflect.ValueOf(s).Elem()

	// Access 'mu'
	muField := statsVal.FieldByName("mu")
	if !muField.IsValid() {
		return nil, nil, errs.ConstError("field 'mu' not found in accounting.StatsInfo")
	}
	muPtr := unsafe.Pointer(muField.UnsafeAddr()) //nolint:gosec // G103: Intentional unsafe access to rclone internals, see function doc
	mu := (*sync.RWMutex)(muPtr)

	// Access 'startedTransfers'
	transfersField := statsVal.FieldByName("startedTransfers")
	if !transfersField.IsValid() {
		return nil, nil, errs.ConstError("field 'startedTransfers' not found in accounting.StatsInfo")
	}
	transfersPtr := unsafe.Pointer(transfersField.UnsafeAddr()) //nolint:gosec // G103: Intentional unsafe access to rclone internals, see function doc
	transfers := (*[]*accounting.Transfer)(transfersPtr)

	return mu, transfers, nil
}

var _ ports.SyncEngine = (*SyncEngine)(nil)
