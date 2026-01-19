package hook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

func TestExecuteHTTP(t *testing.T) {
	exec := createTestExecutor()
	ctx := createTestContext()

	tests := []struct {
		name        string
		setupServer func(t *testing.T) (*httptest.Server, *struct {
			method  string
			body    string
			headers http.Header
		})
		hookConfig  func(url string) *model.HookConfig
		wantErr     bool
		errContains string
		check       func(t *testing.T, code int64, err error, captured *struct {
			method  string
			body    string
			headers http.Header
		})
	}{
		{
			name: "BasicRequest",
			setupServer: func(t *testing.T) (*httptest.Server, *struct {
				method  string
				body    string
				headers http.Header
			}) {
				captured := &struct {
					method  string
					body    string
					headers http.Header
				}{}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					captured.method = r.Method
					captured.headers = r.Header.Clone()
					body, _ := io.ReadAll(r.Body)
					captured.body = string(body)
					w.WriteHeader(http.StatusOK)
				}))
				return server, captured
			},
			hookConfig: func(url string) *model.HookConfig {
				method := "POST"
				body := `{"task": "{{.Task.Name}}", "event": "{{.Event}}"}`
				return &model.HookConfig{
					URL:    &url,
					Method: &method,
					Body:   &body,
				}
			},
			check: func(t *testing.T, code int64, err error, captured *struct {
				method  string
				body    string
				headers http.Header
			}) {
				require.NoError(t, err)
				assert.Equal(t, int64(0), code)
				assert.Equal(t, "POST", captured.method)
				assert.Contains(t, captured.body, `"task": "test-task"`)
				assert.Contains(t, captured.body, `"event": "ON_SUCCESS"`)
				assert.Equal(t, "application/json", captured.headers.Get("Content-Type"))
			},
		},
		{
			name: "CustomHeaders",
			setupServer: func(t *testing.T) (*httptest.Server, *struct {
				method  string
				body    string
				headers http.Header
			}) {
				captured := &struct {
					method  string
					body    string
					headers http.Header
				}{}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					captured.headers = r.Header.Clone()
					w.WriteHeader(http.StatusOK)
				}))
				return server, captured
			},
			hookConfig: func(url string) *model.HookConfig {
				return &model.HookConfig{
					URL: &url,
					Headers: map[string]string{
						"X-Task-ID":     "{{.Task.ID}}",
						"X-Event-Type":  "{{.Event}}",
						"Authorization": "Bearer secret-token",
					},
				}
			},
			check: func(t *testing.T, code int64, err error, captured *struct {
				method  string
				body    string
				headers http.Header
			}) {
				require.NoError(t, err)
				assert.Equal(t, "11111111-1111-1111-1111-111111111111", captured.headers.Get("X-Task-ID"))
				assert.Equal(t, "ON_SUCCESS", captured.headers.Get("X-Event-Type"))
				assert.Equal(t, "Bearer secret-token", captured.headers.Get("Authorization"))
			},
		},
		{
			name: "ErrorOnMissingURL",
			hookConfig: func(url string) *model.HookConfig {
				return &model.HookConfig{URL: nil}
			},
			wantErr:     true,
			errContains: i18n.ErrMissingParameter,
		},
		{
			name: "ErrorOnInvalidTemplate",
			hookConfig: func(url string) *model.HookConfig {
				u := "http://example.com/{{.Invalid"
				return &model.HookConfig{URL: &u}
			},
			wantErr:     true,
			errContains: i18n.ErrHookInvalidTemplate,
		},
		{
			name: "ErrorOn4xxResponse",
			setupServer: func(t *testing.T) (*httptest.Server, *struct {
				method  string
				body    string
				headers http.Header
			}) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				}))
				return server, nil
			},
			hookConfig: func(url string) *model.HookConfig {
				return &model.HookConfig{URL: &url}
			},
			wantErr:     true,
			errContains: i18n.ErrHookHTTPFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var url string
			var captured *struct {
				method  string
				body    string
				headers http.Header
			}
			if tt.setupServer != nil {
				server, c := tt.setupServer(t)
				url = server.URL
				captured = c
				defer server.Close()
			}

			hook := &ent.TaskHook{
				Type:   model.HookTypeHTTP,
				Config: tt.hookConfig(url),
			}

			code, err := exec.executeHTTP(context.Background(), hook, ctx)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			if tt.check != nil {
				tt.check(t, code, err, captured)
			}
		})
	}
}

func TestExecuteHTTP_Functional(t *testing.T) {
	exec := createTestExecutor()
	ctx := createTestContext()

	t.Run("TemplateRendering", func(t *testing.T) {
		var receivedBody string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			receivedBody = string(body)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		url := server.URL
		body := `{"summary": "{{Summary .}}", "duration": "{{FormatDuration .Duration}}"}`
		hook := &ent.TaskHook{
			Type:   model.HookTypeHTTP,
			Config: &model.HookConfig{URL: &url, Body: &body},
		}

		_, err := exec.executeHTTP(context.Background(), hook, ctx)
		require.NoError(t, err)
		assert.Contains(t, receivedBody, "test-task")
		assert.Contains(t, receivedBody, "1m30s")
	})

	t.Run("Timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		e := &executor{
			httpClient: &http.Client{Timeout: 50 * time.Millisecond},
			enabled:    ptrBool(true),
		}
		url := server.URL
		hook := &ent.TaskHook{
			Type:   model.HookTypeHTTP,
			Config: &model.HookConfig{URL: &url},
		}

		_, err := e.executeHTTP(context.Background(), hook, ctx)
		assert.Error(t, err)
	})
}

func createTestExecutor() *executor {
	return &executor{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		enabled:        ptrBool(true),
		defaultTimeout: 30,
	}
}
