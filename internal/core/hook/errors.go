// Package hook provides task event hook execution capabilities.
package hook

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

var (
	// ErrCommandMissing is returned when a Command hook is missing its command string.
	ErrCommandMissing = i18n.NewI18nError(i18n.ErrMissingParameter).WithData(map[string]interface{}{"Parameter": "command"})
	// ErrURLMissing is returned when an HTTP hook is missing its URL.
	ErrURLMissing = i18n.NewI18nError(i18n.ErrMissingParameter).WithData(map[string]interface{}{"Parameter": "url"})
	// ErrConfigRequired is returned when hook configuration is nil.
	ErrConfigRequired = i18n.NewI18nError(i18n.ErrMissingParameter).WithData(map[string]interface{}{"Parameter": "config"})
	// ErrTimeoutOutOfRange is returned when the timeout value is invalid.
	ErrTimeoutOutOfRange = i18n.NewI18nError(i18n.ErrValidationFailed).WithCause(errTimeoutOutOfRangeStatic)
)

var errTimeoutOutOfRangeStatic = errors.New("timeout must be between 1 and 3600 seconds")

// NewTimeoutError creates a new i18n timeout error.
func NewTimeoutError(timeout interface{}) *i18n.Error {
	return i18n.NewI18nError(i18n.ErrHookTimeout).WithData(map[string]interface{}{"Timeout": timeout})
}

// NewHTTPFailedError creates a new i18n HTTP failure error.
func NewHTTPFailedError(status interface{}) *i18n.Error {
	return i18n.NewI18nError(i18n.ErrHookHTTPFailed).WithData(map[string]interface{}{"Status": status})
}

// NewCommandFailedError creates a new i18n command failure error.
func NewCommandFailedError(exitCode interface{}) *i18n.Error {
	return i18n.NewI18nError(i18n.ErrHookCommandFailed).WithData(map[string]interface{}{"ExitCode": exitCode})
}

// NewInvalidURLError creates a new i18n invalid URL error.
func NewInvalidURLError(err error) *i18n.Error {
	return i18n.NewI18nError(i18n.ErrHookInvalidURL).WithCause(err)
}

// NewInvalidTemplateError creates a new i18n invalid template error.
func NewInvalidTemplateError(err error) *i18n.Error {
	return i18n.NewI18nError(i18n.ErrHookInvalidTemplate).WithData(map[string]interface{}{"Error": err.Error()}).WithCause(err)
}

// CancelError indicates that a hook has requested to cancel the sync operation.
type CancelError struct {
	HookID uuid.UUID
	Cause  error
}

// Error implements the error interface.
func (e *CancelError) Error() string {
	return fmt.Sprintf("hook %s requested cancel: %v", e.HookID, e.Cause)
}

// Unwrap returns the original error.
func (e *CancelError) Unwrap() error { return e.Cause }

// I18nError converts the error to a translatable I18nError.
func (e *CancelError) I18nError() *i18n.Error {
	return i18n.NewI18nError(i18n.ErrHookCancelled).
		WithData(map[string]interface{}{"HookID": e.HookID.String()}).
		WithCause(e.Cause)
}

// FatalError indicates that a hook failed fatally, stopping further hook execution.
type FatalError struct {
	HookID uuid.UUID
	Cause  error
}

// Error implements the error interface.
func (e *FatalError) Error() string {
	return fmt.Sprintf("hook %s failed fatally: %v", e.HookID, e.Cause)
}

// Unwrap returns the original error.
func (e *FatalError) Unwrap() error { return e.Cause }

// I18nError converts the error to a translatable I18nError.
func (e *FatalError) I18nError() *i18n.Error {
	return i18n.NewI18nError(i18n.ErrHookFatal).
		WithData(map[string]interface{}{"HookID": e.HookID.String()}).
		WithCause(e.Cause)
}
