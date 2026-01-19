package hook

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
)

type executor struct {
	hookQuery      ports.HookQuery
	jobQuery       ports.JobQuery
	httpClient     *http.Client
	enabled        *bool
	defaultTimeout int
}

// NewExecutor creates a new Executor instance.
func NewExecutor(hookQuery ports.HookQuery, jobQuery ports.JobQuery, enabled *bool, defaultTimeout int) ports.HookExecutor {
	timeout := time.Duration(defaultTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &executor{
		hookQuery: hookQuery,
		jobQuery:  jobQuery,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		enabled:        enabled,
		defaultTimeout: defaultTimeout,
	}
}

func (e *executor) Execute(ctx context.Context, task *ent.Task, job *ent.Job, event model.HookEvent, syncErr error) error {
	if e.enabled != nil && !*e.enabled {
		return nil
	}

	hooks, err := e.hookQuery.GetHooksForEvent(ctx, task.ID, task.ConnectionID, event)
	if err != nil {
		return err
	}

	if len(hooks) == 0 {
		return nil
	}

	hookCtx := e.buildContextFromEntities(task, job, event, syncErr)

	for _, h := range hooks {
		startTime := time.Now()
		code, execErr := e.executeOne(ctx, h, hookCtx)
		duration := time.Since(startTime)

		logSize := duration.Milliseconds()
		if execErr != nil {
			logSize = code
			e.logHookExecution(ctx, job.ID, h, event, logSize, execErr)

			switch h.OnError {
			case model.HookOnErrorCancel:
				return &CancelError{HookID: h.ID, Cause: execErr}
			case model.HookOnErrorFatal:
				return &FatalError{HookID: h.ID, Cause: execErr}
			}
		} else {
			e.logHookExecution(ctx, job.ID, h, event, logSize, nil)
		}
	}

	return nil
}

func (e *executor) buildContextFromEntities(task *ent.Task, job *ent.Job, event model.HookEvent, syncErr error) *Context {
	var errMsg string
	if syncErr != nil {
		errMsg = syncErr.Error()
	}

	endTime := job.EndTime
	duration := time.Duration(0)
	if !endTime.IsZero() {
		duration = endTime.Sub(job.StartTime)
	}

	return BuildContext(
		task.ID, task.Name, task.SourcePath, task.RemotePath, string(task.Direction),
		job.ID, string(job.Status), string(job.Trigger), job.StartTime, endTime,
		string(event), errMsg, duration,
		int64(job.FilesTransferred), job.BytesTransferred, int64(job.FilesDeleted), int64(job.ErrorCount),
	)
}

func (e *executor) executeOne(ctx context.Context, h *ent.TaskHook, hookCtx *Context) (int64, error) {
	switch h.Type {
	case model.HookTypeHTTP:
		return e.executeHTTP(ctx, h, hookCtx)
	case model.HookTypeCommand:
		return e.executeCommand(ctx, h, hookCtx)
	default:
		return 0, nil
	}
}

func (e *executor) logHookExecution(ctx context.Context, jobID uuid.UUID, h *ent.TaskHook, event model.HookEvent, size int64, execErr error) {
	level := model.LogLevelInfo
	if execErr != nil {
		level = model.LogLevelError
	}

	path := "hook:" + h.ID.String() + ":" + strings.ToLower(string(event))

	_, _ = e.jobQuery.AddJobLog(ctx, jobID, string(level), string(model.LogActionHook), path, size)
}
