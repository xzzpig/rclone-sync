package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apicontext "github.com/xzzpig/rclone-sync/internal/api/context"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

func TestMain(m *testing.M) {
	// Initialize i18n before running tests
	if err := i18n.Init(); err != nil {
		panic(fmt.Sprintf("Failed to initialize i18n: %v", err))
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		want     string
	}{
		{
			name: "standard error",
			appError: &AppError{
				Code:    http.StatusNotFound,
				Message: "Resource Not Found",
				Details: "user not found",
			},
			want: "404: Resource Not Found",
		},
		{
			name: "internal server error",
			appError: &AppError{
				Code:    http.StatusInternalServerError,
				Message: "Internal Server Error",
				Details: "",
			},
			want: "500: Internal Server Error",
		},
		{
			name: "bad request error",
			appError: &AppError{
				Code:    http.StatusBadRequest,
				Message: "Invalid Input",
				Details: "missing required field",
			},
			want: "400: Invalid Input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.appError.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewError(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		message string
		details string
	}{
		{
			name:    "with details",
			code:    http.StatusNotFound,
			message: "Resource Not Found",
			details: "item not found",
		},
		{
			name:    "without details",
			code:    http.StatusBadRequest,
			message: "Bad Request",
			details: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.code, tt.message, tt.details)

			assert.NotNil(t, err)
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.details, err.Details)

			// Verify it implements error interface
			assert.Contains(t, err.Error(), tt.message)
			assert.Contains(t, err.Error(), fmt.Sprintf("%d", tt.code))
		})
	}
}

func TestHandleError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   int
		expectedMsg    string
	}{
		{
			name: "AppError type",
			err: &AppError{
				Code:    http.StatusBadRequest,
				Message: "Bad Request",
				Details: "invalid parameter",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   http.StatusBadRequest,
			expectedMsg:    "Bad Request",
		},
		{
			name:           "ErrNotFound",
			err:            errs.ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   http.StatusNotFound,
			expectedMsg:    "Resource not found",
		},
		{
			name:           "wrapped ErrNotFound",
			err:            fmt.Errorf("task: %w", errs.ErrNotFound),
			expectedStatus: http.StatusNotFound,
			expectedCode:   http.StatusNotFound,
			expectedMsg:    "Resource not found",
		},
		{
			name:           "ErrAlreadyExists",
			err:            errs.ErrAlreadyExists,
			expectedStatus: http.StatusConflict,
			expectedCode:   http.StatusConflict,
			expectedMsg:    "Resource already exists",
		},
		{
			name:           "wrapped ErrAlreadyExists",
			err:            fmt.Errorf("duplicate: %w", errs.ErrAlreadyExists),
			expectedStatus: http.StatusConflict,
			expectedCode:   http.StatusConflict,
			expectedMsg:    "Resource already exists",
		},
		{
			name:           "ErrInvalidInput",
			err:            errs.ErrInvalidInput,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   http.StatusBadRequest,
			expectedMsg:    "Invalid input provided",
		},
		{
			name:           "wrapped ErrInvalidInput",
			err:            fmt.Errorf("validation failed: %w", errs.ErrInvalidInput),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   http.StatusBadRequest,
			expectedMsg:    "Invalid input provided",
		},
		{
			name:           "ErrUnauthorized",
			err:            errs.ErrUnauthorized,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   http.StatusUnauthorized,
			expectedMsg:    "Unauthorized access",
		},
		{
			name:           "wrapped ErrUnauthorized",
			err:            fmt.Errorf("access denied: %w", errs.ErrUnauthorized),
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   http.StatusUnauthorized,
			expectedMsg:    "Unauthorized access",
		},
		{
			name:           "generic error",
			err:            errors.New("something went wrong"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   http.StatusInternalServerError,
			expectedMsg:    "An error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(apicontext.LocaleMiddleware())
			router.GET("/test", func(c *gin.Context) {
				HandleError(c, tt.err)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Language", "en")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			var appErr AppError
			err := json.NewDecoder(resp.Body).Decode(&appErr)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedCode, appErr.Code)
			assert.Equal(t, tt.expectedMsg, appErr.Message)
			assert.NotEmpty(t, appErr.Details)
		})
	}
}

func TestNotFoundHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(apicontext.LocaleMiddleware())
	router.NoRoute(NotFoundHandler)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req.Header.Set("Accept-Language", "en")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)

	var appErr AppError
	err := json.NewDecoder(resp.Body).Decode(&appErr)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, appErr.Code)
	assert.Equal(t, "Resource not found", appErr.Message)
	assert.Empty(t, appErr.Details)
}

func TestHandleError_JSONFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(apicontext.LocaleMiddleware())
	router.GET("/test", func(c *gin.Context) {
		HandleError(c, NewError(http.StatusBadRequest, "Test Message", "test details"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Language", "en")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	// Verify JSON structure
	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, float64(http.StatusBadRequest), result["code"])
	assert.Equal(t, "Test Message", result["message"])
	assert.Equal(t, "test details", result["details"])
}

func TestHandleError_AppErrorWithoutDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(apicontext.LocaleMiddleware())
	router.GET("/test", func(c *gin.Context) {
		HandleError(c, NewError(http.StatusNotFound, "Not Found", ""))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Language", "en")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)

	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// When details is empty, it should still be present but empty (due to omitempty in struct)
	// Actually with omitempty, empty string won't be in JSON
	_, hasDetails := result["details"]
	assert.False(t, hasDetails, "details field should be omitted when empty")
}
