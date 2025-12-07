package logger

import (
	"log"
	"log/slog"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
)

// TODO: replace with function with name
var L *zap.Logger

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	Info          LogLevel = "info"
	Warn          LogLevel = "warn"
	Error         LogLevel = "error"
)

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
	defer L.Sync()

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
