// Package logger provides logging utilities for the application.
package logger

import (
	"log"
	"log/slog"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
)

// L is the global logger instance.
// TODO: replace with function with name
var L *zap.Logger

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
	L, err = cfg.Build()
	if err != nil {
		log.Printf("Failed to initialize zap logger: %v", err)
		os.Exit(1)
	}
	defer func() { _ = L.Sync() }()

	// Redirect standard log to zap
	zap.RedirectStdLog(L)

	// Redirect slog to zap (for rclone)
	slogHandler := zapslog.NewHandler(L.Core())
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
