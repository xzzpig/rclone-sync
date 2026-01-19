package hook

import (
"errors"
"testing"

"github.com/google/uuid"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
"github.com/xzzpig/rclone-sync/internal/i18n"
)

func TestErrorFactories(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{"Timeout", NewTimeoutError(30), i18n.ErrHookTimeout},
		{"HTTPFailed", NewHTTPFailedError(500), i18n.ErrHookHTTPFailed},
		{"CommandFailed", NewCommandFailedError(1), i18n.ErrHookCommandFailed},
		{"InvalidURL", NewInvalidURLError(errors.New("invalid")), i18n.ErrHookInvalidURL},
		{"InvalidTemplate", NewInvalidTemplateError(errors.New("syntax")), i18n.ErrHookInvalidTemplate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
require.NotNil(t, tt.err)
assert.Contains(t, tt.err.Error(), tt.contains)
		})
	}
}

func TestCancelError(t *testing.T) {
	hookID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	cause := errors.New("hook requested cancel")
	cancelErr := &CancelError{HookID: hookID, Cause: cause}

	t.Run("Error", func(t *testing.T) {
errMsg := cancelErr.Error()
		assert.Contains(t, errMsg, hookID.String())
		assert.Contains(t, errMsg, "cancel")
		assert.Contains(t, errMsg, cause.Error())
	})

	t.Run("Unwrap", func(t *testing.T) {
assert.Equal(t, cause, cancelErr.Unwrap())
	})

	t.Run("I18nError", func(t *testing.T) {
i18nErr := cancelErr.I18nError()
		require.NotNil(t, i18nErr)
		assert.Contains(t, i18nErr.Error(), i18n.ErrHookCancelled)
	})

	t.Run("ErrorsIs", func(t *testing.T) {
assert.True(t, errors.Is(cancelErr, cause))
})
}

func TestFatalError(t *testing.T) {
	hookID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	cause := errors.New("fatal failure")
	fatalErr := &FatalError{HookID: hookID, Cause: cause}

	t.Run("Error", func(t *testing.T) {
errMsg := fatalErr.Error()
		assert.Contains(t, errMsg, hookID.String())
		assert.Contains(t, errMsg, "fatal")
		assert.Contains(t, errMsg, cause.Error())
	})

	t.Run("Unwrap", func(t *testing.T) {
assert.Equal(t, cause, fatalErr.Unwrap())
	})

	t.Run("I18nError", func(t *testing.T) {
i18nErr := fatalErr.I18nError()
		require.NotNil(t, i18nErr)
		assert.Contains(t, i18nErr.Error(), i18n.ErrHookFatal)
	})

	t.Run("ErrorsIs", func(t *testing.T) {
assert.True(t, errors.Is(fatalErr, cause))
})
}

func TestPackageLevelErrors(t *testing.T) {
	assert.NotNil(t, ErrCommandMissing)
	assert.NotNil(t, ErrURLMissing)
	assert.NotNil(t, ErrConfigRequired)
	assert.NotNil(t, ErrTimeoutOutOfRange)
}
