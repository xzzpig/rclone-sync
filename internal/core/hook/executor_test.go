package hook

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

type mockHookQuery struct {
	hooks []*ent.TaskHook
	err   error
}

func (m *mockHookQuery) GetHook(ctx context.Context, id uuid.UUID) (*ent.TaskHook, error) {
	return nil, nil
}

func (m *mockHookQuery) ListHooks(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID, event *model.HookEvent) ([]*ent.TaskHook, error) {
	return m.hooks, m.err
}

func (m *mockHookQuery) CreateHook(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID, input model.HookInput) (*ent.TaskHook, error) {
	return nil, nil
}

func (m *mockHookQuery) UpdateHook(ctx context.Context, id uuid.UUID, input model.UpdateHookInput) (*ent.TaskHook, error) {
	return nil, nil
}

func (m *mockHookQuery) DeleteHook(ctx context.Context, id uuid.UUID) (*ent.TaskHook, error) {
	return nil, nil
}

func (m *mockHookQuery) GetHooksForEvent(ctx context.Context, taskID uuid.UUID, connectionID uuid.UUID, event model.HookEvent) ([]*ent.TaskHook, error) {
	return m.hooks, m.err
}

type mockJobQuery struct {
	logs []*ent.JobLog
}

func (m *mockJobQuery) CreateJob(ctx context.Context, taskID uuid.UUID, trigger model.JobTrigger) (*ent.Job, error) {
	return nil, nil
}

func (m *mockJobQuery) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string, errStr string) (*ent.Job, error) {
	return nil, nil
}

func (m *mockJobQuery) UpdateJobStats(ctx context.Context, jobID uuid.UUID, files, bytes, filesDeleted, errorCount int64) (*ent.Job, error) {
	return nil, nil
}

func (m *mockJobQuery) AddJobLog(ctx context.Context, jobID uuid.UUID, level, what, path string, size int64) (*ent.JobLog, error) {
	log := &ent.JobLog{}
	m.logs = append(m.logs, log)
	return log, nil
}

func (m *mockJobQuery) AddJobLogsBatch(ctx context.Context, jobID uuid.UUID, logs []*ent.JobLog) error {
	return nil
}

func (m *mockJobQuery) GetJob(ctx context.Context, jobID uuid.UUID) (*ent.Job, error) {
	return nil, nil
}

func (m *mockJobQuery) GetLastJobByTaskID(ctx context.Context, taskID uuid.UUID) (*ent.Job, error) {
	return nil, nil
}

func (m *mockJobQuery) ListJobs(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID, limit, offset int) ([]*ent.Job, error) {
	return nil, nil
}

func (m *mockJobQuery) CountJobs(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockJobQuery) GetJobWithLogs(ctx context.Context, jobID uuid.UUID) (*ent.Job, error) {
	return nil, nil
}

func (m *mockJobQuery) ListJobLogs(ctx context.Context, connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string, limit, offset int) ([]*ent.JobLog, error) {
	return nil, nil
}

func (m *mockJobQuery) CountJobLogs(ctx context.Context, connectionID *uuid.UUID, taskID *uuid.UUID, jobID *uuid.UUID, level string) (int, error) {
	return 0, nil
}

func (m *mockJobQuery) DeleteJob(ctx context.Context, jobID uuid.UUID) error {
	return nil
}

func TestNewExecutor(t *testing.T) {
	hookQuery := &mockHookQuery{}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)
	require.NotNil(t, exec)
}

func TestNewExecutor_ZeroTimeout(t *testing.T) {
	hookQuery := &mockHookQuery{}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 0)
	require.NotNil(t, exec)
}

func TestNewExecutor_NegativeTimeout(t *testing.T) {
	hookQuery := &mockHookQuery{}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, -10)
	require.NotNil(t, exec)
}

func TestExecutor_Execute_Disabled(t *testing.T) {
	hookQuery := &mockHookQuery{}
	jobQuery := &mockJobQuery{}
	enabled := false

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	err := exec.Execute(context.Background(), task, job, model.HookEventOnSuccess, nil)
	assert.NoError(t, err)
}

func TestExecutor_Execute_NoHooks(t *testing.T) {
	hookQuery := &mockHookQuery{hooks: []*ent.TaskHook{}}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	err := exec.Execute(context.Background(), task, job, model.HookEventOnSuccess, nil)
	assert.NoError(t, err)
}

func TestExecutor_Execute_HookQueryError(t *testing.T) {
	expectedErr := errors.New("database error")
	hookQuery := &mockHookQuery{err: expectedErr}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	err := exec.Execute(context.Background(), task, job, model.HookEventOnSuccess, nil)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestExecutor_Execute_HTTPHook_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	url := server.URL
	hookQuery := &mockHookQuery{
		hooks: []*ent.TaskHook{
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config: &model.HookConfig{
					URL: &url,
				},
			},
		},
	}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	err := exec.Execute(context.Background(), task, job, model.HookEventOnSuccess, nil)
	assert.NoError(t, err)
	assert.Len(t, jobQuery.logs, 1)
}

