package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"go.uber.org/zap"
)

// LogCleanupService provides operations for cleaning up old job logs.
type LogCleanupService struct {
	client   *ent.Client
	logger   *zap.Logger
	maxLogs  int
	cron     *cron.Cron
	entryID  cron.EntryID
	jobSvc   *JobService
	connList func(ctx context.Context) ([]*ent.Connection, error)
}

// NewLogCleanupService creates a new LogCleanupService instance.
func NewLogCleanupService(client *ent.Client, maxLogsPerConnection int) *LogCleanupService {
	jobSvc := NewJobService(client)
	return &LogCleanupService{
		client:  client,
		logger:  logger.Named("service.log_cleanup"),
		maxLogs: maxLogsPerConnection,
		jobSvc:  jobSvc,
		connList: func(ctx context.Context) ([]*ent.Connection, error) {
			return client.Connection.Query().All(ctx)
		},
	}
}

// Start starts the log cleanup cron job with the given schedule.
func (s *LogCleanupService) Start(schedule string) error {
	s.logger.Info("Starting log cleanup service",
		zap.String("schedule", schedule),
		zap.Int("max_logs_per_connection", s.maxLogs))

	s.cron = cron.New()

	entryID, err := s.cron.AddFunc(schedule, func() {
		ctx := context.Background()
		if err := s.CleanupLogs(ctx); err != nil {
			s.logger.Error("Log cleanup failed", zap.Error(err))
		}
	})

	if err != nil {
		return err
	}

	s.entryID = entryID
	s.cron.Start()

	s.logger.Info("Log cleanup service started")
	return nil
}

// Stop stops the log cleanup cron job.
func (s *LogCleanupService) Stop() {
	if s.cron != nil {
		s.logger.Info("Stopping log cleanup service")
		s.cron.Stop()
		s.cron = nil
	}
}

// CleanupLogs cleans up old logs for all connections.
func (s *LogCleanupService) CleanupLogs(ctx context.Context) error {
	s.logger.Info("Starting log cleanup for all connections")

	// Get all connections
	connections, err := s.connList(ctx)
	if err != nil {
		s.logger.Error("Failed to list connections", zap.Error(err))
		return err
	}

	for _, conn := range connections {
		if err := s.CleanupLogsForConnection(ctx, conn.ID); err != nil {
			s.logger.Error("Failed to cleanup logs for connection",
				zap.String("connection_id", conn.ID.String()),
				zap.Error(err))
			// Continue with other connections
			continue
		}
	}

	s.logger.Info("Log cleanup completed",
		zap.Int("connections_processed", len(connections)))

	return nil
}

// CleanupLogsForConnection cleans up old logs for a specific connection.
// Returns the number of logs deleted.
func (s *LogCleanupService) CleanupLogsForConnection(ctx context.Context, connectionID uuid.UUID) error {
	deleted, err := s.jobSvc.DeleteOldLogsForConnection(ctx, connectionID, s.maxLogs)
	if err != nil {
		return err
	}

	if deleted > 0 {
		s.logger.Debug("Cleaned up logs for connection",
			zap.String("connection_id", connectionID.String()),
			zap.Int("deleted", deleted))
	}

	return nil
}
