package resolver_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/ent/joblog"
	"github.com/xzzpig/rclone-sync/internal/core/hook"
)

type HookTestSuite struct {
	ResolverTestSuite
}

func TestHookSuite(t *testing.T) {
	suite.Run(t, new(HookTestSuite))
}

func (s *HookTestSuite) TestCreateHTTPHook() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-test-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-test-task", connID)

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook {
				create(taskId: $taskId, connectionId: $connectionId, input: $input) {
					id
					enabled
					event
					type
					onError
					config {
						url
						method
						body
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"taskId": task.ID.String(),
		"input": map[string]interface{}{
			"event":   "ON_SUCCESS",
			"type":    "HTTP",
			"onError": "IGNORE",
			"config": map[string]interface{}{
				"url":    "https://example.com/webhook",
				"method": "POST",
				"body":   `{"task": "{{.Task.Name}}"}`,
			},
		},
	})

	require.Empty(s.T(), resp.Errors)
	hookData := gjson.Get(string(resp.Data), "hook.create")
	assert.True(s.T(), hookData.Get("enabled").Bool())
	assert.Equal(s.T(), "ON_SUCCESS", hookData.Get("event").String())
	assert.Equal(s.T(), "HTTP", hookData.Get("type").String())
	assert.Equal(s.T(), "IGNORE", hookData.Get("onError").String())
	assert.Equal(s.T(), "https://example.com/webhook", hookData.Get("config.url").String())
	assert.Equal(s.T(), "POST", hookData.Get("config.method").String())
}

func (s *HookTestSuite) TestCreateCommandHook() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-test-conn-2")
	task := s.Env.CreateTestTask(s.T(), "hook-test-task-2", connID)

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook {
				create(taskId: $taskId, connectionId: $connectionId, input: $input) {
					id
					enabled
					event
					type
					onError
					config {
						command
						workDir
						timeout
					}
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"taskId": task.ID.String(),
		"input": map[string]interface{}{
			"event":   "ON_FAILURE",
			"type":    "COMMAND",
			"onError": "IGNORE",
			"config": map[string]interface{}{
				"command": "echo 'Task {{.Task.Name}} failed'",
				"workDir": "/tmp",
				"timeout": 30,
			},
		},
	})

	require.Empty(s.T(), resp.Errors)
	hookData := gjson.Get(string(resp.Data), "hook.create")
	assert.True(s.T(), hookData.Get("enabled").Bool())
	assert.Equal(s.T(), "ON_FAILURE", hookData.Get("event").String())
	assert.Equal(s.T(), "COMMAND", hookData.Get("type").String())
	assert.Equal(s.T(), "echo 'Task {{.Task.Name}} failed'", hookData.Get("config.command").String())
	assert.Equal(s.T(), "/tmp", hookData.Get("config.workDir").String())
	assert.Equal(s.T(), int64(30), hookData.Get("config.timeout").Int())
}

func (s *HookTestSuite) TestListHooksByTaskId() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-list-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-list-task", connID)

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { create(taskId: $taskId, connectionId: $connectionId, input: $input) { id } }
		}
	`

	url := "https://example.com/hook1"
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"taskId": task.ID.String(),
		"input": map[string]interface{}{
			"event":  "ON_SUCCESS",
			"type":   "HTTP",
			"config": map[string]interface{}{"url": url},
		},
	})
	require.Empty(s.T(), resp.Errors)

	url2 := "https://example.com/hook2"
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"taskId": task.ID.String(),
		"input": map[string]interface{}{
			"event":  "ON_FAILURE",
			"type":   "HTTP",
			"config": map[string]interface{}{"url": url2},
		},
	})
	require.Empty(s.T(), resp.Errors)

	listQuery := `

		query($taskId: ID!) {
			hook {
				list(taskId: $taskId) {
					id
					event
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), listQuery, map[string]interface{}{
		"taskId": task.ID.String(),
	})
	require.Empty(s.T(), resp.Errors)
	hooks := gjson.Get(string(resp.Data), "hook.list").Array()
	assert.Len(s.T(), hooks, 2)
}

