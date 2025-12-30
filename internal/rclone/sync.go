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
	"github.com/rclone/rclone/fs/filter"
	rclonesync "github.com/rclone/rclone/fs/sync"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"github.com/xzzpig/rclone-sync/internal/i18n"
	"github.com/xzzpig/rclone-sync/internal/utils"
	"go.uber.org/zap"
)

// SyncOptions contains configuration options for sync operations.
// These options are applied per-task and can override global defaults.
type SyncOptions struct {
	// Filters contains rclone filter rules (e.g., "- node_modules/**", "+ **").
	// Each rule should start with "+" (include) or "-" (exclude) followed by a pattern.
	// Rules are applied in order; the first matching rule wins.
	Filters []string

	// NoDelete prevents deletion of files in the destination that don't exist in the source.
	// Only applies to one-way sync (Upload/Download). Ignored for bidirectional sync.
	// When true, uses CopyDir instead of Sync.
	NoDelete bool

	// Transfers is the number of parallel file transfers (1-64).
	// If 0, the global default from config is used.
	Transfers int
}

// SyncEngine handles file synchronization operations using rclone.
type SyncEngine struct {
	jobService          ports.JobService
	jobProgressBus      *subscription.JobProgressBus
	transferProgressBus *subscription.TransferProgressBus
	logger              *zap.Logger
	workDir             string
	autoDeleteEmptyJobs bool
	defaultTransfers    int // Global default for parallel transfers (from config)
	statsMu             sync.RWMutex
	lastEvents          map[uuid.UUID]*model.JobProgressEvent
	lastTransferEvents  map[uuid.UUID]*model.TransferProgressEvent
}

// DefaultTransfers is the built-in default for parallel transfers when not configured.
const DefaultTransfers = 4

// NewSyncEngine creates a new SyncEngine instance.
// defaultTransfers specifies the global default for parallel transfers (from config).
// If defaultTransfers is 0 or negative, DefaultTransfers (4) will be used.
func NewSyncEngine(jobService ports.JobService, jobProgressBus *subscription.JobProgressBus, transferProgressBus *subscription.TransferProgressBus, dataDir string, autoDeleteEmptyJobs bool, defaultTransfers int) *SyncEngine {
	workDir := filepath.Join(dataDir, "bisync_state")
	if defaultTransfers <= 0 {
		defaultTransfers = DefaultTransfers
	}
	return &SyncEngine{
		jobService:          jobService,
		jobProgressBus:      jobProgressBus,
		transferProgressBus: transferProgressBus,
		logger:              logger.Named("sync.engine"),
		workDir:             workDir,
		autoDeleteEmptyJobs: autoDeleteEmptyJobs,
		defaultTransfers:    defaultTransfers,
		lastEvents:          make(map[uuid.UUID]*model.JobProgressEvent),
		lastTransferEvents:  make(map[uuid.UUID]*model.TransferProgressEvent),
	}
}

// GetJobProgress returns the current progress of a running job.
// Returns the latest cached JobProgressEvent if the job is running, nil otherwise.
func (e *SyncEngine) GetJobProgress(jobID uuid.UUID) *model.JobProgressEvent {
	e.statsMu.RLock()
	defer e.statsMu.RUnlock()
	return e.lastEvents[jobID]
}

