package rclone

import (
	"context"
	"testing"

	"github.com/rclone/rclone/fs"
	"github.com/stretchr/testify/assert"
)

func TestSetupLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel fs.LogLevel
	}{
		{
			name:          "debug level",
			level:         "debug",
			expectedLevel: fs.LogLevelDebug,
		},
		{
			name:          "info level",
			level:         "info",
			expectedLevel: fs.LogLevelInfo,
		},
		{
			name:          "warn level maps to Notice",
			level:         "warn",
			expectedLevel: fs.LogLevelNotice,
		},
		{
			name:          "error level",
			level:         "error",
			expectedLevel: fs.LogLevelError,
		},
		{
			name:          "unknown level defaults to Notice",
			level:         "unknown",
			expectedLevel: fs.LogLevelNotice,
		},
		{
			name:          "empty string defaults to Notice",
			level:         "",
			expectedLevel: fs.LogLevelNotice,
		},
		{
			name:          "invalid level defaults to Notice",
			level:         "invalid",
			expectedLevel: fs.LogLevelNotice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call SetupLogLevel with the test level
			SetupLogLevel(tt.level)

			// Get the rclone config and verify the log level was set correctly
			cfg := fs.GetConfig(context.Background())
			assert.Equal(t, tt.expectedLevel, cfg.LogLevel,
				"LogLevel should be set to %v for input %q", tt.expectedLevel, tt.level)
		})
	}
}

// TestSetupLogLevel_CaseSensitivity tests that the function is case-sensitive
func TestSetupLogLevel_CaseSensitivity(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel fs.LogLevel
	}{
		{
			name:          "DEBUG (uppercase) defaults to Notice",
			level:         "DEBUG",
			expectedLevel: fs.LogLevelNotice,
		},
		{
			name:          "Info (mixed case) defaults to Notice",
			level:         "Info",
			expectedLevel: fs.LogLevelNotice,
		},
		{
			name:          "WARN (uppercase) defaults to Notice",
			level:         "WARN",
			expectedLevel: fs.LogLevelNotice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetupLogLevel(tt.level)
			cfg := fs.GetConfig(context.Background())
			assert.Equal(t, tt.expectedLevel, cfg.LogLevel,
				"LogLevel should default to Notice for case-mismatched input %q", tt.level)
		})
	}
}
