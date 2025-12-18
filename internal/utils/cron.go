// Package utils provides utility functions for the application.
package utils //nolint:revive // utils is a meaningful package name for utility functions

import (
	"github.com/robfig/cron/v3"
)

// ValidateCronSchedule validates a cron schedule expression.
// It uses the standard 5-field format: minute hour day month weekday
// Returns nil if the schedule is valid, otherwise returns an error.
func ValidateCronSchedule(schedule string) error {
	if schedule == "" {
		return nil // Empty schedule is valid (means no schedule)
	}

	// Standard 5-field format: minute hour day month weekday
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(schedule)
	return err
}