func (s *HookTestSuite) TestListHooksValidation() {
	listQuery := `
		query {
			hook {
				list {
					id
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQL(s.T(), GraphQLRequest{
		Query: listQuery,
	})
	assert.NotEmpty(s.T(), resp.Errors)
}

func (s *HookTestSuite) TestUpdateHook() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-update-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-update-task", connID)

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { create(taskId: $taskId, connectionId: $connectionId, input: $input) { id } }
		}
	`

	url := "https://example.com/original"
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"taskId": task.ID.String(),
		"input": map[string]interface{}{
			"event":  "ON_SUCCESS",
			"type":   "HTTP",
			"config": map[string]interface{}{"url": url},
		},
	})

	require.Empty(s.T(), resp.Errors)
	hookID := gjson.Get(string(resp.Data), "hook.create.id").String()

	updateMutation := `
		mutation($id: ID!, $input: UpdateHookInput!) {
			hook {
				update(id: $id, input: $input) {
					id
					enabled
					event
					config { url }
				}
			}
		}
	`

	newURL := "https://example.com/updated"
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), updateMutation, map[string]interface{}{
		"id": hookID,
		"input": map[string]interface{}{
			"enabled": false,
			"event":   "ON_END",
			"config":  map[string]interface{}{"url": newURL},
		},
	})
	require.Empty(s.T(), resp.Errors)
	hookData := gjson.Get(string(resp.Data), "hook.update")
	assert.False(s.T(), hookData.Get("enabled").Bool())
	assert.Equal(s.T(), "ON_END", hookData.Get("event").String())
	assert.Equal(s.T(), newURL, hookData.Get("config.url").String())
}

func (s *HookTestSuite) TestDeleteHook() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-delete-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-delete-task", connID)

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { create(taskId: $taskId, connectionId: $connectionId, input: $input) { id } }
		}
	`

	url := "https://example.com/to-delete"
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"taskId": task.ID.String(),
		"input": map[string]interface{}{
			"event":  "ON_SUCCESS",
			"type":   "HTTP",
			"config": map[string]interface{}{"url": url},
		},
	})

	require.Empty(s.T(), resp.Errors)
	hookID := gjson.Get(string(resp.Data), "hook.create.id").String()

	deleteMutation := `
		mutation($id: ID!) {
			hook { delete(id: $id) { id } }
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), deleteMutation, map[string]interface{}{
		"id": hookID,
	})
	require.Empty(s.T(), resp.Errors)
	assert.Equal(s.T(), hookID, gjson.Get(string(resp.Data), "hook.delete.id").String())

	getQuery := `
		query($id: ID!) {
			hook { get(id: $id) { id } }
		}
	`
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), getQuery, map[string]interface{}{
		"id": hookID,
	})
	require.Empty(s.T(), resp.Errors)
	hookData := gjson.Get(string(resp.Data), "hook.get")
	assert.True(s.T(), hookData.Type == gjson.Null || !hookData.Exists())
}

func (s *HookTestSuite) TestHookExecutorTriggersHTTPHook() {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	connID := s.Env.CreateTestConnection(s.T(), "hook-exec-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-exec-task", connID)

	url := server.URL
	_, err := s.Env.Client.TaskHook.Create().
		SetTaskID(task.ID).
		SetEnabled(true).
		SetEvent(model.HookEventOnSuccess).
		SetType(model.HookTypeHTTP).
		SetOnError(model.HookOnErrorIgnore).
		SetConfig(&model.HookConfig{
			URL:    &url,
			Method: ptrString("POST"),
		}).
		Save(context.Background())
	require.NoError(s.T(), err)

	cfg := &config.Config{}
	cfg.App.Hook.Enabled = true
	cfg.App.Hook.DefaultTimeout = 30

	executor := hook.NewExecutor(s.Env.HookQuery, s.Env.JobQuery, &cfg.App.Hook.Enabled, cfg.App.Hook.DefaultTimeout)

	taskWithConn, err := s.Env.TaskQuery.GetTaskWithConnection(context.Background(), task.ID)
	require.NoError(s.T(), err)

	job, err := s.Env.JobQuery.CreateJob(context.Background(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	err = executor.Execute(context.Background(), taskWithConn, job, model.HookEventOnSuccess, nil)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), int32(1), atomic.LoadInt32(&callCount))
}

