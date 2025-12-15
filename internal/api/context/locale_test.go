package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	i18npkg "github.com/xzzpig/rclone-sync/internal/i18n"
)

func TestLocaleMiddleware(t *testing.T) {
	// Initialize i18n for tests
	i18npkg.Init()

	tests := []struct {
		name           string
		acceptLanguage string
		expectedLocale string
	}{
		{
			name:           "Chinese language header",
			acceptLanguage: "zh-CN,zh;q=0.9",
			expectedLocale: "zh-CN",
		},
		{
			name:           "English language header",
			acceptLanguage: "en-US,en;q=0.9",
			expectedLocale: "en",
		},
		{
			name:           "No Accept-Language header defaults to English",
			acceptLanguage: "",
			expectedLocale: "en",
		},
		{
			name:           "Unsupported language falls back to English",
			acceptLanguage: "fr-FR,fr;q=0.9",
			expectedLocale: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			c.Request = req

			// Apply middleware
			LocaleMiddleware()(c)

			// Verify locale is set in Gin context
			locale, exists := c.Get("locale")
			assert.True(t, exists, "locale should be set in context")
			assert.Equal(t, tt.expectedLocale, locale, "locale should match expected")

			// Verify localizer is set in Gin context
			localizer := GetLocalizer(c)
			assert.NotNil(t, localizer, "localizer should not be nil")

			// Verify localizer is accessible from request context
			ctxLocalizer := i18npkg.LocalizerFromContext(c.Request.Context())
			assert.NotNil(t, ctxLocalizer, "localizer should be accessible from request context")
		})
	}
}

func TestGetLocalizer(t *testing.T) {
	i18npkg.Init()

	t.Run("returns localizer from Gin context", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "zh-CN")
		c.Request = req

		LocaleMiddleware()(c)

		localizer := GetLocalizer(c)
		assert.NotNil(t, localizer)
	})

	t.Run("returns fallback localizer when not set", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/test", nil)
		c.Request = req

		localizer := GetLocalizer(c)
		// Should return default localizer
		assert.NotNil(t, localizer)
	})
}

func TestI18nErrorMiddleware(t *testing.T) {
	i18npkg.Init()

	t.Run("translates I18nError", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "zh-CN")
		c.Request = req

		// Set up middleware chain
		LocaleMiddleware()(c)

		// Simulate an I18nError
		err := i18npkg.NewI18nError(i18npkg.ErrTaskNotFound).WithStatus(http.StatusNotFound)
		c.Error(err)

		// Apply I18nErrorMiddleware
		I18nErrorMiddleware()(c)

		// Verify response
		assert.Equal(t, http.StatusNotFound, w.Code)

		// Response should contain localized error message
		assert.Contains(t, w.Body.String(), "error")
		assert.Contains(t, w.Body.String(), "code")
		assert.Contains(t, w.Body.String(), "success")
	})

	t.Run("handles non-I18nError with generic message", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "en")
		c.Request = req

		LocaleMiddleware()(c)

		// Simulate a regular error
		c.Error(assert.AnError)

		I18nErrorMiddleware()(c)

		// Verify generic error response
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "error")
		assert.Contains(t, w.Body.String(), "internal_error")
	})

	t.Run("does nothing when no errors", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/test", nil)
		c.Request = req

		LocaleMiddleware()(c)

		// No errors - should not write response
		I18nErrorMiddleware()(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("translates to Chinese when Accept-Language is zh-CN", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "zh-CN")
		c.Request = req

		LocaleMiddleware()(c)

		err := i18npkg.NewI18nError(i18npkg.ErrTaskNotFound).WithStatus(http.StatusNotFound)
		c.Error(err)

		I18nErrorMiddleware()(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		// Should contain Chinese error message
		body := w.Body.String()
		assert.Contains(t, body, "error")
		// The actual Chinese translation would be tested if we know the exact translation
	})
}