// getConflictResolutionFromOptions extracts conflict resolution setting from task options.
// Returns the default value (PreferNewer) if not specified.
func getConflictResolutionFromOptions(options *model.TaskSyncOptions) (bisync.Prefer, bisync.ConflictLoserAction) {
	conflictResolve := bisync.PreferNewer
	conflictLoser := bisync.ConflictLoserNumber // Default: rename conflicting files

	if options == nil || options.ConflictResolution == nil {
		return conflictResolve, conflictLoser
	}

	switch *options.ConflictResolution {
	case model.ConflictResolutionNewer:
		// Keep newer file, rename the older one
		conflictResolve = bisync.PreferNewer
		conflictLoser = bisync.ConflictLoserNumber
	case model.ConflictResolutionLocal:
		// Keep local (path1), delete remote
		conflictResolve = bisync.PreferPath1
		conflictLoser = bisync.ConflictLoserDelete
	case model.ConflictResolutionRemote:
		// Keep remote (path2), delete local
		conflictResolve = bisync.PreferPath2
		conflictLoser = bisync.ConflictLoserDelete
	case model.ConflictResolutionBoth:
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
func (e *SyncEngine) RunTask(ctx context.Context, task *ent.Task, trigger model.JobTrigger) error {
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

	// Ensure cleanup of cached events when done
	defer func() {
		e.statsMu.Lock()
		delete(e.lastEvents, jobEntity.ID)
		delete(e.lastTransferEvents, jobEntity.ID)
		e.statsMu.Unlock()
	}()

	e.logger.Info("Starting sync task", zap.String("task", task.Name), zap.String("job_id", jobEntity.ID.String()))

	// 2. Update Job status to running
	_, err = e.jobService.UpdateJobStatus(ctx, jobEntity.ID, string(model.JobStatusRunning), "")
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
	// For source (local paths), use GetFs with empty remote to skip caching (per FR-009).
	// For destination (remote), use GetFs with remote name to leverage Fs cache.
	fSrc, err := GetFs(statsCtx, "", task.SourcePath)
	if err != nil {
		e.failJob(ctx, jobEntity.ID, err)
		return err
	}

	// For remote destinations, use cached Fs to avoid repeated connection setup
	fDst, err := GetFs(statsCtx, connectionName, task.RemotePath)
	if err != nil {
		e.failJob(ctx, jobEntity.ID, err)
		return err
	}

	// 6. Extract sync options from task
	syncOpts := getSyncOptionsFromTask(task.Options)
	e.logger.Debug("Sync options extracted",
		zap.Strings("filters", syncOpts.Filters),
		zap.Bool("noDelete", syncOpts.NoDelete),
		zap.Int("transfers", syncOpts.Transfers),
	)

	// 7. Apply common rclone config (transfers) to context
	transfers := determineTransfers(syncOpts.Transfers, e.defaultTransfers)
	statsCtx, rcloneCfg := fs.AddConfig(statsCtx)
	rcloneCfg.Transfers = transfers
	e.logger.Debug("Transfers configured", zap.Int("transfers", transfers))

	// 8. Run sync based on task direction
	var syncErr error
	switch task.Direction {
	case model.SyncDirectionBidirectional:
		syncErr = e.runBidirectional(statsCtx, task, fSrc, fDst, syncOpts)
	case model.SyncDirectionUpload:
		syncErr = e.runOneWay(statsCtx, fSrc, fDst, syncOpts)
	case model.SyncDirectionDownload:
		syncErr = e.runOneWay(statsCtx, fDst, fSrc, syncOpts)
	default:
		syncErr = i18n.NewI18nError(i18n.ErrInvalidInput).WithCause(fmt.Errorf("unsupported sync direction: %s", task.Direction)) //nolint:err113
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
			if _, updateErr := e.jobService.UpdateJobStatus(dbCtx, jobEntity.ID, string(model.JobStatusCancelled), "Task cancelled by user or shutdown"); updateErr != nil {
				e.logger.Error("Failed to update job status to cancelled", zap.Error(updateErr))
			}
			dbCancel()
			// Broadcast cancellation
			s := accounting.Stats(statsCtx)
			var files, bytes int64
			if s != nil {
				files, bytes = s.GetTransfers(), s.GetBytes()
			}
			e.broadcastJobUpdate(&model.JobProgressEvent{
				JobID:            jobEntity.ID,
				TaskID:           task.ID,
				ConnectionID:     task.Edges.Connection.ID,
				Status:           model.JobStatusCancelled,
				FilesTransferred: int(files),
				BytesTransferred: bytes,
				StartTime:        jobEntity.StartTime,
				EndTime:          func() *time.Time { t := time.Now(); return &t }(),
			})
			return syncErr
		}

		if _, updateErr := e.jobService.AddJobLog(ctx, jobEntity.ID, string(model.LogLevelError), string(model.LogActionError), syncErr.Error(), 0); updateErr != nil {
			e.logger.Error("Failed to add job log for sync failure", zap.Error(updateErr))
		}

		e.logger.Error("Sync operation failed", zap.Error(syncErr))
		if _, updateErr := e.jobService.UpdateJobStatus(ctx, jobEntity.ID, string(model.JobStatusFailed), syncErr.Error()); updateErr != nil {
			e.logger.Error("Failed to update job status to failed", zap.Error(updateErr))
		}
		// Broadcast failure
		s := accounting.Stats(statsCtx)
		var files, bytes int64
		if s != nil {
			files, bytes = s.GetTransfers(), s.GetBytes()
		}
		e.broadcastJobUpdate(&model.JobProgressEvent{
			JobID:            jobEntity.ID,
			TaskID:           task.ID,
			ConnectionID:     task.Edges.Connection.ID,
			Status:           model.JobStatusFailed,
			FilesTransferred: int(files),
			BytesTransferred: bytes,
			StartTime:        jobEntity.StartTime,
			EndTime:          func() *time.Time { t := time.Now(); return &t }(),
		})
		return syncErr
	}

	// Update final stats
	s := accounting.Stats(statsCtx)
	var files, bytes, filesDeleted, errorCount int64
	if s != nil {
		files, bytes, filesDeleted, errorCount = s.GetTransfers(), s.GetBytes(), s.GetDeletes(), s.GetErrors()
		if _, updateErr := e.jobService.UpdateJobStats(ctx, jobEntity.ID, files, bytes, filesDeleted, errorCount); updateErr != nil {
			e.logger.Error("Failed to update final job stats", zap.Error(updateErr))
		}
	}

	if _, updateErr := e.jobService.UpdateJobStatus(ctx, jobEntity.ID, string(model.JobStatusSuccess), ""); updateErr != nil {
		e.logger.Error("Failed to update job status to success", zap.Error(updateErr))
	}

	// Broadcast success
	e.broadcastJobUpdate(&model.JobProgressEvent{
		JobID:            jobEntity.ID,
		TaskID:           task.ID,
		ConnectionID:     task.Edges.Connection.ID,
		Status:           model.JobStatusSuccess,
		FilesTransferred: int(files),
		BytesTransferred: bytes,
		StartTime:        jobEntity.StartTime,
		EndTime:          func() *time.Time { t := time.Now(); return &t }(),
	})

	e.logger.Info("Sync task completed successfully", zap.Stringer("job_id", jobEntity.ID))

	// Auto-delete empty jobs if configured
	if shouldDeleteEmptyJob(e.autoDeleteEmptyJobs, model.JobStatusSuccess, int(files), bytes, int(filesDeleted), int(errorCount)) {
		e.logger.Debug("Auto-deleting empty job", zap.Stringer("job_id", jobEntity.ID))
		if err := e.jobService.DeleteJob(ctx, jobEntity.ID); err != nil {
			// Log warning but don't fail the task - the job has already succeeded
			e.logger.Warn("Failed to auto-delete empty job", zap.Stringer("job_id", jobEntity.ID), zap.Error(err))
		} else {
			e.logger.Debug("Successfully auto-deleted empty job", zap.Stringer("job_id", jobEntity.ID))
		}
	}

	return nil
}

func (e *SyncEngine) failJob(ctx context.Context, jobID uuid.UUID, err error) {
	e.logger.Error("Job failed during setup", zap.Error(err))
	_, _ = e.jobService.UpdateJobStatus(ctx, jobID, string(model.JobStatusFailed), err.Error())
}

// shouldDeleteEmptyJob determines if a job should be deleted based on its configuration and result.
// A job is considered "empty" if:
// - filesTransferred = 0 (no files were transferred)
// - bytesTransferred = 0 (no bytes were transferred)
// - filesDeleted = 0 (no files were deleted)
// - errorCount = 0 (no errors occurred)
// - status = SUCCESS (the job completed successfully)
// Failed jobs are always kept for debugging purposes.
func shouldDeleteEmptyJob(autoDeleteEmptyJobs bool, status model.JobStatus, filesTransferred int, bytesTransferred int64, filesDeleted int, errorCount int) bool {
	// If auto-delete is disabled, never delete
	if !autoDeleteEmptyJobs {
		return false
	}

	// Only delete successful jobs
	if status != model.JobStatusSuccess {
		return false
	}

	// Check if job had any activity (transfers, bytes, deletes, or errors)
	if filesTransferred > 0 || bytesTransferred > 0 || filesDeleted > 0 || errorCount > 0 {
		return false
	}

	// Job is empty and successful, delete it
	return true
}

// getSyncOptionsFromTask extracts SyncOptions from task.Options.
// Returns default SyncOptions if options are not set or invalid.
func getSyncOptionsFromTask(options *model.TaskSyncOptions) SyncOptions {
	opts := SyncOptions{}

	if options == nil {
		return opts
	}

	// Extract filters
	opts.Filters = options.Filters

	// Extract noDelete
	if options.NoDelete != nil {
		opts.NoDelete = *options.NoDelete
	}

	// Extract transfers
	if options.Transfers != nil {
		opts.Transfers = *options.Transfers
	}

	return opts
}

// determineTransfers returns the effective transfers count using three-level fallback:
// 1. Task-level value (opts.Transfers) if > 0
// 2. Global config value (defaultTransfers) if > 0
// 3. Built-in default (DefaultTransfers = 4)
func determineTransfers(taskTransfers, defaultTransfers int) int {
	if taskTransfers > 0 {
		return taskTransfers
	}
	if defaultTransfers > 0 {
		return defaultTransfers
	}
	return DefaultTransfers
}

// applyFilterRules creates a filter from rules and injects it into the context.
// Returns the modified context and any error encountered.
func applyFilterRules(ctx context.Context, rules []string) (context.Context, error) {
	if len(rules) == 0 {
		return ctx, nil
	}

	fi, err := createFilterFromRules(rules)
	if err != nil {
		return ctx, err
	}

	return filter.ReplaceConfig(ctx, fi), nil
}

// runBidirectional executes a bidirectional sync using bisync.
// It applies SyncOptions including filters.
// Note: noDelete is ignored for bidirectional sync as deletion propagation is inherent to bisync.
// Note: transfers setting is applied in RunTask before calling this method.
func (e *SyncEngine) runBidirectional(ctx context.Context, task *ent.Task, f1, f2 fs.Fs, opts SyncOptions) error {
	// Apply filter rules if specified
	var err error
	if len(opts.Filters) > 0 {
		ctx, err = applyFilterRules(ctx, opts.Filters)
		if err != nil {
			return i18n.NewI18nError(i18n.ErrSyncFailed).WithCause(err)
		}
	}

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
		Recover:         true,
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
// It applies SyncOptions including filters and noDelete.
// Note: transfers setting is applied in RunTask before calling this method.
//
// Parameters:
//   - fSrc: source filesystem (files to copy from)
//   - fDst: destination filesystem (files to copy to)
//   - opts: sync options including filters and noDelete flag
func (e *SyncEngine) runOneWay(ctx context.Context, fSrc, fDst fs.Fs, opts SyncOptions) error {
	// Apply filter rules if specified
	var err error
	if len(opts.Filters) > 0 {
		ctx, err = applyFilterRules(ctx, opts.Filters)
		if err != nil {
			return i18n.NewI18nError(i18n.ErrSyncFailed).WithCause(err)
		}
	}

	// Use CopyDir instead of Sync when noDelete is true
	// CopyDir copies from src to dst without deleting existing files
	if opts.NoDelete {
		return rclonesync.CopyDir(ctx, fDst, fSrc, true) // dst = fDst, src = fSrc
	}
	// Sync makes dst identical to src, including deletions
	return rclonesync.Sync(ctx, fDst, fSrc, true) // dst = fDst, src = fSrc
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
	var activeTransfers []*model.TransferItem

	e.logger.Debug("Processing stats", zap.Any("transfers", *transfers))

	// Process all transfers: collect completed for logging/removal, active for progress broadcast
	for _, tr := range *transfers {
		if tr.IsDone() {
			snapshot := tr.Snapshot()
			transfersToRemove = append(transfersToRemove, tr)

			// Handle failed transfers
			if snapshot.Error != nil {
				logsToSave = append(logsToSave, &ent.JobLog{
					Level: model.LogLevelError,
					What:  model.LogActionError,
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
					Level: model.LogLevelInfo,
					What:  model.LogActionDelete,
					Path:  snapshot.Name,
					Size:  snapshot.Size,
					Time:  snapshot.CompletedAt,
				})
			case "moving":
				// Log file move operations
				logsToSave = append(logsToSave, &ent.JobLog{
					Level: model.LogLevelInfo,
					What:  model.LogActionMove,
					Path:  snapshot.Name,
					Size:  snapshot.Size,
					Time:  snapshot.CompletedAt,
				})
			case "checking", "hashing", "listing", "listing file - Path1", "listing file - Path2":
				// Skip pure check operations (e.g., MD5 verification, listing)
				continue
			case "transferring":
				what := model.LogActionUpload
				if snapshot.SrcFs != task.SourcePath {
					what = model.LogActionDownload
				}
				// Log successful transfers (including 0-byte files)
				logsToSave = append(logsToSave, &ent.JobLog{
					Level: model.LogLevelInfo,
					What:  what,
					Path:  snapshot.Name,
					Size:  snapshot.Size,
					Time:  snapshot.CompletedAt,
				})
				// Include completed transfers in broadcast (bytes == size signals completion to frontend)
				activeTransfers = append(activeTransfers, &model.TransferItem{
					Name:  snapshot.Name,
					Size:  snapshot.Size,
					Bytes: snapshot.Size, // bytes == size indicates completion
				})
			default:
				// Unknown operation type
				logsToSave = append(logsToSave, &ent.JobLog{
					Level: model.LogLevelWarning,
					What:  model.LogActionUnknown,
					Size:  snapshot.Size,
					Time:  time.Now(),
				})
			}
		} else {
			// Collect active transfers for progress broadcast
			snapshot := tr.Snapshot()
			activeTransfers = append(activeTransfers, &model.TransferItem{
				Name:  snapshot.Name,
				Size:  snapshot.Size,
				Bytes: snapshot.Bytes,
			})
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

	// Get total stats for progress display
	totalTransfers, totalBytes := getTotalStats(s)
	filesDeleted, errorCount := s.GetDeletes(), s.GetErrors()

	// Broadcast progress update
	if task.Edges.Connection != nil {
		e.broadcastJobUpdate(&model.JobProgressEvent{
			JobID:            jobID,
			TaskID:           task.ID,
			ConnectionID:     task.Edges.Connection.ID,
			Status:           model.JobStatusRunning,
			FilesTransferred: int(s.GetTransfers()),
			BytesTransferred: s.GetBytes(),
			FilesTotal:       int(totalTransfers),
			BytesTotal:       totalBytes,
			FilesDeleted:     int(filesDeleted),
			ErrorCount:       int(errorCount),
			StartTime:        startTime,
		})

		// Broadcast transfer progress update (using snapshots collected while holding the lock)
		e.broadcastTransferProgress(jobID, task, activeTransfers)
	}
}

// broadcastTransferProgress broadcasts the current transfer progress for active file transfers.
func (e *SyncEngine) broadcastTransferProgress(jobID uuid.UUID, task *ent.Task, activeTransfers []*model.TransferItem) {
	if e.transferProgressBus == nil || task.Edges.Connection == nil {
		return
	}

	event := &model.TransferProgressEvent{
		JobID:        jobID,
		TaskID:       task.ID,
		ConnectionID: task.Edges.Connection.ID,
		Transfers:    activeTransfers,
	}

	e.broadcastTransferUpdate(event)
}

func (e *SyncEngine) broadcastTransferUpdate(event *model.TransferProgressEvent) {
	if e.transferProgressBus == nil {
		return
	}

	e.statsMu.Lock()
	defer e.statsMu.Unlock()

	last, ok := e.lastTransferEvents[event.JobID]
	if ok && transferEventsEqual(last, event) {
		return
	}

	e.lastTransferEvents[event.JobID] = event
	e.transferProgressBus.Publish(event)
}

// transferEventsEqual compares two TransferProgressEvent for equality.
func transferEventsEqual(a, b *model.TransferProgressEvent) bool {
	if a.JobID != b.JobID || a.TaskID != b.TaskID || a.ConnectionID != b.ConnectionID {
		return false
	}
	if len(a.Transfers) != len(b.Transfers) {
		return false
	}
	for i, ta := range a.Transfers {
		tb := b.Transfers[i]
		if ta.Name != tb.Name || ta.Size != tb.Size || ta.Bytes != tb.Bytes {
			return false
		}
	}
	return true
}

func (e *SyncEngine) broadcastJobUpdate(event *model.JobProgressEvent) {
	if e.jobProgressBus == nil {
		return
	}

	e.statsMu.Lock()
	defer e.statsMu.Unlock()

	last, ok := e.lastEvents[event.JobID]
	if ok && last.Status == event.Status &&
		last.FilesTransferred == event.FilesTransferred &&
		last.BytesTransferred == event.BytesTransferred &&
		last.FilesTotal == event.FilesTotal &&
		last.BytesTotal == event.BytesTotal &&
		last.TaskID == event.TaskID &&
		last.ConnectionID == event.ConnectionID &&
		last.StartTime.Equal(event.StartTime) &&
		utils.ComparePtr(last.EndTime, event.EndTime) {
		return
	}

	e.lastEvents[event.JobID] = event
	e.jobProgressBus.Publish(event)
}

// getTotalStats retrieves total transfers and total bytes from rclone stats using RemoteStats API.
// Returns (totalTransfers, totalBytes). Returns (0, 0) if stats is nil or on error.
func getTotalStats(s *accounting.StatsInfo) (int64, int64) {
	if s == nil {
		return 0, 0
	}

	rc, err := s.RemoteStats(false)
	if err != nil {
		return 0, 0
	}

	// Extract totalTransfers and totalBytes from rc.Params
	var totalTransfers, totalBytes int64

	if v, ok := rc["totalTransfers"]; ok {
		switch val := v.(type) {
		case int64:
			totalTransfers = val
		case int:
			totalTransfers = int64(val)
		case float64:
			totalTransfers = int64(val)
		}
	}

	if v, ok := rc["totalBytes"]; ok {
		switch val := v.(type) {
		case int64:
			totalBytes = val
		case int:
			totalBytes = int64(val)
		case float64:
			totalBytes = int64(val)
		}
	}

	return totalTransfers, totalBytes
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
