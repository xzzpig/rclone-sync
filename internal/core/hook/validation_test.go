package hook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

func ptrString(s string) *string {
	return &s
}

func ptrInt(i int) *int {
	return &i
}

func TestValidateHookConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *model.HookConfigInput
		hookType    model.HookType
		wantErr     bool
		expectedErr error
		errContains string
	}{
		{
			name:        "nil config",
			config:      nil,
			hookType:    model.HookTypeHTTP,
			wantErr:     true,
			expectedErr: ErrConfigRequired,
		},
		{
			name:        "HTTP missing URL",
			config:      &model.HookConfigInput{},
			hookType:    model.HookTypeHTTP,
			wantErr:     true,
			expectedErr: ErrURLMissing,
		},
		{
			name:        "HTTP empty URL",
			config:      &model.HookConfigInput{URL: ptrString("")},
			hookType:    model.HookTypeHTTP,
			wantErr:     true,
			expectedErr: ErrURLMissing,
		},
		{
			name:        "HTTP invalid URL",
			config:      &model.HookConfigInput{URL: ptrString("not-a-valid-url")},
			hookType:    model.HookTypeHTTP,
			wantErr:     true,
			errContains: i18n.ErrHookInvalidURL,
		},
		{
			name:     "HTTP valid URL",
			config:   &model.HookConfigInput{URL: ptrString("https://example.com/webhook")},
			hookType: model.HookTypeHTTP,
			wantErr:  false,
		},
		{
			name:        "HTTP invalid body template",
			config:      &model.HookConfigInput{URL: ptrString("https://example.com/webhook"), Body: ptrString("{{.Invalid")},
			hookType:    model.HookTypeHTTP,
			wantErr:     true,
			errContains: i18n.ErrHookInvalidTemplate,
		},
		{
			name:     "HTTP valid body template",
			config:   &model.HookConfigInput{URL: ptrString("https://example.com/webhook"), Body: ptrString(`{"task": "{{.Task.Name}}"}`)},
			hookType: model.HookTypeHTTP,
			wantErr:  false,
		},
		{
			name:     "HTTP empty body is valid",
			config:   &model.HookConfigInput{URL: ptrString("https://example.com/webhook"), Body: ptrString("")},
			hookType: model.HookTypeHTTP,
			wantErr:  false,
		},
		{
			name:     "HTTP with valid timeout",
			config:   &model.HookConfigInput{URL: ptrString("https://example.com/webhook"), Timeout: ptrInt(30)},
			hookType: model.HookTypeHTTP,
			wantErr:  false,
		},
		{
			name:     "HTTP URL with templating",
			config:   &model.HookConfigInput{URL: ptrString("https://example.com/webhook?task={{.Task.Name}}")},
			hookType: model.HookTypeHTTP,
			wantErr:  false,
		},
		{
			name:        "HTTP URL invalid template",
			config:      &model.HookConfigInput{URL: ptrString("https://example.com/webhook?task={{.Task.Name")},
			hookType:    model.HookTypeHTTP,
			wantErr:     true,
			errContains: i18n.ErrHookInvalidTemplate,
		},
		{
			name:        "HTTP headers invalid template",
			config:      &model.HookConfigInput{URL: ptrString("https://example.com/webhook"), Headers: map[string]string{"X-Trace": "{{.Task.Name"}},
			hookType:    model.HookTypeHTTP,
			wantErr:     true,
			errContains: i18n.ErrHookInvalidTemplate,
		},
		{
			name:     "HTTP headers valid template",
			config:   &model.HookConfigInput{URL: ptrString("https://example.com/webhook"), Headers: map[string]string{"X-Trace": "{{.Task.Name}}"}},
			hookType: model.HookTypeHTTP,
			wantErr:  false,
		},
		{
			name:        "Command missing",
			config:      &model.HookConfigInput{},
			hookType:    model.HookTypeCommand,
			wantErr:     true,
			expectedErr: ErrCommandMissing,
		},
		{
			name:        "Command invalid template",
			config:      &model.HookConfigInput{Command: ptrString("echo {{.Task.Name")},
			hookType:    model.HookTypeCommand,
			wantErr:     true,
			errContains: i18n.ErrHookInvalidTemplate,
		},
		{
			name:        "Command empty",
			config:      &model.HookConfigInput{Command: ptrString("")},
			hookType:    model.HookTypeCommand,
			wantErr:     true,
			expectedErr: ErrCommandMissing,
		},
		{
			name:     "Command valid",
			config:   &model.HookConfigInput{Command: ptrString("echo 'hello'")},
			hookType: model.HookTypeCommand,
			wantErr:  false,
		},
		{
			name:        "timeout zero",
			config:      &model.HookConfigInput{Command: ptrString("echo test"), Timeout: ptrInt(0)},
			hookType:    model.HookTypeCommand,
			wantErr:     true,
			expectedErr: ErrTimeoutOutOfRange,
		},
		{
			name:        "timeout negative",
			config:      &model.HookConfigInput{Command: ptrString("echo test"), Timeout: ptrInt(-1)},
			hookType:    model.HookTypeCommand,
			wantErr:     true,
			expectedErr: ErrTimeoutOutOfRange,
		},
		{
			name:        "timeout too high",
			config:      &model.HookConfigInput{Command: ptrString("echo test"), Timeout: ptrInt(3601)},
			hookType:    model.HookTypeCommand,
			wantErr:     true,
			expectedErr: ErrTimeoutOutOfRange,
		},
		{
			name:     "timeout valid min",
			config:   &model.HookConfigInput{Command: ptrString("echo test"), Timeout: ptrInt(1)},
			hookType: model.HookTypeCommand,
			wantErr:  false,
		},
		{
			name:     "timeout valid max",
			config:   &model.HookConfigInput{Command: ptrString("echo test"), Timeout: ptrInt(3600)},
			hookType: model.HookTypeCommand,
			wantErr:  false,
		},
		{
			name:     "timeout nil",
			config:   &model.HookConfigInput{Command: ptrString("echo test"), Timeout: nil},
			hookType: model.HookTypeCommand,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHookConfig(tt.config, tt.hookType)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				}
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
