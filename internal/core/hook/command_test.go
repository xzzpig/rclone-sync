package hook

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

func TestExecuteCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	exec := createCommandTestExecutor()
	ctx := createTestContext()

	tests := []struct {
		name        string
		command     *string
		workDir     *string
		timeout     *int
		wantErr     bool
		errContains string
		check       func(t *testing.T, code int64, err error)
	}{
		{
			name:    "BasicExecution",
			command: ptrString("echo 'Hello World'"),
			check: func(t *testing.T, code int64, err error) {
				require.NoError(t, err)
				assert.Equal(t, int64(0), code)
			},
		},
		{
			name:        "ErrorOnMissingCommand",
			command:     nil,
			wantErr:     true,
			errContains: i18n.ErrMissingParameter,
		},
		{
			name:        "ExitCodeNonZero",
			command:     ptrString("exit 1"),
			wantErr:     true,
			errContains: i18n.ErrHookCommandFailed,
		},
		{
			name:        "ExitCodeCustom",
			command:     ptrString("exit 42"),
			wantErr:     true,
			errContains: i18n.ErrHookCommandFailed,
		},
		{
			name:        "InvalidTemplate",
			command:     ptrString("echo '{{.Invalid'"),
			wantErr:     true,
			errContains: i18n.ErrHookInvalidTemplate,
		},
		{
			name:        "StderrOnFailure",
			command:     ptrString("echo 'error message' >&2 && exit 1"),
			wantErr:     true,
			errContains: i18n.ErrHookCommandFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &ent.TaskHook{
				Type: model.HookTypeCommand,
				Config: &model.HookConfig{
					Command: tt.command,
					WorkDir: tt.workDir,
					Timeout: tt.timeout,
				},
			}

			code, err := exec.executeCommand(context.Background(), hook, ctx)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			if tt.check != nil {
				tt.check(t, code, err)
			}
		})
	}
}

func TestExecuteCommand_Functional(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	exec := createCommandTestExecutor()
	ctx := createTestContext()

	t.Run("TemplateRendering", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "output.txt")
		command := "echo '{{.Task.Name}}:{{.Event}}' > " + outputFile
		hook := &ent.TaskHook{
			Type:   model.HookTypeCommand,
			Config: &model.HookConfig{Command: &command},
		}

		_, err := exec.executeCommand(context.Background(), hook, ctx)
		require.NoError(t, err)

		content, _ := os.ReadFile(outputFile)
		assert.Contains(t, string(content), "test-task:ON_SUCCESS")
	})

	t.Run("WorkingDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "pwd.txt")
		command := "pwd > " + outputFile
		hook := &ent.TaskHook{
			Type:   model.HookTypeCommand,
			Config: &model.HookConfig{Command: &command, WorkDir: &tmpDir},
		}

		_, err := exec.executeCommand(context.Background(), hook, ctx)
		require.NoError(t, err)

		content, _ := os.ReadFile(outputFile)
		assert.Contains(t, string(content), tmpDir)
	})

	t.Run("EnvironmentVariables", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "env.txt")
		command := "env | grep RCLONE_SYNC > " + outputFile
		hook := &ent.TaskHook{
			Type:   model.HookTypeCommand,
			Config: &model.HookConfig{Command: &command},
		}

		_, err := exec.executeCommand(context.Background(), hook, ctx)
		require.NoError(t, err)

		content, _ := os.ReadFile(outputFile)
		envOutput := string(content)
		assert.Contains(t, envOutput, "RCLONE_SYNC_TASK_ID=11111111-1111-1111-1111-111111111111")
		assert.Contains(t, envOutput, "RCLONE_SYNC_TASK_NAME=test-task")
		assert.Contains(t, envOutput, "RCLONE_SYNC_EVENT=ON_SUCCESS")
		assert.Contains(t, envOutput, "RCLONE_SYNC_STATUS=SUCCESS")
	})

	t.Run("Timeout", func(t *testing.T) {
		e := &executor{defaultTimeout: 1, enabled: ptrBool(true)}
		timeout := 1
		command := "sleep 10"
		hook := &ent.TaskHook{
			Type:   model.HookTypeCommand,
			Config: &model.HookConfig{Command: &command, Timeout: &timeout},
		}

		start := time.Now()
		_, err := e.executeCommand(context.Background(), hook, ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), i18n.ErrHookTimeout)
		assert.Less(t, time.Since(start), 5*time.Second)
	})
}

func createCommandTestExecutor() *executor {
	return &executor{
		defaultTimeout: 30,
		enabled:        ptrBool(true),
	}
}

func ptrBool(b bool) *bool {
	return &b
}
