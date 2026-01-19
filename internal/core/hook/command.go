package hook

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

func (e *executor) executeCommand(ctx context.Context, h *ent.TaskHook, hookCtx *Context) (int64, error) {
	cfg := h.Config

	if cfg.Command == nil || *cfg.Command == "" {
		return -1, ErrCommandMissing
	}

	renderedCmd, err := RenderTemplate(*cfg.Command, hookCtx)
	if err != nil {
		return -1, NewInvalidTemplateError(err)
	}

	timeout := time.Duration(e.defaultTimeout) * time.Second
	if cfg.Timeout != nil && *cfg.Timeout > 0 {
		timeout = time.Duration(*cfg.Timeout) * time.Second
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// #nosec G204
	cmd := exec.CommandContext(execCtx, "sh", "-c", renderedCmd)

	// Set process group so we can kill all child processes on timeout (Unix only)
	setupProcessGroup(cmd)

	if cfg.WorkDir != nil && *cfg.WorkDir != "" {
		cmd.Dir = *cfg.WorkDir
	}

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, buildEnvVars(hookCtx)...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		return -1, NewTimeoutError(timeout.Seconds())
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return int64(-exitErr.ExitCode()), NewCommandFailedError(exitErr.ExitCode())
		}
		return -1, i18n.NewI18nError(i18n.ErrGeneric).WithCause(err)
	}

	return 0, nil
}

func buildEnvVars(ctx *Context) []string {
	return []string{
		"RCLONE_SYNC_TASK_ID=" + ctx.Task.ID.String(),
		"RCLONE_SYNC_TASK_NAME=" + ctx.Task.Name,
		"RCLONE_SYNC_JOB_ID=" + ctx.Job.ID.String(),
		"RCLONE_SYNC_EVENT=" + ctx.Event,
		"RCLONE_SYNC_STATUS=" + ctx.Job.Status,
		"RCLONE_SYNC_ERROR=" + ctx.Error,
		"RCLONE_SYNC_FILES_TRANSFERRED=" + strconv.FormatInt(ctx.Stats.FilesTransferred, 10),
		"RCLONE_SYNC_BYTES_TRANSFERRED=" + strconv.FormatInt(ctx.Stats.BytesTransferred, 10),
		"RCLONE_SYNC_DURATION_SECONDS=" + strconv.FormatInt(int64(ctx.Duration.Seconds()), 10),
	}
}
