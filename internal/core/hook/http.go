package hook

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

// ErrUpstreamError is returned when the webhook upstream returns an error status code.
var ErrUpstreamError = errors.New("upstream error")

func (e *executor) executeHTTP(ctx context.Context, h *ent.TaskHook, hookCtx *Context) (int64, error) {
	cfg := h.Config

	if cfg.URL == nil || *cfg.URL == "" {
		return -1, ErrURLMissing
	}

	renderedURL, err := RenderTemplate(*cfg.URL, hookCtx)
	if err != nil {
		return -1, NewInvalidTemplateError(err)
	}

	method := "POST"
	if cfg.Method != nil && *cfg.Method != "" {
		method = *cfg.Method
	}

	var bodyReader io.Reader
	if cfg.Body != nil && *cfg.Body != "" {
		renderedBody, err := RenderTemplate(*cfg.Body, hookCtx)
		if err != nil {
			return -1, NewInvalidTemplateError(err)
		}
		bodyReader = strings.NewReader(renderedBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, renderedURL, bodyReader)
	if err != nil {
		return -1, i18n.NewI18nError(i18n.ErrGeneric).WithCause(err)
	}

	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range cfg.Headers {
		renderedValue, err := RenderTemplate(v, hookCtx)
		if err != nil {
			return -1, NewInvalidTemplateError(err)
		}
		req.Header.Set(k, renderedValue)
	}

	httpClient := e.httpClient
	if cfg.Timeout != nil && *cfg.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(*cfg.Timeout)*time.Second)
		defer cancel()
		req = req.WithContext(ctx)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return -1, i18n.NewI18nError(i18n.ErrGeneric).WithCause(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))

	if resp.StatusCode >= 400 {
		return int64(-resp.StatusCode), NewHTTPFailedError(resp.StatusCode)
	}

	return 0, nil
}
