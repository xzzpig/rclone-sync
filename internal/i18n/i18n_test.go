package i18n

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	err := Init()
	require.NoError(t, err, "Init should not return error")
	assert.NotNil(t, bundle, "bundle should be initialized")
}

func TestParseLocale(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Chinese with region", "zh-CN", "zh-CN"},
		{"Chinese without region", "zh", "zh-CN"},
		{"Chinese with other region", "zh-TW", "zh-CN"},
		{"English", "en", "en"},
		{"English with region", "en-US", "en"},
		{"Other language defaults to en", "fr", "en"},
		{"Empty string defaults to en", "", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLocale(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTranslationFunctions(t *testing.T) {
	// Initialize bundle first
	err := Init()
	require.NoError(t, err)

	t.Run("T function with English", func(t *testing.T) {
		localizer := NewLocalizer("en")
		msg := T(localizer, ErrGeneric)
		assert.Equal(t, "An error occurred", msg)
	})

	t.Run("T function with Chinese", func(t *testing.T) {
		localizer := NewLocalizer("zh-CN")
		msg := T(localizer, ErrGeneric)
		assert.Equal(t, "发生错误", msg)
	})

	t.Run("TWithData function", func(t *testing.T) {
		localizer := NewLocalizer("en")
		msg := TWithData(localizer, ErrConnectionFailed, map[string]interface{}{
			"Reason": "network timeout",
		})
		assert.Contains(t, msg, "Connection failed")
		assert.Contains(t, msg, "network timeout")
	})

	t.Run("TPlural function", func(t *testing.T) {
		localizer := NewLocalizer("en")

		// Single file
		msg1 := TPlural(localizer, StatusSyncingFiles, 1, nil)
		assert.Contains(t, msg1, "1 file")

		// Multiple files
		msg5 := TPlural(localizer, StatusSyncingFiles, 5, nil)
		assert.Contains(t, msg5, "5 files")
	})
}

func TestContextFunctions(t *testing.T) {
	err := Init()
	require.NoError(t, err)

	t.Run("WithLocalizer and LocalizerFromContext", func(t *testing.T) {
		localizer := NewLocalizer("zh-CN")
		ctx := WithLocalizer(context.Background(), localizer)

		retrieved := LocalizerFromContext(ctx)
		assert.NotNil(t, retrieved)

		// Test that it's the same localizer by translating
		msg := T(retrieved, ErrGeneric)
		assert.Equal(t, "发生错误", msg)
	})

	t.Run("WithLocale and LocaleFromContext", func(t *testing.T) {
		ctx := WithLocale(context.Background(), "zh-CN")
		locale := LocaleFromContext(ctx)
		assert.Equal(t, "zh-CN", locale)
	})

	t.Run("LocalizerFromContext with empty context", func(t *testing.T) {
		localizer := LocalizerFromContext(context.Background())
		assert.NotNil(t, localizer)

		// Should default to English
		msg := T(localizer, ErrGeneric)
		assert.Equal(t, "An error occurred", msg)
	})

	t.Run("LocaleFromContext with empty context", func(t *testing.T) {
		locale := LocaleFromContext(context.Background())
		assert.Equal(t, "en", locale)
	})

	t.Run("Ctx convenience function", func(t *testing.T) {
		localizer := NewLocalizer("zh-CN")
		ctx := WithLocalizer(context.Background(), localizer)

		msg := Ctx(ctx, ErrNotFound)
		assert.Equal(t, "资源不存在", msg)
	})

	t.Run("CtxWithData convenience function", func(t *testing.T) {
		localizer := NewLocalizer("en")
		ctx := WithLocalizer(context.Background(), localizer)

		msg := CtxWithData(ctx, ErrConnectionFailed, map[string]interface{}{
			"Reason": "timeout",
		})
		assert.Contains(t, msg, "timeout")
	})

	t.Run("CtxPlural convenience function", func(t *testing.T) {
		localizer := NewLocalizer("zh-CN")
		ctx := WithLocalizer(context.Background(), localizer)

		msg := CtxPlural(ctx, StatusSyncingFiles, 3, nil)
		assert.Contains(t, msg, "3")
		assert.Contains(t, msg, "文件")
	})
}

func TestI18nError(t *testing.T) {
	err := Init()
	require.NoError(t, err)

	t.Run("NewI18nError", func(t *testing.T) {
		i18nErr := NewI18nError(ErrNotFound)
		assert.Equal(t, ErrNotFound, i18nErr.MsgID)
		assert.Equal(t, 400, i18nErr.StatusCode)
		assert.Nil(t, i18nErr.Data)
		assert.Nil(t, i18nErr.Cause)
	})

	t.Run("NewI18nErrorWithData", func(t *testing.T) {
		data := map[string]interface{}{"Field": "name"}
		i18nErr := NewI18nErrorWithData(ErrValidationFailed, data)
		assert.Equal(t, ErrValidationFailed, i18nErr.MsgID)
		assert.Equal(t, data, i18nErr.Data)
	})

	t.Run("Error method", func(t *testing.T) {
		i18nErr := NewI18nError(ErrNotFound)
		assert.Equal(t, ErrNotFound, i18nErr.Error())
	})

	t.Run("Error method with cause", func(t *testing.T) {
		cause := assert.AnError
		i18nErr := NewI18nError(ErrDatabaseError).WithCause(cause)
		assert.Contains(t, i18nErr.Error(), ErrDatabaseError)
		assert.Contains(t, i18nErr.Error(), cause.Error())
	})

	t.Run("Translate method", func(t *testing.T) {
		localizer := NewLocalizer("en")
		i18nErr := NewI18nError(ErrTaskNotFound)
		msg := i18nErr.Translate(localizer)
		assert.Equal(t, "Task not found", msg)
	})

	t.Run("TranslateCtx method", func(t *testing.T) {
		localizer := NewLocalizer("zh-CN")
		ctx := WithLocalizer(context.Background(), localizer)
		i18nErr := NewI18nError(ErrTaskNotFound)
		msg := i18nErr.TranslateCtx(ctx)
		assert.Equal(t, "任务未找到", msg)
	})

	t.Run("WithStatus", func(t *testing.T) {
		i18nErr := NewI18nError(ErrNotFound).WithStatus(404)
		assert.Equal(t, 404, i18nErr.StatusCode)
	})

	t.Run("WithCause", func(t *testing.T) {
		cause := assert.AnError
		i18nErr := NewI18nError(ErrDatabaseError).WithCause(cause)
		assert.Equal(t, cause, i18nErr.Cause)
		assert.Equal(t, cause, i18nErr.Unwrap())
	})

	t.Run("WithData", func(t *testing.T) {
		data := map[string]interface{}{"Key": "value"}
		i18nErr := NewI18nError(ErrValidationFailed).WithData(data)
		assert.Equal(t, data, i18nErr.Data)
	})

	t.Run("Helper constructors", func(t *testing.T) {
		assert.Equal(t, 404, ErrNotFoundI18n(ErrNotFound).StatusCode)
		assert.Equal(t, 400, ErrBadRequestI18n(ErrValidationFailed).StatusCode)
		assert.Equal(t, 500, ErrInternalI18n(ErrDatabaseError).StatusCode)
		assert.Equal(t, 401, ErrUnauthorizedI18n(ErrUnauthorized).StatusCode)
	})

	t.Run("IsI18nError", func(t *testing.T) {
		i18nErr := NewI18nError(ErrNotFound)

		// Test with I18nError
		extracted, ok := IsI18nError(i18nErr)
		assert.True(t, ok)
		assert.Equal(t, i18nErr, extracted)

		// Test with regular error
		regularErr := assert.AnError
		extracted, ok = IsI18nError(regularErr)
		assert.False(t, ok)
		assert.Nil(t, extracted)
	})
}

func TestTranslationFallback(t *testing.T) {
	err := Init()
	require.NoError(t, err)

	t.Run("Missing translation key fallback", func(t *testing.T) {
		localizer := NewLocalizer("en")
		nonExistentKey := "non_existent_key_12345"

		// Should return the key itself as fallback
		msg := T(localizer, nonExistentKey)
		assert.Equal(t, nonExistentKey, msg, "Should fallback to key when translation is missing")
	})

	t.Run("Missing translation in Chinese, fallback to English", func(t *testing.T) {
		// Create a key that exists in English but not in Chinese
		// Since all our keys exist in both, we'll test with a non-existent key
		localizer := NewLocalizer("zh-CN")
		nonExistentKey := "test_missing_key"

		msg := T(localizer, nonExistentKey)
		assert.Equal(t, nonExistentKey, msg, "Should fallback to key when translation is missing in Chinese")
	})

	t.Run("TWithData fallback", func(t *testing.T) {
		localizer := NewLocalizer("en")
		nonExistentKey := "non_existent_template_key"

		msg := TWithData(localizer, nonExistentKey, map[string]interface{}{
			"Data": "value",
		})
		assert.Equal(t, nonExistentKey, msg, "Should fallback to key when template translation is missing")
	})

	t.Run("TPlural fallback", func(t *testing.T) {
		localizer := NewLocalizer("en")
		nonExistentKey := "non_existent_plural_key"

		msg := TPlural(localizer, nonExistentKey, 5, nil)
		assert.Equal(t, nonExistentKey, msg, "Should fallback to key when plural translation is missing")
	})

	t.Run("Context function fallback", func(t *testing.T) {
		localizer := NewLocalizer("en")
		ctx := WithLocalizer(context.Background(), localizer)
		nonExistentKey := "ctx_missing_key"

		msg := Ctx(ctx, nonExistentKey)
		assert.Equal(t, nonExistentKey, msg, "Ctx should fallback to key when translation is missing")
	})

	t.Run("I18nError translation fallback", func(t *testing.T) {
		localizer := NewLocalizer("en")
		nonExistentKey := "error_missing_key"
		i18nErr := NewI18nError(nonExistentKey)

		msg := i18nErr.Translate(localizer)
		assert.Equal(t, nonExistentKey, msg, "I18nError should fallback to key when translation is missing")
	})
}
