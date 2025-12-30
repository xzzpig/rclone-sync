/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xzzpig/rclone-sync/internal/api"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"

	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/db"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/runner"
	"github.com/xzzpig/rclone-sync/internal/core/scheduler"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/core/watcher"
	"github.com/xzzpig/rclone-sync/internal/i18n"
	"github.com/xzzpig/rclone-sync/internal/rclone"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use: "serve",
	Run: func(_ *cobra.Command, _ []string) {
		// 1. Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			// Use default logger since config is not loaded yet
			logger.Get().Fatal("Failed to load config", zap.Error(err))
		}

		// 2. Initialize Logger with hierarchical level configuration
		logger.InitLogger(logger.Environment(cfg.App.Environment), logger.LogLevel(cfg.Log.Level), cfg.Log.Levels)
		log := logger.Named("cmd.serve")
		log.Info("Starting cloud-sync server...")
		rclone.SetupLogLevel(cfg.Log.Level)

		// 3. Initialize i18n
		if err := i18n.Init(); err != nil {
			log.Fatal("Failed to initialize i18n", zap.Error(err))
		}
		log.Info("i18n initialized successfully")

		// 4. Initialize database with configured options
		dbClient, err := db.InitDB(db.InitDBOptions{
			DSN:           db.FileSDN(cfg.Database.Path),
			MigrationMode: db.ParseMigrationMode(cfg.Database.MigrationMode),
			EnableDebug:   logger.GetLevelForName("core.db.query") == zap.DebugLevel,
			Environment:   cfg.App.Environment,
		})
		if err != nil {
			log.Fatal("Failed to initialize database", zap.Error(err))
		}
		defer db.CloseDB(dbClient)

		// 5. Initialize encryptor for connection storage
		encryptor, err := crypto.NewEncryptor(cfg.Security.EncryptionKey)
		if err != nil {
			log.Fatal("Failed to initialize encryptor", zap.Error(err))
		}

		// 6. Initialize connection service and DBStorage
		connSvc := services.NewConnectionService(dbClient, encryptor)
		dbStorage := rclone.NewDBStorage(connSvc)
		dbStorage.Install()
		log.Info("DBStorage installed - rclone will use database for configuration")

		// Note: rclone.InitConfig is no longer needed as DBStorage replaces it

		// 7. Initialize services
		taskSvc := services.NewTaskService(dbClient)
		jobSvc := services.NewJobService(dbClient)
		jobProgressBus := subscription.NewJobProgressBus()
		transferProgressBus := subscription.NewTransferProgressBus()
		syncEngine := rclone.NewSyncEngine(jobSvc, jobProgressBus, transferProgressBus, cfg.App.DataDir, cfg.App.Job.AutoDeleteEmptyJobs, cfg.App.Sync.Transfers)
		taskRunner := runner.NewRunner(syncEngine)

		// Reset any stuck jobs from previous crash/shutdown
		if err := jobSvc.ResetStuckJobs(context.Background()); err != nil {
			log.Error("Failed to reset stuck jobs", zap.Error(err))
		}

		// 8. Initialize and start scheduler
		sched := scheduler.NewScheduler(taskSvc, taskRunner)
		sched.Start()
		defer sched.Stop()

		// 9. Initialize and start watcher
		watch, err := watcher.NewWatcher(taskSvc, taskRunner)
		if err != nil {
			log.Fatal("Failed to initialize watcher", zap.Error(err))
		}
		watch.Start()
		defer watch.Stop()

		// 10. Initialize and start log cleanup service
		if cfg.App.Job.MaxLogsPerConnection > 0 && cfg.App.Job.CleanupSchedule != "" {
			logCleanupSvc := services.NewLogCleanupService(dbClient, cfg.App.Job.MaxLogsPerConnection)
			if err := logCleanupSvc.Start(cfg.App.Job.CleanupSchedule); err != nil {
			} else {
				defer logCleanupSvc.Stop()
			}
		}

		// 11. Setup router with dependencies
		routerDeps := api.RouterDeps{
			Client:              dbClient,
			Config:              cfg,
			SyncEngine:          syncEngine,
			Runner:              taskRunner,
			JobService:          jobSvc,
			Watcher:             watch,
			Scheduler:           sched,
			JobProgressBus:      jobProgressBus,
			TransferProgressBus: transferProgressBus,
		}
		r := api.SetupRouter(routerDeps)

		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Info("Server starting", zap.String("address", addr))

		srv := &http.Server{
			Addr:              addr,
			Handler:           r,
			ReadHeaderTimeout: 10 * time.Second,
		}

		go func() {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatal("Server failed to start", zap.Error(err))
			}
		}()

		// Wait for interrupt signal to gracefully shutdown the server with
		// a timeout of 5 seconds.
		quit := make(chan os.Signal, 1)
		// kill (no param) default send syscall.SIGTERM
		// kill -2 is syscall.SIGINT
		// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Info("Shutdown signal received, stopping server...")

		// The context is used to inform the server it has 5 seconds to finish
		// the request it is currently handling
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal("Server forced to shutdown", zap.Error(err))
		}

		// Stop the task runner (this waits for tasks to finish/cancel)
		taskRunner.Stop()

		log.Info("Server exiting")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
