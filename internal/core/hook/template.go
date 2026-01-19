package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"
)

var hookFuncMap = template.FuncMap{
	"FormatTime":      formatTime,
	"FormatDuration":  formatDuration,
	"FormatSizeBytes": formatSizeBytes,
	"JsonMarshal":     jsonMarshal,
	"Summary":         generateSummary,
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

func formatDuration(d time.Duration) string {
	return d.Round(time.Second).String()
}

func formatSizeBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func jsonMarshal(v any) string {
	out, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(out)
}

func generateSummary(ctx *Context) string {
	return fmt.Sprintf("Task %s %s: %d files (%s) in %s",
		ctx.Task.Name,
		ctx.Event,
		ctx.Stats.FilesTransferred,
		formatSizeBytes(ctx.Stats.BytesTransferred),
		ctx.Duration.Round(time.Second).String(),
	)
}

// RenderTemplate renders a template string using the provided HookContext.
func RenderTemplate(tmplStr string, ctx *Context) (string, error) {
	tmpl, err := template.New("hook").Funcs(hookFuncMap).Parse(tmplStr)
	if err != nil {
		return "", NewInvalidTemplateError(err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", NewInvalidTemplateError(err)
	}
	return buf.String(), nil
}
