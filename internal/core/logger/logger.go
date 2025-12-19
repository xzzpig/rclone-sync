// Package logger provides logging utilities for the application.
package logger

import (
	"log"
	"log/slog"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
)

// logger is the global logger instance, lazily initialized with a default Info-level logger.
var (
	logger     *zap.Logger
	loggerOnce sync.Once
)

// initDefaultLogger initializes a default Info-level logger if none has been set.
func initDefaultLogger() {
	loggerOnce.Do(func() {
		if logger == nil {
			cfg := zap.NewProductionConfig()
			cfg.Level.SetLevel(zapcore.InfoLevel)
			var err error
			logger, err = cfg.Build()
			if err != nil {
				// Fallback to nop logger if we can't create default
				logger = zap.NewNop()
			}
		}
	})
}

// Get returns the logger instance. If InitLogger hasn't been called, returns a default Info-level logger.
func Get() *zap.Logger {
	initDefaultLogger()
	return logger
}

// Named returns a named logger. If Init hasn't been called, returns a named default logger.
func Named(name string) *zap.Logger {
	return Get().Named(name)
}

// Environment represents the application environment type.
type Environment string

const (
	// EnvironmentDevelopment represents the development environment.
	EnvironmentDevelopment Environment = "development"
	// EnvironmentProduction represents the production environment.
	EnvironmentProduction Environment = "production"
)

// LogLevel represents the logging level type.
type LogLevel string

const (
	// LogLevelDebug represents the debug logging level.
	LogLevelDebug LogLevel = "debug"
	// Info represents the info logging level.
	Info LogLevel = "info"
	// Warn represents the warn logging level.
	Warn LogLevel = "warn"
	// Error represents the error logging level.
	Error LogLevel = "error"
)

// InitLogger initializes the global logger with the specified environment and log level.
func InitLogger(environment Environment, logLevel LogLevel) {
	var cfg zap.Config

	if environment == EnvironmentDevelopment {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	cfg.Level.SetLevel(getZapLevel(string(logLevel)))

	var err error
	logger, err = cfg.Build()
	if err != nil {
		log.Printf("Failed to initialize zap logger: %v", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	// Redirect standard log to zap
	zap.RedirectStdLog(logger)

	// Redirect slog to zap (for rclone)
	slogHandler := zapslog.NewHandler(logger.Core())
	slogLogger := slog.New(slogHandler)
	slog.SetDefault(slogLogger)
}

func getZapLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
