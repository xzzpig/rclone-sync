package hook

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

func TestFormatTime(t *testing.T) {
	tm := time.Date(2025, 1, 17, 14, 30, 0, 0, time.UTC)
	result := formatTime(tm)
	assert.Equal(t, "2025-01-17T14:30:00Z", result)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 3*time.Minute + 30*time.Second, "3m30s"},
		{"hours", 2*time.Hour + 15*time.Minute, "2h15m0s"},
		{"sub-second rounds to zero", 500 * time.Millisecond, "1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatSizeBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"bytes", 512, "512 B"},
		{"kilobytes", 1536, "1.5 KB"},
		{"megabytes", 1048576, "1.0 MB"},
		{"gigabytes", 2147483648, "2.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSizeBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJsonMarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", `"hello"`},
		{"int", 42, "42"},
		{"struct", struct{ Name string }{"test"}, `{"Name":"test"}`},
		{"map", map[string]int{"a": 1}, `{"a":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jsonMarshal(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateSummary(t *testing.T) {
	ctx := &Context{
		Task: TaskInfo{
			ID:   uuid.New(),
			Name: "backup-task",
		},
		Event:    "ON_SUCCESS",
		Duration: 2*time.Minute + 30*time.Second,
		Stats: TransferStats{
			FilesTransferred: 150,
			BytesTransferred: 1073741824,
		},
	}

	result := generateSummary(ctx)
	assert.Contains(t, result, "backup-task")
	assert.Contains(t, result, "ON_SUCCESS")
	assert.Contains(t, result, "150 files")
	assert.Contains(t, result, "1.0 GB")
	assert.Contains(t, result, "2m30s")
}

func TestRenderTemplate(t *testing.T) {
	ctx := createTestContext()

	tests := []struct {
		name     string
		template string
		check    func(t *testing.T, result string)
	}{
		{
			name:     "simple field access",
			template: "Task: {{.Task.Name}}",
			check: func(t *testing.T, result string) {
				assert.Equal(t, "Task: test-task", result)
			},
		},
		{
			name:     "event field",
			template: "Event: {{.Event}}",
			check: func(t *testing.T, result string) {
				assert.Equal(t, "Event: ON_SUCCESS", result)
			},
		},
		{
			name:     "stats fields",
			template: "Files: {{.Stats.FilesTransferred}}, Bytes: {{.Stats.BytesTransferred}}",
			check: func(t *testing.T, result string) {
				assert.Equal(t, "Files: 100, Bytes: 1048576", result)
			},
		},
		{
			name:     "FormatTime function",
			template: "Started: {{FormatTime .Job.StartTime}}",
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, "Started: 2025-01-17")
			},
		},
		{
			name:     "FormatDuration function",
			template: "Duration: {{FormatDuration .Duration}}",
			check: func(t *testing.T, result string) {
				assert.Equal(t, "Duration: 1m30s", result)
			},
		},
		{
			name:     "FormatSizeBytes function",
			template: "Size: {{FormatSizeBytes .Stats.BytesTransferred}}",
			check: func(t *testing.T, result string) {
				assert.Equal(t, "Size: 1.0 MB", result)
			},
		},
		{
			name:     "JsonMarshal function",
			template: `{{JsonMarshal .Task}}`,
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, `"name":"test-task"`)
			},
		},
		{
			name:     "Summary function",
			template: "{{Summary .}}",
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, "test-task")
				assert.Contains(t, result, "ON_SUCCESS")
			},
		},
		{
			name:     "error field when present",
			template: "Error: {{.Error}}",
			check: func(t *testing.T, result string) {
				assert.Equal(t, "Error: ", result)
			},
		},
		{
			name:     "job status",
			template: "Status: {{.Job.Status}}",
			check: func(t *testing.T, result string) {
				assert.Equal(t, "Status: SUCCESS", result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderTemplate(tt.template, ctx)
			require.NoError(t, err)
			tt.check(t, result)
		})
	}
}

func TestRenderTemplate_InvalidTemplate(t *testing.T) {
	ctx := createTestContext()

	_, err := RenderTemplate("{{.Invalid", ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), i18n.ErrHookInvalidTemplate)
}

func TestRenderTemplate_InvalidField(t *testing.T) {
	ctx := createTestContext()

	_, err := RenderTemplate("{{.NonExistent.Field}}", ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), i18n.ErrHookInvalidTemplate)
}

func TestRenderTemplate_EnvVariableInterpolation(t *testing.T) {
	ctx := createTestContext()
	ctx.Env["TEST_VAR"] = "test_value"
	ctx.Env["HOME"] = "/home/user"

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "access env variable",
			template: `Env: {{index .Env "TEST_VAR"}}`,
			expected: "Env: test_value",
		},
		{
			name:     "access HOME env",
			template: `Home: {{index .Env "HOME"}}`,
			expected: "Home: /home/user",
		},
		{
			name:     "missing env returns empty",
			template: `Missing: {{index .Env "NON_EXISTENT"}}`,
			expected: "Missing: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderTemplate(tt.template, ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func createTestContext() *Context {
	return &Context{
		Task: TaskInfo{
			ID:         uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Name:       "test-task",
			SourcePath: "/local/path",
			RemotePath: "remote:backup",
			Direction:  "UPLOAD",
		},
		Job: JobInfo{
			ID:        uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Status:    "SUCCESS",
			Trigger:   "MANUAL",
			StartTime: time.Date(2025, 1, 17, 10, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2025, 1, 17, 10, 1, 30, 0, time.UTC),
		},
		Event:    "ON_SUCCESS",
		Error:    "",
		Duration: 90 * time.Second,
		Stats: TransferStats{
			FilesTransferred: 100,
			BytesTransferred: 1048576,
			FilesDeleted:     5,
			ErrorCount:       0,
		},
		Env: make(map[string]string),
	}
}
