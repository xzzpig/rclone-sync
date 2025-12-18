package rclone

import (
	"context"

	"github.com/rclone/rclone/fs"
)

// SetupLogLevel sets the rclone log level based on the provided string.
func SetupLogLevel(level string) {
	switch level {
	case "debug":
		fs.GetConfig(context.Background()).LogLevel = fs.LogLevelDebug
	case "info":
		fs.GetConfig(context.Background()).LogLevel = fs.LogLevelInfo
	case "warn":
		fs.GetConfig(context.Background()).LogLevel = fs.LogLevelNotice
	case "error":
		fs.GetConfig(context.Background()).LogLevel = fs.LogLevelError
	default:
		fs.GetConfig(context.Background()).LogLevel = fs.LogLevelNotice
	}
}
