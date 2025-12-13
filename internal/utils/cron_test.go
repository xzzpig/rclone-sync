package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCronSchedule(t *testing.T) {
	tests := []struct {
		name      string
		schedule  string
		expectErr bool
	}{
		{
			name:      "Empty schedule is valid",
			schedule:  "",
			expectErr: false,
		},
		{
			name:      "Valid cron - every 6 hours",
			schedule:  "0 */6 * * *",
			expectErr: false,
		},
		{
			name:      "Valid cron - daily at midnight",
			schedule:  "0 0 * * *",
			expectErr: false,
		},
		{
			name:      "Valid cron - every minute",
			schedule:  "* * * * *",
			expectErr: false,
		},
		{
			name:      "Valid cron - every Monday at 9am",
			schedule:  "0 9 * * 1",
			expectErr: false,
		},
		{
			name:      "Valid cron - first day of month at midnight",
			schedule:  "0 0 1 * *",
			expectErr: false,
		},
		{
			name:      "Valid cron - descriptor @daily",
			schedule:  "@daily",
			expectErr: false,
		},
		{
			name:      "Valid cron - descriptor @hourly",
			schedule:  "@hourly",
			expectErr: false,
		},
		{
			name:      "Invalid cron - with seconds (6 fields)",
			schedule:  "0 */6 * * * *",
			expectErr: true,
		},
		{
			name:      "Invalid cron - too many fields (7 fields)",
			schedule:  "0 0 * * * * *",
			expectErr: true,
		},
		{
			name:      "Invalid cron - invalid minute range",
			schedule:  "99 0 * * *",
			expectErr: true,
		},
		{
			name:      "Invalid cron - invalid hour range",
			schedule:  "0 99 * * *",
			expectErr: true,
		},
		{
			name:      "Invalid cron - garbage input",
			schedule:  "invalid cron",
			expectErr: true,
		},
		{
			name:      "Invalid cron - only 3 fields",
			schedule:  "0 0 *",
			expectErr: true,
		},
		{
			name:      "Invalid cron - invalid day of week",
			schedule:  "0 0 * * 8",
			expectErr: true,
		},
		{
			name:      "Valid cron - complex expression",
			schedule:  "15,30,45 8-17 * * 1-5",
			expectErr: false,
		},
		{
			name:      "Valid cron - range with step",
			schedule:  "*/15 9-17 * * *",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCronSchedule(tt.schedule)
			if tt.expectErr {
				assert.Error(t, err, "Expected error for schedule: %s", tt.schedule)
			} else {
				assert.NoError(t, err, "Expected no error for schedule: %s", tt.schedule)
			}
		})
	}
}
