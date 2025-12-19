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
		// Initialize Config
		config.InitConfig(cfgFile)

		// Initialize Logger first
		logger.InitLogger(logger.Environment(config.Cfg.App.Environment), logger.LogLevel(config.Cfg.Log.Level))
		logger.L.Info("Starting cloud-sync server...")
		rclone.SetupLogLevel(config.Cfg.Log.Level)

		// Initialize i18n
		if err := i18n.Init(); err != nil {
			logger.L.Fatal("Failed to initialize i18n", zap.Error(err))
		}
		logger.L.Info("i18n initialized successfully")

		// Initialize database with configured options
		db.InitDB(db.InitDBOptions{
			Path:          config.Cfg.Database.Path,
			MigrationMode: db.ParseMigrationMode(config.Cfg.Database.MigrationMode),
			EnableDebug:   config.Cfg.App.Environment == "development",
		})
		defer db.CloseDB()

		// Initialize encryptor for connection storage
		encryptor, err := crypto.NewEncryptor(config.Cfg.Security.EncryptionKey)
		if err != nil {
			logger.L.Fatal("Failed to initialize encryptor", zap.Error(err))
		}

		// Initialize connection service and DBStorage
		connSvc := services.NewConnectionService(db.Client, encryptor)
		dbStorage := rclone.NewDBStorage(connSvc)
		dbStorage.Install()
		logger.L.Info("DBStorage installed - rclone will use database for configuration")

		// Note: rclone.InitConfig is no longer needed as DBStorage replaces it

		// Initialize services
		taskSvc := services.NewTaskService(db.Client)
		jobSvc := services.NewJobService(db.Client)
		syncEngine := rclone.NewSyncEngine(jobSvc, config.Cfg.App.DataDir)
		taskRunner := runner.NewRunner(syncEngine)

		// Reset any stuck jobs from previous crash/shutdown
		if err := jobSvc.ResetStuckJobs(context.Background()); err != nil {
			logger.L.Error("Failed to reset stuck jobs", zap.Error(err))
		}

		// Initialize and start scheduler
		sched := scheduler.NewScheduler(taskSvc, taskRunner)
		sched.Start()
		defer sched.Stop()

		// Initialize and start watcher
		watch, err := watcher.NewWatcher(taskSvc, taskRunner)
		if err != nil {
			logger.L.Fatal("Failed to initialize watcher", zap.Error(err))
		}
		watch.Start()
		defer watch.Stop()

		// Setup graceful shutdowncontext
		// Start API Server
		r := api.SetupRouter(syncEngine, taskRunner, jobSvc, watch, sched)

		addr := fmt.Sprintf("%s:%d", config.Cfg.Server.Host, config.Cfg.Server.Port)
		logger.L.Info("Server starting", zap.String("address", addr))

		srv := &http.Server{
			Addr:              addr,
			Handler:           r,
			ReadHeaderTimeout: 10 * time.Second,
		}

		go func() {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.L.Fatal("Server failed to start", zap.Error(err))
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
		logger.L.Info("Shutdown signal received, stopping server...")

		// The context is used to inform the server it has 5 seconds to finish
		// the request it is currently handling
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.L.Fatal("Server forced to shutdown", zap.Error(err))
		}

		// Stop the task runner (this waits for tasks to finish/cancel)
		taskRunner.Stop()

		logger.L.Info("Server exiting")
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
