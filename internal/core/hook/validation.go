package hook

import (
	"net/url"
	"strings"
	"text/template"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
)

// ValidateHookConfig validates the hook configuration.
func ValidateHookConfig(config *model.HookConfigInput, hookType model.HookType) error {
	if config == nil {
		return ErrConfigRequired
	}

	switch hookType {
	case model.HookTypeHTTP:
		if config.URL == nil || *config.URL == "" {
			return ErrURLMissing
		}

		if _, err := template.New("url").Funcs(hookFuncMap).Parse(*config.URL); err != nil {
			return NewInvalidTemplateError(err)
		}

		if !strings.Contains(*config.URL, "{{") && !strings.Contains(*config.URL, "}}") {
			if _, err := url.ParseRequestURI(*config.URL); err != nil {
				return NewInvalidURLError(err)
			}
		}

		for _, value := range config.Headers {
			if _, err := template.New("header").Funcs(hookFuncMap).Parse(value); err != nil {
				return NewInvalidTemplateError(err)
			}
		}

		if config.Body != nil && *config.Body != "" {
			if _, err := template.New("body").Funcs(hookFuncMap).Parse(*config.Body); err != nil {
				return NewInvalidTemplateError(err)
			}
		}
	case model.HookTypeCommand:
		if config.Command == nil || *config.Command == "" {
			return ErrCommandMissing
		}

		if _, err := template.New("command").Funcs(hookFuncMap).Parse(*config.Command); err != nil {
			return NewInvalidTemplateError(err)
		}
	}

	// Validate timeout (1-3600 seconds)
	if config.Timeout != nil {
		if *config.Timeout < 1 || *config.Timeout > 3600 {
			return ErrTimeoutOutOfRange
		}
	}

	return nil
}