func TestExecutor_Execute_CommandHook_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	command := "echo 'test'"
	hookQuery := &mockHookQuery{
		hooks: []*ent.TaskHook{
			{
				ID:      uuid.New(),
				Type:    model.HookTypeCommand,
				OnError: model.HookOnErrorIgnore,
				Config: &model.HookConfig{
					Command: &command,
				},
			},
		},
	}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	err := exec.Execute(context.Background(), task, job, model.HookEventOnSuccess, nil)
	assert.NoError(t, err)
	assert.Len(t, jobQuery.logs, 1)
}

func TestExecutor_OnErrorStrategies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tests := []struct {
		name     string
		hookType model.HookType
		onError  model.HookOnError
		wantErr  bool
		errType  string
	}{
		{"HTTP_Cancel", model.HookTypeHTTP, model.HookOnErrorCancel, true, "cancel"},
		{"HTTP_Fatal", model.HookTypeHTTP, model.HookOnErrorFatal, true, "fatal"},
		{"HTTP_Ignore", model.HookTypeHTTP, model.HookOnErrorIgnore, false, ""},
		{"Command_Cancel", model.HookTypeCommand, model.HookOnErrorCancel, true, "cancel"},
		{"Command_Fatal", model.HookTypeCommand, model.HookOnErrorFatal, true, "fatal"},
		{"Command_Ignore", model.HookTypeCommand, model.HookOnErrorIgnore, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hookType == model.HookTypeCommand && runtime.GOOS == "windows" {
				t.Skip("Skipping on Windows")
			}

			hookID := uuid.New()
			config := &model.HookConfig{}
			if tt.hookType == model.HookTypeHTTP {
				config.URL = &server.URL
			} else {
				cmd := "exit 1"
				config.Command = &cmd
			}

			hookQuery := &mockHookQuery{
				hooks: []*ent.TaskHook{
					{
						ID:      hookID,
						Type:    tt.hookType,
						OnError: tt.onError,
						Config:  config,
					},
				},
			}
			jobQuery := &mockJobQuery{}
			enabled := true
			exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

			err := exec.Execute(context.Background(), createTestTask(), createTestJob(), model.HookEventOnSuccess, nil)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType == "cancel" {
					var cancelErr *CancelError
					assert.True(t, errors.As(err, &cancelErr))
					assert.Equal(t, hookID, cancelErr.HookID)
				} else if tt.errType == "fatal" {
					var fatalErr *FatalError
					assert.True(t, errors.As(err, &fatalErr))
					assert.Equal(t, hookID, fatalErr.HookID)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutor_Execute_Events(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	events := []model.HookEvent{
		model.HookEventOnStart,
		model.HookEventOnSuccess,
		model.HookEventOnFailure,
		model.HookEventOnEnd,
	}

	for _, event := range events {
		t.Run(string(event), func(t *testing.T) {
			hookQuery := &mockHookQuery{
				hooks: []*ent.TaskHook{
					{
						ID:      uuid.New(),
						Type:    model.HookTypeHTTP,
						OnError: model.HookOnErrorIgnore,
						Config:  &model.HookConfig{URL: &server.URL},
					},
				},
			}
			jobQuery := &mockJobQuery{}
			enabled := true
			exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

			err := exec.Execute(context.Background(), createTestTask(), createTestJob(), event, nil)
			assert.NoError(t, err)
		})
	}
}

func TestExecutor_Execute_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	url := server.URL
	hookQuery := &mockHookQuery{
		hooks: []*ent.TaskHook{
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{URL: &url},
			},
		},
	}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := exec.Execute(ctx, task, job, model.HookEventOnSuccess, nil)
	assert.NoError(t, err)
}

func TestExecutor_Execute_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	url := server.URL
	hookQuery := &mockHookQuery{
		hooks: []*ent.TaskHook{
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{URL: &url},
			},
		},
	}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := exec.Execute(ctx, task, job, model.HookEventOnSuccess, nil)
	assert.NoError(t, err)
}

func TestExecutor_Execute_CommandContextCancelled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	command := "sleep 10"
	hookQuery := &mockHookQuery{
		hooks: []*ent.TaskHook{
			{
				ID:      uuid.New(),
				Type:    model.HookTypeCommand,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{Command: &command},
			},
		},
	}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 1)

	task := createTestTask()
	job := createTestJob()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := exec.Execute(ctx, task, job, model.HookEventOnSuccess, nil)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, elapsed, 2*time.Second)
}