func (s *HookTestSuite) TestHookExecutionLoggedToJobLog() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	connID := s.Env.CreateTestConnection(s.T(), "hook-log-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-log-task", connID)

	url := server.URL
	createdHook, err := s.Env.Client.TaskHook.Create().
		SetTaskID(task.ID).
		SetEnabled(true).
		SetEvent(model.HookEventOnSuccess).
		SetType(model.HookTypeHTTP).
		SetOnError(model.HookOnErrorIgnore).
		SetConfig(&model.HookConfig{
			URL:    &url,
			Method: ptrString("POST"),
		}).
		Save(context.Background())
	require.NoError(s.T(), err)

	cfg := &config.Config{}
	cfg.App.Hook.Enabled = true
	cfg.App.Hook.DefaultTimeout = 30

	executor := hook.NewExecutor(s.Env.HookQuery, s.Env.JobQuery, &cfg.App.Hook.Enabled, cfg.App.Hook.DefaultTimeout)

	taskWithConn, err := s.Env.TaskQuery.GetTaskWithConnection(context.Background(), task.ID)
	require.NoError(s.T(), err)

	job, err := s.Env.JobQuery.CreateJob(context.Background(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	err = executor.Execute(context.Background(), taskWithConn, job, model.HookEventOnSuccess, nil)
	require.NoError(s.T(), err)

	logs, err := s.Env.Client.JobLog.Query().
		Where(joblog.JobIDEQ(job.ID), joblog.WhatEQ(model.LogActionHook)).
		All(context.Background())
	require.NoError(s.T(), err)
	require.Len(s.T(), logs, 1)

	assert.Equal(s.T(), model.LogLevelInfo, logs[0].Level)
	assert.Contains(s.T(), logs[0].Path, createdHook.ID.String())
	assert.Contains(s.T(), logs[0].Path, "on_success")
	assert.True(s.T(), logs[0].Size >= 0)
}

func (s *HookTestSuite) TestDisabledHookNotTriggered() {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	connID := s.Env.CreateTestConnection(s.T(), "hook-disabled-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-disabled-task", connID)

	url := server.URL
	_, err := s.Env.Client.TaskHook.Create().
		SetTaskID(task.ID).
		SetEnabled(false).
		SetEvent(model.HookEventOnSuccess).
		SetType(model.HookTypeHTTP).
		SetOnError(model.HookOnErrorIgnore).
		SetConfig(&model.HookConfig{
			URL:    &url,
			Method: ptrString("POST"),
		}).
		Save(context.Background())
	require.NoError(s.T(), err)

	cfg := &config.Config{}
	cfg.App.Hook.Enabled = true
	cfg.App.Hook.DefaultTimeout = 30

	executor := hook.NewExecutor(s.Env.HookQuery, s.Env.JobQuery, &cfg.App.Hook.Enabled, cfg.App.Hook.DefaultTimeout)

	taskWithConn, err := s.Env.TaskQuery.GetTaskWithConnection(context.Background(), task.ID)
	require.NoError(s.T(), err)

	job, err := s.Env.JobQuery.CreateJob(context.Background(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	err = executor.Execute(context.Background(), taskWithConn, job, model.HookEventOnSuccess, nil)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), int32(0), atomic.LoadInt32(&callCount))
}

func ptrString(s string) *string {
	return &s
}

func (s *HookTestSuite) TestOnStartHookExecutesBeforeSync() {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	connID := s.Env.CreateTestConnection(s.T(), "hook-onstart-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-onstart-task", connID)

	url := server.URL
	_, err := s.Env.Client.TaskHook.Create().
		SetTaskID(task.ID).
		SetEnabled(true).
		SetEvent(model.HookEventOnStart).
		SetType(model.HookTypeHTTP).
		SetOnError(model.HookOnErrorIgnore).
		SetConfig(&model.HookConfig{
			URL:    &url,
			Method: ptrString("POST"),
		}).
		Save(context.Background())
	require.NoError(s.T(), err)

	cfg := &config.Config{}
	cfg.App.Hook.Enabled = true
	cfg.App.Hook.DefaultTimeout = 30

	executor := hook.NewExecutor(s.Env.HookQuery, s.Env.JobQuery, &cfg.App.Hook.Enabled, cfg.App.Hook.DefaultTimeout)

	taskWithConn, err := s.Env.TaskQuery.GetTaskWithConnection(context.Background(), task.ID)
	require.NoError(s.T(), err)

	job, err := s.Env.JobQuery.CreateJob(context.Background(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	err = executor.Execute(context.Background(), taskWithConn, job, model.HookEventOnStart, nil)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), int32(1), atomic.LoadInt32(&callCount))
}

func (s *HookTestSuite) TestOnStartHookCancelBehavior() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	connID := s.Env.CreateTestConnection(s.T(), "hook-cancel-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-cancel-task", connID)

	url := server.URL
	_, err := s.Env.Client.TaskHook.Create().
		SetTaskID(task.ID).
		SetEnabled(true).
		SetEvent(model.HookEventOnStart).
		SetType(model.HookTypeHTTP).
		SetOnError(model.HookOnErrorCancel).
		SetConfig(&model.HookConfig{
			URL:    &url,
			Method: ptrString("POST"),
		}).
		Save(context.Background())
	require.NoError(s.T(), err)

	cfg := &config.Config{}
	cfg.App.Hook.Enabled = true
	cfg.App.Hook.DefaultTimeout = 30

	executor := hook.NewExecutor(s.Env.HookQuery, s.Env.JobQuery, &cfg.App.Hook.Enabled, cfg.App.Hook.DefaultTimeout)

	taskWithConn, err := s.Env.TaskQuery.GetTaskWithConnection(context.Background(), task.ID)
	require.NoError(s.T(), err)

	job, err := s.Env.JobQuery.CreateJob(context.Background(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	err = executor.Execute(context.Background(), taskWithConn, job, model.HookEventOnStart, nil)

	var cancelErr *hook.CancelError
	assert.True(s.T(), errors.As(err, &cancelErr))
}

func (s *HookTestSuite) TestOnStartHookFatalBehavior() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	connID := s.Env.CreateTestConnection(s.T(), "hook-fatal-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-fatal-task", connID)

	url := server.URL
	_, err := s.Env.Client.TaskHook.Create().
		SetTaskID(task.ID).
		SetEnabled(true).
		SetEvent(model.HookEventOnStart).
		SetType(model.HookTypeHTTP).
		SetOnError(model.HookOnErrorFatal).
		SetConfig(&model.HookConfig{
			URL:    &url,
			Method: ptrString("POST"),
		}).
		Save(context.Background())
	require.NoError(s.T(), err)

	cfg := &config.Config{}
	cfg.App.Hook.Enabled = true
	cfg.App.Hook.DefaultTimeout = 30

	executor := hook.NewExecutor(s.Env.HookQuery, s.Env.JobQuery, &cfg.App.Hook.Enabled, cfg.App.Hook.DefaultTimeout)

	taskWithConn, err := s.Env.TaskQuery.GetTaskWithConnection(context.Background(), task.ID)
	require.NoError(s.T(), err)

	job, err := s.Env.JobQuery.CreateJob(context.Background(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	err = executor.Execute(context.Background(), taskWithConn, job, model.HookEventOnStart, nil)

	var fatalErr *hook.FatalError
	assert.True(s.T(), errors.As(err, &fatalErr))
}

func (s *HookTestSuite) TestOnStartHookIgnoreBehavior() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	connID := s.Env.CreateTestConnection(s.T(), "hook-ignore-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-ignore-task", connID)

	url := server.URL
	_, err := s.Env.Client.TaskHook.Create().
		SetTaskID(task.ID).
		SetEnabled(true).
		SetEvent(model.HookEventOnStart).
		SetType(model.HookTypeHTTP).
		SetOnError(model.HookOnErrorIgnore).
		SetConfig(&model.HookConfig{
			URL:    &url,
			Method: ptrString("POST"),
		}).
		Save(context.Background())
	require.NoError(s.T(), err)

	cfg := &config.Config{}
	cfg.App.Hook.Enabled = true
	cfg.App.Hook.DefaultTimeout = 30

	executor := hook.NewExecutor(s.Env.HookQuery, s.Env.JobQuery, &cfg.App.Hook.Enabled, cfg.App.Hook.DefaultTimeout)

	taskWithConn, err := s.Env.TaskQuery.GetTaskWithConnection(context.Background(), task.ID)
	require.NoError(s.T(), err)

	job, err := s.Env.JobQuery.CreateJob(context.Background(), task.ID, "MANUAL")
	require.NoError(s.T(), err)

	err = executor.Execute(context.Background(), taskWithConn, job, model.HookEventOnStart, nil)
	require.NoError(s.T(), err)
}

func (s *HookTestSuite) TestConnectionHooksResolver() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-conn-resolver")
	s.Env.CreateTestTask(s.T(), "hook-conn-task", connID)

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { create(taskId: $taskId, connectionId: $connectionId, input: $input) { id } }
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"connectionId": connID.String(),
		"input": map[string]interface{}{
			"event":   "ON_SUCCESS",
			"type":    "HTTP",
			"onError": "IGNORE",
			"config": map[string]interface{}{
				"url": "https://example.com/conn-hook",
			},
		},
	})
	require.Empty(s.T(), resp.Errors)

	connQuery := `
		query($id: ID!) {
			connection {
				get(id: $id) {
					id
					hooks {
						id
						event
						config { url }
					}
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), connQuery, map[string]interface{}{
		"id": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	hooks := gjson.Get(data, "connection.get.hooks").Array()
	assert.Len(s.T(), hooks, 1)
	assert.Equal(s.T(), "ON_SUCCESS", hooks[0].Get("event").String())
	assert.Equal(s.T(), "https://example.com/conn-hook", hooks[0].Get("config.url").String())
}

func (s *HookTestSuite) TestHookTaskResolverWithNilTaskID() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-nil-task")

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { 
				create(taskId: $taskId, connectionId: $connectionId, input: $input) { 
					id 
					task { id }
				} 
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"connectionId": connID.String(),
		"input": map[string]interface{}{
			"event":   "ON_SUCCESS",
			"type":    "HTTP",
			"onError": "IGNORE",
			"config":  map[string]interface{}{"url": "https://example.com/test"},
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	task := gjson.Get(data, "hook.create.task")
	assert.True(s.T(), task.Type == gjson.Null || !task.Exists())
}

func (s *HookTestSuite) TestHookConnectionResolverWithNilConnectionID() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-nil-conn")
	task := s.Env.CreateTestTask(s.T(), "hook-nil-conn-task", connID)

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { 
				create(taskId: $taskId, connectionId: $connectionId, input: $input) { 
					id 
					connection { id }
				} 
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"taskId": task.ID.String(),
		"input": map[string]interface{}{
			"event":   "ON_SUCCESS",
			"type":    "HTTP",
			"onError": "IGNORE",
			"config":  map[string]interface{}{"url": "https://example.com/test2"},
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	conn := gjson.Get(data, "hook.create.connection")
	assert.True(s.T(), conn.Type == gjson.Null || !conn.Exists())
}

func (s *HookTestSuite) TestListHooksByConnectionId() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-list-by-conn")

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { create(taskId: $taskId, connectionId: $connectionId, input: $input) { id } }
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"connectionId": connID.String(),
		"input": map[string]interface{}{
			"event":  "ON_SUCCESS",
			"type":   "HTTP",
			"config": map[string]interface{}{"url": "https://example.com/hook1"},
		},
	})
	require.Empty(s.T(), resp.Errors)

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"connectionId": connID.String(),
		"input": map[string]interface{}{
			"event":  "ON_FAILURE",
			"type":   "HTTP",
			"config": map[string]interface{}{"url": "https://example.com/hook2"},
		},
	})
	require.Empty(s.T(), resp.Errors)

	listQuery := `
		query($connectionId: ID!) {
			hook {
				list(connectionId: $connectionId) {
					id
					event
				}
			}
		}
	`

	resp = s.Env.ExecuteGraphQLWithVars(s.T(), listQuery, map[string]interface{}{
		"connectionId": connID.String(),
	})
	require.Empty(s.T(), resp.Errors)
	hooks := gjson.Get(string(resp.Data), "hook.list").Array()
	assert.Len(s.T(), hooks, 2)
}

func (s *HookTestSuite) TestCreateHookWithBothTaskAndConnectionId() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-both-ids")
	task := s.Env.CreateTestTask(s.T(), "hook-both-ids-task", connID)

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { create(taskId: $taskId, connectionId: $connectionId, input: $input) { id } }
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"taskId":       task.ID.String(),
		"connectionId": connID.String(),
		"input": map[string]interface{}{
			"event":  "ON_SUCCESS",
			"type":   "HTTP",
			"config": map[string]interface{}{"url": "https://example.com/both"},
		},
	})
	require.NotEmpty(s.T(), resp.Errors)
}

func (s *HookTestSuite) TestCreateHookWithNoTaskOrConnectionId() {
	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { create(taskId: $taskId, connectionId: $connectionId, input: $input) { id } }
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"event":  "ON_SUCCESS",
			"type":   "HTTP",
			"config": map[string]interface{}{"url": "https://example.com/none"},
		},
	})
	require.NotEmpty(s.T(), resp.Errors)
}

func (s *HookTestSuite) TestHookTaskResolver() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-task-resolver")
	task := s.Env.CreateTestTask(s.T(), "hook-task-resolver-task", connID)

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { 
				create(taskId: $taskId, connectionId: $connectionId, input: $input) { 
					id 
					task { 
						id 
						name 
					}
				} 
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"taskId": task.ID.String(),
		"input": map[string]interface{}{
			"event":  "ON_SUCCESS",
			"type":   "HTTP",
			"config": map[string]interface{}{"url": "https://example.com/task-resolver"},
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), task.ID.String(), gjson.Get(data, "hook.create.task.id").String())
	assert.Equal(s.T(), "hook-task-resolver-task", gjson.Get(data, "hook.create.task.name").String())
}

func (s *HookTestSuite) TestHookConnectionResolver() {
	connID := s.Env.CreateTestConnection(s.T(), "hook-conn-resolver-2")

	createHookMutation := `
		mutation($taskId: ID, $connectionId: ID, $input: HookInput!) {
			hook { 
				create(taskId: $taskId, connectionId: $connectionId, input: $input) { 
					id 
					connection { 
						id 
						name 
					}
				} 
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), createHookMutation, map[string]interface{}{
		"connectionId": connID.String(),
		"input": map[string]interface{}{
			"event":  "ON_SUCCESS",
			"type":   "HTTP",
			"config": map[string]interface{}{"url": "https://example.com/conn-resolver"},
		},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	assert.Equal(s.T(), connID.String(), gjson.Get(data, "hook.create.connection.id").String())
	assert.Equal(s.T(), "hook-conn-resolver-2", gjson.Get(data, "hook.create.connection.name").String())
}