func TestExecutor_Concurrent_Execute(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	url := server.URL
	hookQuery := &mockHookQuery{
		hooks: []*ent.TaskHook{
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{URL: &url},
			},
		},
	}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	concurrency := 10
	done := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			err := exec.Execute(context.Background(), task, job, model.HookEventOnSuccess, nil)
			done <- err
		}()
	}

	for i := 0; i < concurrency; i++ {
		err := <-done
		assert.NoError(t, err)
	}

	assert.Equal(t, concurrency, requestCount)
}

func TestNewExecutor_Config(t *testing.T) {
	hookQuery := &mockHookQuery{}
	jobQuery := &mockJobQuery{}
	enabled := true

	t.Run("HTTPClientConfig", func(t *testing.T) {
		exec := NewExecutor(hookQuery, jobQuery, &enabled, 60)
		require.NotNil(t, exec)
		e, ok := exec.(*executor)
		require.True(t, ok)
		assert.Equal(t, 60*time.Second, e.httpClient.Timeout)
	})

	t.Run("DefaultTimeout", func(t *testing.T) {
		exec := NewExecutor(hookQuery, jobQuery, &enabled, 0)
		require.NotNil(t, exec)
		e, ok := exec.(*executor)
		require.True(t, ok)
		assert.Equal(t, 30*time.Second, e.httpClient.Timeout)
	})
}

func TestExecutor_Execute_SyncErrInContext(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		receivedBody = string(body[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	url := server.URL
	body := `{"error": "{{.Error}}"}`
	hookQuery := &mockHookQuery{
		hooks: []*ent.TaskHook{
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{URL: &url, Body: &body},
			},
		},
	}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()
	syncErr := errors.New("connection timeout: remote server unavailable")

	err := exec.Execute(context.Background(), task, job, model.HookEventOnFailure, syncErr)
	assert.NoError(t, err)
	assert.Contains(t, receivedBody, "connection timeout: remote server unavailable")
}

func TestExecutor_Execute_MultiHook_FatalStopsExecution(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	url := server.URL
	fatalHookID := uuid.New()
	hookQuery := &mockHookQuery{
		hooks: []*ent.TaskHook{
			{
				ID:      fatalHookID,
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorFatal,
				Config:  &model.HookConfig{URL: &url},
			},
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{URL: &url},
			},
		},
	}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	err := exec.Execute(context.Background(), task, job, model.HookEventOnSuccess, nil)
	assert.Error(t, err)

	var fatalErr *FatalError
	assert.True(t, errors.As(err, &fatalErr))
	assert.Equal(t, fatalHookID, fatalErr.HookID)
	assert.Equal(t, 1, callCount)
	assert.Len(t, jobQuery.logs, 1)
}

func TestExecutor_Execute_MultiHook_CancelStopsExecution(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	url := server.URL
	cancelHookID := uuid.New()
	hookQuery := &mockHookQuery{
		hooks: []*ent.TaskHook{
			{
				ID:      cancelHookID,
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorCancel,
				Config:  &model.HookConfig{URL: &url},
			},
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{URL: &url},
			},
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{URL: &url},
			},
		},
	}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	err := exec.Execute(context.Background(), task, job, model.HookEventOnSuccess, nil)
	assert.Error(t, err)

	var cancelErr *CancelError
	assert.True(t, errors.As(err, &cancelErr))
	assert.Equal(t, cancelHookID, cancelErr.HookID)
	assert.Equal(t, 1, callCount)
	assert.Len(t, jobQuery.logs, 1)
}

func TestExecutor_Execute_MultiHook_IgnoreContinuesExecution(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	url := server.URL
	hookQuery := &mockHookQuery{
		hooks: []*ent.TaskHook{
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{URL: &url},
			},
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{URL: &url},
			},
			{
				ID:      uuid.New(),
				Type:    model.HookTypeHTTP,
				OnError: model.HookOnErrorIgnore,
				Config:  &model.HookConfig{URL: &url},
			},
		},
	}
	jobQuery := &mockJobQuery{}
	enabled := true

	exec := NewExecutor(hookQuery, jobQuery, &enabled, 30)

	task := createTestTask()
	job := createTestJob()

	err := exec.Execute(context.Background(), task, job, model.HookEventOnSuccess, nil)
	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
	assert.Len(t, jobQuery.logs, 3)
}

func createTestTask() *ent.Task {
	return &ent.Task{
		ID:           uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Name:         "test-task",
		SourcePath:   "/local/path",
		RemotePath:   "remote:backup",
		Direction:    model.SyncDirectionUpload,
		ConnectionID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
	}
}

func createTestJob() *ent.Job {
	return &ent.Job{
		ID:               uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		TaskID:           uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Status:           model.JobStatusSuccess,
		Trigger:          model.JobTriggerManual,
		StartTime:        time.Date(2025, 1, 17, 10, 0, 0, 0, time.UTC),
		EndTime:          time.Date(2025, 1, 17, 10, 1, 30, 0, time.UTC),
		FilesTransferred: 100,
		BytesTransferred: 1048576,
		FilesDeleted:     5,
		ErrorCount:       0,
	}
}
